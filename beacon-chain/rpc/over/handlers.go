package over

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	valhelpers "github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/math"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"go.opencensus.io/trace"
)

// EstimatedActivation returns EstimatedActivationResponse for given validator id.
// This will handle request generously: calculate the estimation for empty request.
// Status is related to life cycle of deposited validator like following:
// Status 0: beacon chain isn't aware of its deposit
// Status 1: beacon chain knows its deposit, but the validator doesn't have ActivationEpoch explicitly
// Status 2: the validator has ActivationEpoch explicitly
// Status 3: the validator has been already activated
func (s *Server) EstimatedActivation(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "over.EstimatedActivation")
	defer span.End()

	st, err := s.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not retrieve head state", http.StatusBadRequest))
		return
	}

	rawId := r.PathValue("validator_id")
	valIndex, err := decodeValidatorId(st, rawId)
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not decode validator id from raw id", http.StatusBadRequest))
		return
	}

	headSlot := st.Slot()
	epoch := slots.ToEpoch(headSlot)
	validators := st.ValidatorsReadOnly()
	lastActiveIdx, pendingQueuedCount, activeCount := uint64(0), uint64(0), uint64(0)

	for i, val := range validators {
		valSubStatus, err := valhelpers.ValidatorSubStatus(val, epoch)
		if err != nil {
			httputil.WriteError(w, handleWrapError(err, "could not get validator sub status", http.StatusBadRequest))
			return
		}

		switch valSubStatus {
		case validator.PendingQueued:
			pendingQueuedCount++

			// Fast path
			if primitives.ValidatorIndex(i) == valIndex {
				if val.ActivationEpoch() == params.BeaconConfig().FarFutureEpoch {
					pendingQueuedCount = uint64(valIndex) - lastActiveIdx
					httputil.WriteJson(w, &structs.EstimatedActivationResponse{
						WaitingEpoch:  calculateWaitingEpoch(activeCount, pendingQueuedCount),
						EligibleEpoch: uint64(val.ActivationEligibilityEpoch()),
						Status:        1,
					})
				} else {
					httputil.WriteJson(w, &structs.EstimatedActivationResponse{
						WaitingEpoch:  uint64(val.ActivationEpoch()) - uint64(epoch),
						EligibleEpoch: uint64(val.ActivationEligibilityEpoch()),
						Status:        2,
					})
				}
				return
			}
		case validator.ActiveOngoing, validator.ActiveSlashed, validator.ActiveExiting:
			activeCount++
			lastActiveIdx = uint64(i)

			// Fast path
			if primitives.ValidatorIndex(i) == valIndex {
				httputil.WriteJson(w, &structs.EstimatedActivationResponse{
					WaitingEpoch:  uint64(0),
					EligibleEpoch: uint64(val.ActivationEligibilityEpoch()),
					Status:        3,
				})
				return
			}
		}
	}

	// If validator is not found, it will return an estimation based on current state
	// when new deposit is included. (Status = 0)
	status := uint64(0)
	eligibleEpoch := calculateEligibleEpoch(headSlot)

	if pendingQueuedCount == 0 {
		httputil.WriteJson(w, &structs.EstimatedActivationResponse{
			WaitingEpoch:  uint64(0),
			EligibleEpoch: eligibleEpoch,
			Status:        status,
		})
		return
	}

	httputil.WriteJson(w, &structs.EstimatedActivationResponse{
		WaitingEpoch:  calculateWaitingEpoch(activeCount, pendingQueuedCount),
		EligibleEpoch: eligibleEpoch,
		Status:        status,
	})
}

func calculateEligibleEpoch(headSlot primitives.Slot) uint64 {
	epochsPerEth1VotingPeriod := params.BeaconConfig().EpochsPerEth1VotingPeriod

	currentEpoch := slots.ToEpoch(headSlot)
	currentPeriodStartEpoch := currentEpoch - currentEpoch.Mod(uint64(epochsPerEth1VotingPeriod))
	midEpochInThisPeriod := currentPeriodStartEpoch + epochsPerEth1VotingPeriod/2
	if currentEpoch < midEpochInThisPeriod {
		return uint64(currentPeriodStartEpoch.Add(uint64(epochsPerEth1VotingPeriod))+epochsPerEth1VotingPeriod/2) + 1
	} else {
		return uint64(currentPeriodStartEpoch.Add(uint64(epochsPerEth1VotingPeriod.Mul(2)))+epochsPerEth1VotingPeriod/2) + 1
	}
}

// GetEpochReward returns total reward at given epoch.
func (s *Server) GetEpochReward(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "over.GetEpochReward")
	defer span.End()

	var requestedEpoch primitives.Epoch
	epochId := r.PathValue("epoch")
	curEpoch := slots.ToEpoch(s.GenesisTimeFetcher.CurrentSlot())

	if epochId == "latest" {
		requestedEpoch = curEpoch
	} else {
		uintEpoch, err := strconv.ParseUint(epochId, 10, 64)
		if err != nil {
			httputil.HandleError(w, "Could not parse uint: "+err.Error(), http.StatusBadRequest)
			return
		}
		if uintEpoch > uint64(curEpoch) {
			httputil.HandleError(w, fmt.Sprintf("Cannot retrieve information for an future epoch, current epoch %d, requesting %d", curEpoch, uintEpoch), http.StatusBadRequest)
			return
		}
		requestedEpoch = primitives.Epoch(uintEpoch)
	}

	slot, err := slots.EpochStart(requestedEpoch)
	if err != nil {
		httputil.HandleError(w, "Could not get start slot of requested : "+err.Error(), http.StatusBadRequest)
		return
	}

	reqState, err := s.Stater.StateBySlot(ctx, slot)
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not retrieve state", http.StatusNotFound))
		return
	}

	reward, _ := helpers.TotalRewardWithReserveUsage(reqState)

	httputil.WriteJson(w, &structs.EpochReward{
		Reward: strconv.FormatUint(reward, 10),
	})
}

// GetReserves get reserves data from the requested state.
// e.g. RewardAdjustmentFactor, PreviousEpochReserve, CurrentEpochReserve, etc.
func (s *Server) GetReserves(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "over.GetReserves")
	defer span.End()

	// Retrieve beacon state
	stateId := r.PathValue("state_id")
	if stateId == "" {
		httputil.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}
	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not retrieve state", http.StatusNotFound))
		return
	}

	// Get metadata for response
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not get optimistic mode info", http.StatusInternalServerError))
		return
	}
	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		httputil.HandleError(w, errors.Wrap(err, "Could not calculate root of latest block header: ").Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	rewardAdjustmentFactor := st.RewardAdjustmentFactor()
	previousEpochReserve := st.PreviousEpochReserve()
	currentEpochReserve := st.CurrentEpochReserve()

	httputil.WriteJson(w, &structs.GetReservesResponse{
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
		Data: &structs.Reserves{
			RewardAdjustmentFactor: strconv.FormatUint(rewardAdjustmentFactor, 10),
			PreviousEpochReserve:   strconv.FormatUint(previousEpochReserve, 10),
			CurrentEpochReserve:    strconv.FormatUint(currentEpochReserve, 10),
		},
	})
}

func handleWrapError(err error, message string, code int) *httputil.DefaultJsonError {
	return &httputil.DefaultJsonError{
		Message: errors.Wrapf(err, message).Error(),
		Code:    code,
	}
}

// decodeValidatorId takes a raw validator ID(rawId) string (as either a pubkey or a validator index)
// and returns the corresponding validator index.
// If raw ID is valid but unknown in current state, return MaxUint64.
func decodeValidatorId(st state.ReadOnlyBeaconState, rawId string) (idx primitives.ValidatorIndex, err error) {
	// Case 0. accept blank id request
	if rawId == "" {
		return math.MaxUint64, nil
	}

	numVals := uint64(st.NumValidators())

	// Case 1. pubkey
	hexId := append0x(rawId)
	pubkey, err := hexutil.Decode(hexId)
	if err == nil {
		if len(pubkey) != fieldparams.BLSPubkeyLength {
			// try uint parsing
			index, err := strconv.ParseUint(rawId, 10, 64)
			if err != nil {
				return math.MaxUint64, errors.New(fmt.Sprintf("Pubkey length is %d instead of %d", len(pubkey), fieldparams.BLSPubkeyLength))
			}
			// Case 2. validator index: if parsing succeeds, return index.
			if index >= numVals {
				// Unknown validator
				return math.MaxUint64, nil
			}
			return primitives.ValidatorIndex(index), nil
		}
		valIndex, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
		if !ok {
			// Unknown validator
			return math.MaxUint64, nil
		}
		return valIndex, nil
	}

	// Case 2. validator index
	index, err := strconv.ParseUint(rawId, 10, 64)
	if err != nil {
		return math.MaxUint64, errors.New(fmt.Sprintf("could not parse validator id: %s", rawId))
	}
	if index >= numVals {
		// Unknown validator
		return math.MaxUint64, nil
	}
	return primitives.ValidatorIndex(index), nil
}

// calculateWaitingEpoch returns a waiting epoch based on given state regarding with validators.
func calculateWaitingEpoch(activeCount, pendingQueuedCount uint64) uint64 {
	activationsPerEpoch := helpers.ValidatorExitNoBiasChurnLimit(activeCount)
	return (pendingQueuedCount+activationsPerEpoch)/activationsPerEpoch + uint64(params.BeaconConfig().MaxSeedLookahead)
}

func append0x(input string) string {
	if has0xPrefix(input) {
		return input
	}
	return "0x" + input
}

func has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}
