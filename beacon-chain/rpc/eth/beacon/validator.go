package beacon

import (
	"context"
	"encoding/hex"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corehelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	statenative "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/network"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// invalidValidatorIdError represents an error scenario where a validator's ID is invalid.
type invalidValidatorIdError struct {
	message string
}

// newInvalidValidatorIdError creates a new error instance.
func newInvalidValidatorIdError(validatorId []byte, reason error) invalidValidatorIdError {
	return invalidValidatorIdError{
		message: errors.Wrapf(reason, "could not decode validator id '%s'", string(validatorId)).Error(),
	}
}

// Error returns the underlying error message.
func (e *invalidValidatorIdError) Error() string {
	return e.message
}

// GetValidator returns a validator specified by state and id or public key along with status and balance.
func (bs *Server) GetValidator(ctx context.Context, req *ethpb.StateValidatorRequest) (*ethpb.StateValidatorResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetValidator")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}
	if len(req.ValidatorId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Validator ID is required")
	}
	valContainer, err := valContainersByRequestIds(st, [][]byte{req.ValidatorId})
	if err != nil {
		return nil, handleValContainerErr(err)
	}
	if len(valContainer) == 0 {
		return nil, status.Error(codes.NotFound, "Could not find validator")
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	return &ethpb.StateValidatorResponse{Data: valContainer[0], ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}

// ListValidators returns filterable list of validators with their balance, status and index.
func (bs *Server) ListValidators(ctx context.Context, req *ethpb.StateValidatorsRequest) (*ethpb.StateValidatorsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListValidators")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	valContainers, err := valContainersByRequestIds(st, req.Id)
	if err != nil {
		return nil, handleValContainerErr(err)
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	// Exit early if no matching validators we found or we don't want to further filter validators by status.
	if len(valContainers) == 0 || len(req.Status) == 0 {
		return &ethpb.StateValidatorsResponse{Data: valContainers, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
	}

	filterStatus := make(map[ethpb.ValidatorStatus]bool, len(req.Status))
	const lastValidStatusValue = ethpb.ValidatorStatus(12)
	for _, ss := range req.Status {
		if ss > lastValidStatusValue {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid status "+ss.String())
		}
		filterStatus[ss] = true
	}
	epoch := slots.ToEpoch(st.Slot())
	filteredVals := make([]*ethpb.ValidatorContainer, 0, len(valContainers))
	for _, vc := range valContainers {
		readOnlyVal, err := statenative.NewValidator(migration.V1ValidatorToV1Alpha1(vc.Validator))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert validator: %v", err)
		}
		valStatus, err := helpers.ValidatorStatus(readOnlyVal, epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator status: %v", err)
		}
		valSubStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator sub status: %v", err)
		}
		if filterStatus[valStatus] || filterStatus[valSubStatus] {
			filteredVals = append(filteredVals, vc)
		}
	}

	return &ethpb.StateValidatorsResponse{Data: filteredVals, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}

// ListValidatorBalances returns a filterable list of validator balances.
func (bs *Server) ListValidatorBalances(ctx context.Context, req *ethpb.ValidatorBalancesRequest) (*ethpb.ValidatorBalancesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListValidatorBalances")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	valContainers, err := valContainersByRequestIds(st, req.Id)
	if err != nil {
		return nil, handleValContainerErr(err)
	}
	valBalances := make([]*ethpb.ValidatorBalance, len(valContainers))
	for i := 0; i < len(valContainers); i++ {
		valBalances[i] = &ethpb.ValidatorBalance{
			Index:   valContainers[i].Index,
			Balance: valContainers[i].Balance,
		}
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	return &ethpb.ValidatorBalancesResponse{Data: valBalances, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}

// ListCommittees retrieves the committees for the given state at the given epoch.
// If the requested slot and index are defined, only those committees are returned.
func (bs *Server) ListCommittees(ctx context.Context, req *ethpb.StateCommitteesRequest) (*ethpb.StateCommitteesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListCommittees")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	epoch := slots.ToEpoch(st.Slot())
	if req.Epoch != nil {
		epoch = *req.Epoch
	}
	activeCount, err := corehelpers.ActiveValidatorCount(ctx, st, epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}

	startSlot, err := slots.EpochStart(epoch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid epoch: %v", err)
	}
	endSlot, err := slots.EpochEnd(epoch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid epoch: %v", err)
	}
	committeesPerSlot := corehelpers.SlotCommitteeCount(activeCount)
	committees := make([]*ethpb.Committee, 0)
	for slot := startSlot; slot <= endSlot; slot++ {
		if req.Slot != nil && slot != *req.Slot {
			continue
		}
		for index := primitives.CommitteeIndex(0); index < primitives.CommitteeIndex(committeesPerSlot); index++ {
			if req.Index != nil && index != *req.Index {
				continue
			}
			committee, err := corehelpers.BeaconCommitteeFromState(ctx, st, slot, index)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get committee: %v", err)
			}
			committeeContainer := &ethpb.Committee{
				Index:      index,
				Slot:       slot,
				Validators: committee,
			}
			committees = append(committees, committeeContainer)
		}
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	return &ethpb.StateCommitteesResponse{Data: committees, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}

// This function returns the validator object based on the passed in ID. The validator ID could be its public key,
// or its index.
func valContainersByRequestIds(state state.BeaconState, validatorIds [][]byte) ([]*ethpb.ValidatorContainer, error) {
	epoch := slots.ToEpoch(state.Slot())
	var valContainers []*ethpb.ValidatorContainer
	allBalances := state.Balances()
	if len(validatorIds) == 0 {
		allValidators := state.Validators()
		valContainers = make([]*ethpb.ValidatorContainer, len(allValidators))
		for i, validator := range allValidators {
			readOnlyVal, err := statenative.NewValidator(validator)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not convert validator: %v", err)
			}
			subStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
			if err != nil {
				return nil, errors.Wrap(err, "could not get validator sub status")
			}
			valContainers[i] = &ethpb.ValidatorContainer{
				Index:     primitives.ValidatorIndex(i),
				Balance:   allBalances[i],
				Status:    subStatus,
				Validator: migration.V1Alpha1ValidatorToV1(validator),
			}
		}
	} else {
		valContainers = make([]*ethpb.ValidatorContainer, 0, len(validatorIds))
		for _, validatorId := range validatorIds {
			var valIndex primitives.ValidatorIndex
			if len(validatorId) == params.BeaconConfig().BLSPubkeyLength {
				var ok bool
				valIndex, ok = state.ValidatorIndexByPubkey(bytesutil.ToBytes48(validatorId))
				if !ok {
					// Ignore well-formed yet unknown public keys.
					continue
				}
			} else {
				index, err := strconv.ParseUint(string(validatorId), 10, 64)
				if err != nil {
					e := newInvalidValidatorIdError(validatorId, err)
					return nil, &e
				}
				valIndex = primitives.ValidatorIndex(index)
			}
			validator, err := state.ValidatorAtIndex(valIndex)
			if _, ok := err.(*statenative.ValidatorIndexOutOfRangeError); ok {
				// Ignore well-formed yet unknown indexes.
				continue
			}
			if err != nil {
				return nil, errors.Wrap(err, "could not get validator")
			}
			v1Validator := migration.V1Alpha1ValidatorToV1(validator)
			readOnlyVal, err := statenative.NewValidator(validator)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not convert validator: %v", err)
			}
			subStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
			if err != nil {
				return nil, errors.Wrap(err, "could not get validator sub status")
			}
			valContainers = append(valContainers, &ethpb.ValidatorContainer{
				Index:     valIndex,
				Balance:   allBalances[valIndex],
				Status:    subStatus,
				Validator: v1Validator,
			})
		}
	}

	return valContainers, nil
}

func handleValContainerErr(err error) error {
	if outOfRangeErr, ok := err.(*statenative.ValidatorIndexOutOfRangeError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid validator ID: %v", outOfRangeErr)
	}
	if invalidIdErr, ok := err.(*invalidValidatorIdError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid validator ID: %v", invalidIdErr)
	}
	return status.Errorf(codes.Internal, "Could not get validator container: %v", err)
}

type ValidatorEstimatedActivationResponse struct {
	WaitingEpoch  uint64 `json:"waiting_epoch"`
	EligibleEpoch uint64 `json:"eligible_epoch"`
	Status        uint64 `json:"status"`
}

type Validator struct {
	Index                      uint64
	PublicKey                  []byte
	ActivationEligibilityEpoch uint64
	ActivationEpoch            uint64
}

// EstimatedActivation returns estimated activation timing for depositing validators.
func (bs *Server) EstimatedActivation(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(r.URL.Path, "/")
	pubKey := segments[len(segments)-1]
	if pubKey == "estimated_activation" {
		pubKey = ""
	} else if !is96CharHex(pubKey) {
		handleHTTPError(w, "this is not a proper BLS public key : "+pubKey, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	st, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		handleHTTPError(w, "could not get head state : "+err.Error(), http.StatusBadRequest)
		return
	}

	headSlot := st.Slot()
	epoch := slots.ToEpoch(headSlot)
	allValidators := st.Validators()
	lastActiveIdx, activeValCount := 0, uint64(0)
	activationQ := make([]Validator, 0)
	waitingEpoch, eligibleEpoch, status := uint64(0), uint64(0), uint64(0)
	for i, validator := range allValidators {
		readOnlyVal, err := statenative.NewValidator(validator)
		if err != nil {
			handleHTTPError(w, "could not convert validator: "+err.Error(), http.StatusBadRequest)
			return
		}

		status, err := helpers.ValidatorStatus(readOnlyVal, epoch)
		if err != nil {
			handleHTTPError(w, "could not get validator sub status: "+err.Error(), http.StatusBadRequest)
			return
		}
		if status == ethpb.ValidatorStatus_ACTIVE {
			lastActiveIdx = i
			activeValCount++
			if hex.EncodeToString(validator.GetPublicKey()) == pubKey {
				response := &ValidatorEstimatedActivationResponse{
					WaitingEpoch:  uint64(0),
					EligibleEpoch: uint64(validator.ActivationEligibilityEpoch),
					Status:        3,
				}
				network.WriteJson(w, response)
				return
			}
			continue
		}

		subStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
		if err != nil {
			handleHTTPError(w, "could not get validator sub status: "+err.Error(), http.StatusBadRequest)
			return
		}
		if subStatus == ethpb.ValidatorStatus_PENDING_QUEUED {
			activationQ = append(activationQ, Validator{
				Index:                      uint64(i),
				PublicKey:                  validator.GetPublicKey(),
				ActivationEligibilityEpoch: uint64(validator.GetActivationEligibilityEpoch()),
				ActivationEpoch:            uint64(validator.GetActivationEpoch()),
			})
		}
	}

	activationsPerEpoch := uint64(math.Max(float64(params.BeaconConfig().MinPerEpochChurnLimit), float64(activeValCount/params.BeaconConfig().ChurnLimitQuotient)))

	if len(pubKey) == 0 {
		if len(activationQ) == 0 {
			waitingEpoch = uint64(0)
		} else {
			waitingEpoch = (uint64(len(activationQ))+activationsPerEpoch)/activationsPerEpoch + uint64(params.BeaconConfig().MaxSeedLookahead)
		}
		eligibleEpoch = calculateEligibleEpoch(headSlot)
	} else {
		if len(activationQ) == 0 {
			waitingEpoch = uint64(0)
			eligibleEpoch = calculateEligibleEpoch(headSlot)
		} else {
			for _, val := range activationQ {
				if pubKey == hex.EncodeToString(val.PublicKey) {
					eligibleEpoch = val.ActivationEligibilityEpoch
					if val.ActivationEpoch < uint64(params.BeaconConfig().FarFutureEpoch) {
						waitingEpoch = val.ActivationEpoch - uint64(slots.ToEpoch(headSlot))
						status = 2
					} else {
						pendingQueueSize := val.Index - uint64(lastActiveIdx)
						waitingEpoch = uint64((pendingQueueSize+activationsPerEpoch)/activationsPerEpoch + uint64(params.BeaconConfig().MaxSeedLookahead))
						status = 1
					}
					break
				}
			}
			if status == 0 {
				waitingEpoch = (uint64(len(activationQ))+activationsPerEpoch)/activationsPerEpoch + uint64(params.BeaconConfig().MaxSeedLookahead)
				eligibleEpoch = calculateEligibleEpoch(headSlot)
			}
		}
	}

	response := &ValidatorEstimatedActivationResponse{
		WaitingEpoch:  waitingEpoch,
		EligibleEpoch: eligibleEpoch,
		Status:        status,
	}
	network.WriteJson(w, response)
}

func calculateEligibleEpoch(headSlot primitives.Slot) uint64 {
	epochsPerEth1VotingPeriod := params.BeaconConfig().EpochsPerEth1VotingPeriod

	currentEpoch := slots.ToEpoch(headSlot)
	currentPeriodStartEpoch := currentEpoch - currentEpoch.Mod(uint64(epochsPerEth1VotingPeriod))
	midEpochInThisPeriod := currentPeriodStartEpoch + epochsPerEth1VotingPeriod/2
	if currentEpoch <= midEpochInThisPeriod {
		return uint64(currentPeriodStartEpoch.Add(uint64(epochsPerEth1VotingPeriod))+epochsPerEth1VotingPeriod/2) + 1
	} else {
		return uint64(currentPeriodStartEpoch.Add(uint64(epochsPerEth1VotingPeriod.Mul(2)))+epochsPerEth1VotingPeriod/2) + 1
	}
}

func handleHTTPError(w http.ResponseWriter, message string, code int) {
	errJson := &network.DefaultErrorJson{
		Message: message,
		Code:    code,
	}
	network.WriteError(w, errJson)
}

func is96CharHex(s string) bool {
	if len(s) != 96 {
		return false
	}
	match, err := regexp.MatchString("^[a-fA-F0-9]{96}$", s)
	if err != nil {
		return false
	}

	return match
}
