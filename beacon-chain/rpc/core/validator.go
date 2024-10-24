package core

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/validators"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v5/beacon-chain/forkchoice/types"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
)

var errOptimisticMode = errors.New("the node is currently optimistic and cannot serve validators")

// AggregateBroadcastFailedError represents an error scenario where
// broadcasting an aggregate selection proof failed.
type AggregateBroadcastFailedError struct {
	err error
}

// NewAggregateBroadcastFailedError creates a new error instance.
func NewAggregateBroadcastFailedError(err error) AggregateBroadcastFailedError {
	return AggregateBroadcastFailedError{
		err: err,
	}
}

// Error returns the underlying error message.
func (e *AggregateBroadcastFailedError) Error() string {
	return fmt.Sprintf("could not broadcast signed aggregated attestation: %s", e.err.Error())
}

// ComputeValidatorPerformance reports the validator's latest balance along with other important metrics on
// rewards and penalties throughout its lifecycle in the beacon chain.
func (s *Service) ComputeValidatorPerformance(
	ctx context.Context,
	req *ethpb.ValidatorPerformanceRequest,
) (*ethpb.ValidatorPerformanceResponse, *RpcError) {
	ctx, span := trace.StartSpan(ctx, "coreService.ComputeValidatorPerformance")
	defer span.End()

	if s.SyncChecker.Syncing() {
		return nil, &RpcError{Reason: Unavailable, Err: errors.New("Syncing to latest head, not ready to respond")}
	}

	headState, err := s.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, &RpcError{Err: errors.Wrap(err, "could not get head state"), Reason: Internal}
	}
	currSlot := s.GenesisTimeFetcher.CurrentSlot()
	if currSlot > headState.Slot() {
		headRoot, err := s.HeadFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, &RpcError{Err: errors.Wrap(err, "could not get head root"), Reason: Internal}
		}
		headState, err = transition.ProcessSlotsUsingNextSlotCache(ctx, headState, headRoot, currSlot)
		if err != nil {
			return nil, &RpcError{Err: errors.Wrapf(err, "could not process slots up to %d", currSlot), Reason: Internal}
		}
	}
	var validatorSummary []*precompute.Validator
	if headState.Version() == version.Phase0 {
		vp, bp, err := precompute.New(ctx, headState)
		if err != nil {
			return nil, &RpcError{Err: err, Reason: Internal}
		}
		vp, bp, err = precompute.ProcessAttestations(ctx, headState, vp, bp)
		if err != nil {
			return nil, &RpcError{Err: err, Reason: Internal}
		}
		headState, err = precompute.ProcessRewardsAndPenaltiesPrecompute(headState, bp, vp, precompute.AttestationsDelta, precompute.ProposersDelta)
		if err != nil {
			return nil, &RpcError{Err: err, Reason: Internal}
		}
		validatorSummary = vp
	} else if headState.Version() >= version.Altair {
		vp, bp, err := altair.InitializePrecomputeValidators(ctx, headState)
		if err != nil {
			return nil, &RpcError{Err: err, Reason: Internal}
		}
		vp, bp, err = altair.ProcessEpochParticipation(ctx, headState, bp, vp)
		if err != nil {
			return nil, &RpcError{Err: err, Reason: Internal}
		}
		headState, vp, err = altair.ProcessInactivityScores(ctx, headState, vp)
		if err != nil {
			return nil, &RpcError{Err: err, Reason: Internal}
		}
		headState, err = altair.ProcessRewardsAndPenaltiesPrecompute(headState, bp, vp)
		if err != nil {
			return nil, &RpcError{Err: err, Reason: Internal}
		}
		validatorSummary = vp
	} else {
		return nil, &RpcError{Err: errors.Wrapf(err, "head state version %d not supported", headState.Version()), Reason: Internal}
	}

	responseCap := len(req.Indices) + len(req.PublicKeys)
	validatorIndices := make([]primitives.ValidatorIndex, 0, responseCap)
	missingValidators := make([][]byte, 0, responseCap)

	filtered := map[primitives.ValidatorIndex]bool{} // Track filtered validators to prevent duplication in the response.
	// Convert the list of validator public keys to validator indices and add to the indices set.
	for _, pubKey := range req.PublicKeys {
		// Skip empty public key.
		if len(pubKey) == 0 {
			continue
		}
		pubkeyBytes := bytesutil.ToBytes48(pubKey)
		idx, ok := headState.ValidatorIndexByPubkey(pubkeyBytes)
		if !ok {
			// Validator index not found, track as missing.
			missingValidators = append(missingValidators, pubKey)
			continue
		}
		if !filtered[idx] {
			validatorIndices = append(validatorIndices, idx)
			filtered[idx] = true
		}
	}
	// Add provided indices to the indices set.
	for _, idx := range req.Indices {
		if !filtered[idx] {
			validatorIndices = append(validatorIndices, idx)
			filtered[idx] = true
		}
	}
	// Depending on the indices and public keys given, results might not be sorted.
	sort.Slice(validatorIndices, func(i, j int) bool {
		return validatorIndices[i] < validatorIndices[j]
	})

	currentEpoch := coreTime.CurrentEpoch(headState)
	responseCap = len(validatorIndices)
	pubKeys := make([][]byte, 0, responseCap)
	beforeTransitionBalances := make([]uint64, 0, responseCap)
	afterTransitionBalances := make([]uint64, 0, responseCap)
	effectiveBalances := make([]uint64, 0, responseCap)
	correctlyVotedSource := make([]bool, 0, responseCap)
	correctlyVotedTarget := make([]bool, 0, responseCap)
	correctlyVotedHead := make([]bool, 0, responseCap)
	inactivityScores := make([]uint64, 0, responseCap)
	// Append performance summaries.
	// Also track missing validators using public keys.
	for _, idx := range validatorIndices {
		val, err := headState.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			return nil, &RpcError{Err: errors.Wrap(err, "could not get validator"), Reason: Internal}
		}
		pubKey := val.PublicKey()
		if uint64(idx) >= uint64(len(validatorSummary)) {
			// Not listed in validator summary yet; treat it as missing.
			missingValidators = append(missingValidators, pubKey[:])
			continue
		}
		if !helpers.IsActiveValidatorUsingTrie(val, currentEpoch) {
			// Inactive validator; treat it as missing.
			missingValidators = append(missingValidators, pubKey[:])
			continue
		}

		summary := validatorSummary[idx]
		pubKeys = append(pubKeys, pubKey[:])
		effectiveBalances = append(effectiveBalances, summary.CurrentEpochEffectiveBalance)
		beforeTransitionBalances = append(beforeTransitionBalances, summary.BeforeEpochTransitionBalance)
		afterTransitionBalances = append(afterTransitionBalances, summary.AfterEpochTransitionBalance)
		correctlyVotedTarget = append(correctlyVotedTarget, summary.IsPrevEpochTargetAttester)
		correctlyVotedHead = append(correctlyVotedHead, summary.IsPrevEpochHeadAttester)

		if headState.Version() == version.Phase0 {
			correctlyVotedSource = append(correctlyVotedSource, summary.IsPrevEpochAttester)
		} else {
			correctlyVotedSource = append(correctlyVotedSource, summary.IsPrevEpochSourceAttester)
			inactivityScores = append(inactivityScores, summary.InactivityScore)
		}
	}

	return &ethpb.ValidatorPerformanceResponse{
		PublicKeys:                    pubKeys,
		CorrectlyVotedSource:          correctlyVotedSource,
		CorrectlyVotedTarget:          correctlyVotedTarget, // In altair, when this is true then the attestation was definitely included.
		CorrectlyVotedHead:            correctlyVotedHead,
		CurrentEffectiveBalances:      effectiveBalances,
		BalancesBeforeEpochTransition: beforeTransitionBalances,
		BalancesAfterEpochTransition:  afterTransitionBalances,
		MissingValidators:             missingValidators,
		InactivityScores:              inactivityScores, // Only populated in Altair
	}, nil
}

// IndividualVotes retrieves individual voting status of validators.
func (s *Service) IndividualVotes(
	ctx context.Context,
	req *ethpb.IndividualVotesRequest,
) (*ethpb.IndividualVotesRespond, *RpcError) {
	currentEpoch := slots.ToEpoch(s.GenesisTimeFetcher.CurrentSlot())
	if req.Epoch > currentEpoch {
		return nil, &RpcError{
			Err:    fmt.Errorf("cannot retrieve information about an epoch in the future, current epoch %d, requesting %d\n", currentEpoch, req.Epoch),
			Reason: BadRequest,
		}
	}

	slot, err := slots.EpochEnd(req.Epoch)
	if err != nil {
		return nil, &RpcError{Err: err, Reason: Internal}
	}
	st, err := s.ReplayerBuilder.ReplayerForSlot(slot).ReplayBlocks(ctx)
	if err != nil {
		return nil, &RpcError{
			Err:    errors.Wrapf(err, "failed to replay blocks for state at epoch %d", req.Epoch),
			Reason: Internal,
		}
	}
	// Track filtered validators to prevent duplication in the response.
	filtered := map[primitives.ValidatorIndex]bool{}
	filteredIndices := make([]primitives.ValidatorIndex, 0)
	votes := make([]*ethpb.IndividualVotesRespond_IndividualVote, 0, len(req.Indices)+len(req.PublicKeys))
	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
		if !ok {
			votes = append(votes, &ethpb.IndividualVotesRespond_IndividualVote{PublicKey: pubKey, ValidatorIndex: primitives.ValidatorIndex(^uint64(0))})
			continue
		}
		filtered[index] = true
		filteredIndices = append(filteredIndices, index)
	}
	// Filter out assignments by validator indices.
	for _, index := range req.Indices {
		if !filtered[index] {
			filteredIndices = append(filteredIndices, index)
		}
	}
	sort.Slice(filteredIndices, func(i, j int) bool {
		return filteredIndices[i] < filteredIndices[j]
	})

	var v []*precompute.Validator
	var bal *precompute.Balance
	if st.Version() == version.Phase0 {
		v, bal, err = precompute.New(ctx, st)
		if err != nil {
			return nil, &RpcError{
				Err:    errors.Wrapf(err, "could not set up pre compute instance"),
				Reason: Internal,
			}
		}
		v, _, err = precompute.ProcessAttestations(ctx, st, v, bal)
		if err != nil {
			return nil, &RpcError{
				Err:    errors.Wrapf(err, "could not pre compute attestations"),
				Reason: Internal,
			}
		}
	} else if st.Version() >= version.Altair {
		v, bal, err = altair.InitializePrecomputeValidators(ctx, st)
		if err != nil {
			return nil, &RpcError{
				Err:    errors.Wrapf(err, "could not set up altair pre compute instance"),
				Reason: Internal,
			}
		}
		v, _, err = altair.ProcessEpochParticipation(ctx, st, bal, v)
		if err != nil {
			return nil, &RpcError{
				Err:    errors.Wrapf(err, "could not pre compute attestations"),
				Reason: Internal,
			}
		}
	} else {
		return nil, &RpcError{
			Err:    errors.Wrapf(err, "invalid state type retrieved with a version of %d", st.Version()),
			Reason: Internal,
		}
	}

	for _, index := range filteredIndices {
		if uint64(index) >= uint64(len(v)) {
			votes = append(votes, &ethpb.IndividualVotesRespond_IndividualVote{ValidatorIndex: index})
			continue
		}
		val, err := st.ValidatorAtIndexReadOnly(index)
		if err != nil {
			return nil, &RpcError{
				Err:    errors.Wrapf(err, "could not retrieve validator"),
				Reason: Internal,
			}
		}
		pb := val.PublicKey()
		votes = append(votes, &ethpb.IndividualVotesRespond_IndividualVote{
			Epoch:                            req.Epoch,
			PublicKey:                        pb[:],
			ValidatorIndex:                   index,
			IsSlashed:                        v[index].IsSlashed,
			IsWithdrawableInCurrentEpoch:     v[index].IsWithdrawableCurrentEpoch,
			IsActiveInCurrentEpoch:           v[index].IsActiveCurrentEpoch,
			IsActiveInPreviousEpoch:          v[index].IsActivePrevEpoch,
			IsCurrentEpochAttester:           v[index].IsCurrentEpochAttester,
			IsCurrentEpochTargetAttester:     v[index].IsCurrentEpochTargetAttester,
			IsPreviousEpochAttester:          v[index].IsPrevEpochAttester,
			IsPreviousEpochTargetAttester:    v[index].IsPrevEpochTargetAttester,
			IsPreviousEpochHeadAttester:      v[index].IsPrevEpochHeadAttester,
			CurrentEpochEffectiveBalanceGwei: v[index].CurrentEpochEffectiveBalance,
			InclusionSlot:                    v[index].InclusionSlot,
			InclusionDistance:                v[index].InclusionDistance,
			InactivityScore:                  v[index].InactivityScore,
		})
	}

	return &ethpb.IndividualVotesRespond{
		IndividualVotes: votes,
	}, nil
}

// SubmitSignedAggregateSelectionProof verifies given aggregate and proofs and publishes them on appropriate gossipsub topic.
func (s *Service) SubmitSignedAggregateSelectionProof(
	ctx context.Context,
	agg ethpb.SignedAggregateAttAndProof,
) *RpcError {
	ctx, span := trace.StartSpan(ctx, "coreService.SubmitSignedAggregateSelectionProof")
	defer span.End()

	if agg == nil {
		return &RpcError{Err: errors.New("signed aggregate request can't be nil"), Reason: BadRequest}
	}
	attAndProof := agg.AggregateAttestationAndProof()
	if attAndProof == nil {
		return &RpcError{Err: errors.New("signed aggregate request can't be nil"), Reason: BadRequest}
	}
	att := attAndProof.AggregateVal()
	if att == nil {
		return &RpcError{Err: errors.New("signed aggregate request can't be nil"), Reason: BadRequest}
	}
	data := att.GetData()
	if data == nil {
		return &RpcError{Err: errors.New("signed aggregate request can't be nil"), Reason: BadRequest}
	}
	emptySig := make([]byte, fieldparams.BLSSignatureLength)
	if bytes.Equal(agg.GetSignature(), emptySig) || bytes.Equal(attAndProof.GetSelectionProof(), emptySig) {
		return &RpcError{Err: errors.New("signed signatures can't be zero hashes"), Reason: BadRequest}
	}

	// As a preventive measure, a beacon node shouldn't broadcast an attestation whose slot is out of range.
	if err := helpers.ValidateAttestationTime(
		data.Slot,
		s.GenesisTimeFetcher.GenesisTime(),
		params.BeaconConfig().MaximumGossipClockDisparityDuration(),
	); err != nil {
		return &RpcError{Err: errors.New("attestation slot is no longer valid from current time"), Reason: BadRequest}
	}

	if err := s.Broadcaster.Broadcast(ctx, agg); err != nil {
		return &RpcError{Err: &AggregateBroadcastFailedError{err: err}, Reason: Internal}
	}

	if logrus.GetLevel() >= logrus.DebugLevel {
		var fields logrus.Fields
		if agg.Version() >= version.Electra {
			fields = logrus.Fields{
				"slot":             data.Slot,
				"committeeCount":   att.CommitteeBitsVal().Count(),
				"committeeIndices": att.CommitteeBitsVal().BitIndices(),
				"validatorIndex":   attAndProof.GetAggregatorIndex(),
				"aggregatedCount":  att.GetAggregationBits().Count(),
			}
		} else {
			fields = logrus.Fields{
				"slot":            data.Slot,
				"committeeIndex":  data.CommitteeIndex,
				"validatorIndex":  attAndProof.GetAggregatorIndex(),
				"aggregatedCount": att.GetAggregationBits().Count(),
			}
		}
		log.WithFields(fields).Debug("Broadcasting aggregated attestation and proof")
	}

	return nil
}

// GetAttestationData requests that the beacon node produces attestation data for
// the requested committee index and slot based on the nodes current head.
func (s *Service) GetAttestationData(
	ctx context.Context, req *ethpb.AttestationDataRequest,
) (*ethpb.AttestationData, *RpcError) {
	ctx, span := trace.StartSpan(ctx, "coreService.GetAttestationData")
	defer span.End()

	if req.Slot != s.GenesisTimeFetcher.CurrentSlot() {
		return nil, &RpcError{Reason: BadRequest, Err: errors.Errorf("invalid request: slot %d is not the current slot %d", req.Slot, s.GenesisTimeFetcher.CurrentSlot())}
	}
	if err := helpers.ValidateAttestationTime(
		req.Slot,
		s.GenesisTimeFetcher.GenesisTime(),
		params.BeaconConfig().MaximumGossipClockDisparityDuration(),
	); err != nil {
		return nil, &RpcError{Reason: BadRequest, Err: errors.Errorf("invalid request: %v", err)}
	}

	committeeIndex := primitives.CommitteeIndex(0)
	if slots.ToEpoch(req.Slot) < params.BeaconConfig().ElectraForkEpoch {
		committeeIndex = req.CommitteeIndex
	}

	s.AttestationCache.RLock()
	res := s.AttestationCache.Get()
	if res != nil && res.Slot == req.Slot {
		s.AttestationCache.RUnlock()
		return &ethpb.AttestationData{
			Slot:            res.Slot,
			CommitteeIndex:  committeeIndex,
			BeaconBlockRoot: res.HeadRoot,
			Source: &ethpb.Checkpoint{
				Epoch: res.Source.Epoch,
				Root:  res.Source.Root[:],
			},
			Target: &ethpb.Checkpoint{
				Epoch: res.Target.Epoch,
				Root:  res.Target.Root[:],
			},
		}, nil
	}
	s.AttestationCache.RUnlock()

	s.AttestationCache.Lock()
	defer s.AttestationCache.Unlock()

	// We check the cache again as in the event there are multiple inflight requests for
	// the same attestation data, the cache might have been filled while we were waiting
	// to acquire the lock.
	res = s.AttestationCache.Get()
	if res != nil && res.Slot == req.Slot {
		return &ethpb.AttestationData{
			Slot:            res.Slot,
			CommitteeIndex:  committeeIndex,
			BeaconBlockRoot: res.HeadRoot,
			Source: &ethpb.Checkpoint{
				Epoch: res.Source.Epoch,
				Root:  res.Source.Root[:],
			},
			Target: &ethpb.Checkpoint{
				Epoch: res.Target.Epoch,
				Root:  res.Target.Root[:],
			},
		}, nil
	}
	// cache miss, we need to check for optimistic status before proceeding
	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: err}
	}
	if optimistic {
		return nil, &RpcError{Reason: Unavailable, Err: errOptimisticMode}
	}

	headRoot, err := s.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not get head root")}
	}
	targetEpoch := slots.ToEpoch(req.Slot)
	targetRoot, err := s.HeadFetcher.TargetRootForEpoch(bytesutil.ToBytes32(headRoot), targetEpoch)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not get target root")}
	}

	headState, err := s.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not get head state")}
	}
	if coreTime.CurrentEpoch(headState) < slots.ToEpoch(req.Slot) { // Ensure justified checkpoint safety by processing head state across the boundary.
		headState, err = transition.ProcessSlotsUsingNextSlotCache(ctx, headState, headRoot, req.Slot)
		if err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Errorf("could not process slots up to %d: %v", req.Slot, err)}
		}
	}
	justifiedCheckpoint := headState.CurrentJustifiedCheckpoint()

	if err = s.AttestationCache.Put(&cache.AttestationConsensusData{
		Slot:     req.Slot,
		HeadRoot: headRoot,
		Target: forkchoicetypes.Checkpoint{
			Epoch: targetEpoch,
			Root:  targetRoot,
		},
		Source: forkchoicetypes.Checkpoint{
			Epoch: justifiedCheckpoint.Epoch,
			Root:  bytesutil.ToBytes32(justifiedCheckpoint.Root),
		},
	}); err != nil {
		log.WithError(err).Error("Failed to put attestation data into cache")
	}

	return &ethpb.AttestationData{
		Slot:            req.Slot,
		CommitteeIndex:  committeeIndex,
		BeaconBlockRoot: headRoot,
		Source: &ethpb.Checkpoint{
			Epoch: justifiedCheckpoint.Epoch,
			Root:  justifiedCheckpoint.Root,
		},
		Target: &ethpb.Checkpoint{
			Epoch: targetEpoch,
			Root:  targetRoot[:],
		},
	}, nil
}

// ValidatorParticipation retrieves the validator participation information for a given epoch,
// it returns the information about validator's participation rate in voting on the proof of stake
// rules based on their balance compared to the total active validator balance.
func (s *Service) ValidatorParticipation(
	ctx context.Context,
	requestedEpoch primitives.Epoch,
) (
	*ethpb.ValidatorParticipationResponse,
	*RpcError,
) {
	currentSlot := s.GenesisTimeFetcher.CurrentSlot()
	currentEpoch := slots.ToEpoch(currentSlot)

	if requestedEpoch > currentEpoch {
		return nil, &RpcError{
			Err:    fmt.Errorf("cannot retrieve information about an epoch greater than current epoch, current epoch %d, requesting %d", currentEpoch, requestedEpoch),
			Reason: BadRequest,
		}
	}
	// Use the last slot of requested epoch to obtain current and previous epoch attestations.
	// This ensures that we don't miss previous attestations when input requested epochs.
	endSlot, err := slots.EpochEnd(requestedEpoch)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not get slot from requested epoch")}
	}
	// Get as close as we can to the end of the current epoch without going past the current slot.
	// The above check ensures a future *epoch* isn't requested, but the end slot of the requested epoch could still
	// be past the current slot. In that case, use the current slot as the best approximation of the requested epoch.
	// Replayer will make sure the slot ultimately used is canonical.
	if endSlot > currentSlot {
		endSlot = currentSlot
	}

	// ReplayerBuilder ensures that a canonical chain is followed to the slot
	beaconSt, err := s.ReplayerBuilder.ReplayerForSlot(endSlot).ReplayBlocks(ctx)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrapf(err, "error replaying blocks for state at slot %d", endSlot)}
	}
	var v []*precompute.Validator
	var b *precompute.Balance

	if beaconSt.Version() == version.Phase0 {
		v, b, err = precompute.New(ctx, beaconSt)
		if err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not set up pre compute instance")}
		}
		_, b, err = precompute.ProcessAttestations(ctx, beaconSt, v, b)
		if err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not pre compute attestations")}
		}
	} else if beaconSt.Version() >= version.Altair {
		v, b, err = altair.InitializePrecomputeValidators(ctx, beaconSt)
		if err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not set up altair pre compute instance")}
		}
		_, b, err = altair.ProcessEpochParticipation(ctx, beaconSt, b, v)
		if err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not pre compute attestations: %v")}
		}
	} else {
		return nil, &RpcError{Reason: Internal, Err: fmt.Errorf("invalid state type retrieved with a version of %s", version.String(beaconSt.Version()))}
	}

	cp := s.FinalizedFetcher.FinalizedCheckpt()
	p := &ethpb.ValidatorParticipationResponse{
		Epoch:     requestedEpoch,
		Finalized: requestedEpoch <= cp.Epoch,
		Participation: &ethpb.ValidatorParticipation{
			// TODO(7130): Remove these three deprecated fields.
			GlobalParticipationRate:          float32(b.PrevEpochTargetAttested) / float32(b.ActivePrevEpoch),
			VotedEther:                       b.PrevEpochTargetAttested,
			EligibleEther:                    b.ActivePrevEpoch,
			CurrentEpochActiveGwei:           b.ActiveCurrentEpoch,
			CurrentEpochAttestingGwei:        b.CurrentEpochAttested,
			CurrentEpochTargetAttestingGwei:  b.CurrentEpochTargetAttested,
			PreviousEpochActiveGwei:          b.ActivePrevEpoch,
			PreviousEpochAttestingGwei:       b.PrevEpochAttested,
			PreviousEpochTargetAttestingGwei: b.PrevEpochTargetAttested,
			PreviousEpochHeadAttestingGwei:   b.PrevEpochHeadAttested,
		},
	}
	return p, nil
}

// ValidatorActiveSetChanges retrieves the active set changes for a given epoch.
//
// This data includes any activations, voluntary exits, and bail outs.
func (s *Service) ValidatorActiveSetChanges(
	ctx context.Context,
	requestedEpoch primitives.Epoch,
) (
	*ethpb.ActiveSetChanges,
	*RpcError,
) {
	currentEpoch := slots.ToEpoch(s.GenesisTimeFetcher.CurrentSlot())
	if requestedEpoch > currentEpoch {
		return nil, &RpcError{
			Err:    errors.Errorf("cannot retrieve information about an epoch in the future, current epoch %d, requesting %d", currentEpoch, requestedEpoch),
			Reason: BadRequest,
		}
	}

	slot, err := slots.EpochStart(requestedEpoch)
	if err != nil {
		return nil, &RpcError{Err: err, Reason: BadRequest}
	}
	requestedState, err := s.ReplayerBuilder.ReplayerForSlot(slot).ReplayBlocks(ctx)
	if err != nil {
		return nil, &RpcError{
			Err:    errors.Wrapf(err, "error replaying blocks for state at slot %d", slot),
			Reason: Internal,
		}
	}

	vs := requestedState.Validators()
	activatedIndices := validators.ActivatedValidatorIndices(coreTime.CurrentEpoch(requestedState), vs)

	// Determine whether requested epoch is in inactivity leak period.
	previousEpoch := coreTime.PrevEpoch(requestedState)
	finalizedEpoch := requestedState.FinalizedCheckpointEpoch()
	isInInactivityLeak := helpers.IsInInactivityLeak(previousEpoch, finalizedEpoch)

	exitedIndices, err := validators.ExitedValidatorIndices(requestedState, vs, isInInactivityLeak)
	if err != nil {
		return nil, &RpcError{
			Err:    errors.Wrap(err, "could not determine exited validator indices"),
			Reason: Internal,
		}
	}
	slashedIndices := validators.SlashedValidatorIndices(coreTime.CurrentEpoch(requestedState), vs)
	bailedOutIndices, err := validators.BailedOutValidatorIndices(requestedState, vs, isInInactivityLeak)
	if err != nil {
		return nil, &RpcError{
			Err:    errors.Wrap(err, "could not determine bailed out validator indices"),
			Reason: Internal,
		}
	}

	// Retrieve public keys for the indices.
	activatedKeys := make([][]byte, len(activatedIndices))
	exitedKeys := make([][]byte, len(exitedIndices))
	slashedKeys := make([][]byte, len(slashedIndices))
	bailedOutKeys := make([][]byte, len(bailedOutIndices))
	for i, idx := range activatedIndices {
		pubkey := requestedState.PubkeyAtIndex(idx)
		activatedKeys[i] = pubkey[:]
	}
	for i, idx := range exitedIndices {
		pubkey := requestedState.PubkeyAtIndex(idx)
		exitedKeys[i] = pubkey[:]
	}
	for i, idx := range slashedIndices {
		pubkey := requestedState.PubkeyAtIndex(idx)
		slashedKeys[i] = pubkey[:]
	}
	for i, idx := range bailedOutIndices {
		pubkey := requestedState.PubkeyAtIndex(idx)
		bailedOutKeys[i] = pubkey[:]
	}

	return &ethpb.ActiveSetChanges{
		Epoch:               requestedEpoch,
		ActivatedPublicKeys: activatedKeys,
		ActivatedIndices:    activatedIndices,
		ExitedPublicKeys:    exitedKeys,
		ExitedIndices:       exitedIndices,
		SlashedPublicKeys:   slashedKeys,
		SlashedIndices:      slashedIndices,
		BailedOutPublicKeys: bailedOutKeys,
		BailedOutIndices:    bailedOutIndices,
	}, nil
}
