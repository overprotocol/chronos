package helpers

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v5/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/hash"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	log "github.com/sirupsen/logrus"
)

var (
	CommitteeCacheInProgressHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "committee_cache_in_progress_hit",
		Help: "The number of committee requests that are present in the cache.",
	})

	errProposerIndexMiss = errors.New("propoposer index not found in cache")
)

// IsActiveValidator returns the boolean value on whether the validator
// is active or not.
//
// Spec pseudocode definition:
//
//	def is_active_validator(validator: Validator, epoch: Epoch) -> bool:
//	  """
//	  Check if ``validator`` is active.
//	  """
//	  return validator.activation_epoch <= epoch < validator.exit_epoch
func IsActiveValidator(validator *ethpb.Validator, epoch primitives.Epoch) bool {
	return checkValidatorActiveStatus(validator.ActivationEpoch, validator.ExitEpoch, epoch)
}

// IsActiveValidatorUsingTrie checks if a read only validator is active.
func IsActiveValidatorUsingTrie(validator state.ReadOnlyValidator, epoch primitives.Epoch) bool {
	return checkValidatorActiveStatus(validator.ActivationEpoch(), validator.ExitEpoch(), epoch)
}

// IsActiveNonSlashedValidatorUsingTrie checks if a read only validator is active and not slashed
func IsActiveNonSlashedValidatorUsingTrie(validator state.ReadOnlyValidator, epoch primitives.Epoch) bool {
	active := checkValidatorActiveStatus(validator.ActivationEpoch(), validator.ExitEpoch(), epoch)
	return active && !validator.Slashed()
}

func checkValidatorActiveStatus(activationEpoch, exitEpoch, epoch primitives.Epoch) bool {
	return activationEpoch <= epoch && epoch < exitEpoch
}

// IsPendingValidatorUsingTrie checks if a read only validator is waiting for activation.
func IsPendingValidatorUsingTrie(validator state.ReadOnlyValidator, epoch primitives.Epoch) bool {
	return checkValidatorPendingStatus(validator.ActivationEpoch(), epoch)
}

func checkValidatorPendingStatus(activationEpoch, epoch primitives.Epoch) bool {
	return activationEpoch > epoch
}

// IsExitingValidatorUsingTrie checks if a read only validator is waiting for exit.
func IsExitingValidatorUsingTrie(validator state.ReadOnlyValidator, epoch primitives.Epoch) bool {
	return checkValidatorExitingStatus(validator.ActivationEpoch(), validator.ExitEpoch(), epoch)
}

func checkValidatorExitingStatus(activationEpoch, exitEpoch, epoch primitives.Epoch) bool {
	return activationEpoch <= epoch && epoch < exitEpoch && exitEpoch != params.BeaconConfig().FarFutureEpoch
}

// IsSlashableValidator returns the boolean value on whether the validator
// is slashable or not.
//
// Spec pseudocode definition:
//
//	def is_slashable_validator(validator: Validator, epoch: Epoch) -> bool:
//	"""
//	Check if ``validator`` is slashable.
//	"""
//	return (not validator.slashed) and (validator.activation_epoch <= epoch < get_withdrawable_epoch(validator))
func IsSlashableValidator(activationEpoch, withdrawableEpoch primitives.Epoch, slashed bool, epoch primitives.Epoch) bool {
	return checkValidatorSlashable(activationEpoch, withdrawableEpoch, slashed, epoch)
}

// IsSlashableValidatorUsingTrie checks if a read only validator is slashable.
func IsSlashableValidatorUsingTrie(val state.ReadOnlyValidator, epoch primitives.Epoch) bool {
	withdrawableEpoch := GetWithdrawableEpoch(val.ExitEpoch(), val.Slashed())
	return checkValidatorSlashable(val.ActivationEpoch(), withdrawableEpoch, val.Slashed(), epoch)
}

func checkValidatorSlashable(activationEpoch, withdrawableEpoch primitives.Epoch, slashed bool, epoch primitives.Epoch) bool {
	active := activationEpoch <= epoch
	beforeWithdrawable := epoch < withdrawableEpoch
	return beforeWithdrawable && active && !slashed
}

// ActiveValidatorIndices filters out active validators based on validator status
// and returns their indices in a list.
//
// WARNING: This method allocates a new copy of the validator index set and is
// considered to be very memory expensive. Avoid using this unless you really
// need the active validator indices for some specific reason.
//
// Spec pseudocode definition:
//
//	def get_active_validator_indices(state: BeaconState, epoch: Epoch) -> Sequence[ValidatorIndex]:
//	  """
//	  Return the sequence of active validator indices at ``epoch``.
//	  """
//	  return [ValidatorIndex(i) for i, v in enumerate(state.validators) if is_active_validator(v, epoch)]
func ActiveValidatorIndices(ctx context.Context, s state.ReadOnlyBeaconState, epoch primitives.Epoch) ([]primitives.ValidatorIndex, error) {
	seed, err := Seed(s, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return nil, errors.Wrap(err, "could not get seed")
	}
	activeIndices, err := committeeCache.ActiveIndices(ctx, seed)
	if err != nil {
		return nil, errors.Wrap(err, "could not interface with committee cache")
	}
	if activeIndices != nil {
		return activeIndices, nil
	}

	if err := committeeCache.MarkInProgress(seed); err != nil {
		if errors.Is(err, cache.ErrAlreadyInProgress) {
			activeIndices, err := committeeCache.ActiveIndices(ctx, seed)
			if err != nil {
				return nil, err
			}
			if activeIndices == nil {
				return nil, errors.New("nil active indices")
			}
			CommitteeCacheInProgressHit.Inc()
			return activeIndices, nil
		}
		return nil, errors.Wrap(err, "could not mark committee cache as in progress")
	}
	defer func() {
		if err := committeeCache.MarkNotInProgress(seed); err != nil {
			log.WithError(err).Error("Could not mark cache not in progress")
		}
	}()

	var indices []primitives.ValidatorIndex
	if err := s.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		if IsActiveValidatorUsingTrie(val, epoch) {
			indices = append(indices, primitives.ValidatorIndex(idx))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if len(indices) == 0 {
		return nil, errors.New("no active validator indices")
	}

	if err := UpdateCommitteeCache(ctx, s, epoch); err != nil {
		return nil, errors.Wrap(err, "could not update committee cache")
	}

	return indices, nil
}

// ActiveValidatorCount returns the number of active validators in the state
// at the given epoch.
func ActiveValidatorCount(ctx context.Context, s state.ReadOnlyBeaconState, epoch primitives.Epoch) (uint64, error) {
	seed, err := Seed(s, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return 0, errors.Wrap(err, "could not get seed")
	}
	activeCount, err := committeeCache.ActiveIndicesCount(ctx, seed)
	if err != nil {
		return 0, errors.Wrap(err, "could not interface with committee cache")
	}
	if activeCount != 0 && s.Slot() != 0 {
		return uint64(activeCount), nil
	}

	if err := committeeCache.MarkInProgress(seed); err != nil {
		if errors.Is(err, cache.ErrAlreadyInProgress) {
			activeCount, err := committeeCache.ActiveIndicesCount(ctx, seed)
			if err != nil {
				return 0, err
			}
			CommitteeCacheInProgressHit.Inc()
			return uint64(activeCount), nil
		}
		return 0, errors.Wrap(err, "could not mark committee cache as in progress")
	}
	defer func() {
		if err := committeeCache.MarkNotInProgress(seed); err != nil {
			log.WithError(err).Error("Could not mark cache not in progress")
		}
	}()

	count := uint64(0)
	if err := s.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		if IsActiveValidatorUsingTrie(val, epoch) {
			count++
		}
		return nil
	}); err != nil {
		return 0, err
	}

	if err := UpdateCommitteeCache(ctx, s, epoch); err != nil {
		return 0, errors.Wrap(err, "could not update committee cache")
	}

	return count, nil
}

// ActivationExitEpoch takes in epoch number and returns when
// the validator is eligible for activation and exit.
//
// Spec pseudocode definition:
//
//	def compute_activation_exit_epoch(epoch: Epoch) -> Epoch:
//	  """
//	  Return the epoch during which validator activations and exits initiated in ``epoch`` take effect.
//	  """
//	  return Epoch(epoch + 1 + MAX_SEED_LOOKAHEAD)
func ActivationExitEpoch(epoch primitives.Epoch) primitives.Epoch {
	return epoch + 1 + params.BeaconConfig().MaxSeedLookahead
}

// calculateChurnLimit based on the formula in the spec.
//
//	def get_validator_churn_limit(state: BeaconState) -> uint64:
//	 """
//	 Return the validator churn limit for the current epoch.
//	 """
//	 active_validator_indices = get_active_validator_indices(state, get_current_epoch(state))
//	 return max(MIN_PER_EPOCH_CHURN_LIMIT, uint64(len(active_validator_indices)) // CHURN_LIMIT_QUOTIENT)
func calculateChurnLimit(activeValidatorCount uint64) uint64 {
	churnLimit := activeValidatorCount / params.BeaconConfig().ChurnLimitQuotient
	if churnLimit < params.BeaconConfig().MinPerEpochChurnLimit {
		return params.BeaconConfig().MinPerEpochChurnLimit
	}
	return churnLimit
}

// ValidatorActivationChurnLimit returns the maximum number of validators that can be activated in a slot.
func ValidatorActivationChurnLimit(activeValidatorCount uint64) uint64 {
	return calculateChurnLimit(activeValidatorCount)
}

// ValidatorExitChurnLimit returns the maximum number of validators that can be exited in a slot.
func ValidatorExitChurnLimit(activeValidatorCount uint64) uint64 {
	return calculateChurnLimit(activeValidatorCount)
}

// BeaconProposerIndex returns proposer index of a current slot.
//
// Spec pseudocode definition:
//
//	def get_beacon_proposer_index(state: BeaconState) -> ValidatorIndex:
//	  """
//	  Return the beacon proposer index at the current slot.
//	  """
//	  epoch = get_current_epoch(state)
//	  seed = hash(get_seed(state, epoch, DOMAIN_BEACON_PROPOSER) + uint_to_bytes(state.slot))
//	  indices = get_active_validator_indices(state, epoch)
//	  return compute_proposer_index(state, indices, seed)
func BeaconProposerIndex(ctx context.Context, state state.ReadOnlyBeaconState) (primitives.ValidatorIndex, error) {
	return BeaconProposerIndexAtSlot(ctx, state, state.Slot())
}

// cachedProposerIndexAtSlot returns the proposer index at the given slot from
// the cache at the given root key.
func cachedProposerIndexAtSlot(slot primitives.Slot, root [32]byte) (primitives.ValidatorIndex, error) {
	proposerIndices, has := proposerIndicesCache.ProposerIndices(slots.ToEpoch(slot), root)
	if !has {
		return 0, errProposerIndexMiss
	}
	if len(proposerIndices) != int(params.BeaconConfig().SlotsPerEpoch) {
		return 0, errProposerIndexMiss
	}
	return proposerIndices[slot%params.BeaconConfig().SlotsPerEpoch], nil
}

// ProposerIndexAtSlotFromCheckpoint returns the proposer index at the given
// slot from the cache at the given checkpoint
func ProposerIndexAtSlotFromCheckpoint(c *forkchoicetypes.Checkpoint, slot primitives.Slot) (primitives.ValidatorIndex, error) {
	proposerIndices, has := proposerIndicesCache.IndicesFromCheckpoint(*c)
	if !has {
		return 0, errProposerIndexMiss
	}
	if len(proposerIndices) != int(params.BeaconConfig().SlotsPerEpoch) {
		return 0, errProposerIndexMiss
	}
	return proposerIndices[slot%params.BeaconConfig().SlotsPerEpoch], nil
}

// BeaconProposerIndexAtSlot returns proposer index at the given slot from the
// point of view of the given state as head state
func BeaconProposerIndexAtSlot(ctx context.Context, state state.ReadOnlyBeaconState, slot primitives.Slot) (primitives.ValidatorIndex, error) {
	e := slots.ToEpoch(slot)
	// The cache uses the state root of the previous epoch - minimum_seed_lookahead last slot as key. (e.g. Starting epoch 1, slot 32, the key would be block root at slot 31)
	// For simplicity, the node will skip caching of genesis epoch.
	if e > params.BeaconConfig().GenesisEpoch+params.BeaconConfig().MinSeedLookahead {
		s, err := slots.EpochEnd(e - 1)
		if err != nil {
			return 0, err
		}
		r, err := StateRootAtSlot(state, s)
		if err != nil {
			return 0, err
		}
		if r != nil && !bytes.Equal(r, params.BeaconConfig().ZeroHash[:]) {
			pid, err := cachedProposerIndexAtSlot(slot, [32]byte(r))
			if err == nil {
				return pid, nil
			}
			if err := UpdateProposerIndicesInCache(ctx, state, e); err != nil {
				return 0, errors.Wrap(err, "could not update proposer index cache")
			}
			pid, err = cachedProposerIndexAtSlot(slot, [32]byte(r))
			if err == nil {
				return pid, nil
			}
		}
	}

	seed, err := Seed(state, e, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		return 0, errors.Wrap(err, "could not generate seed")
	}

	seedWithSlot := append(seed[:], bytesutil.Bytes8(uint64(slot))...)
	seedWithSlotHash := hash.Hash(seedWithSlot)

	indices, err := ActiveValidatorIndices(ctx, state, e)
	if err != nil {
		return 0, errors.Wrap(err, "could not get active indices")
	}

	return ComputeProposerIndex(state, indices, seedWithSlotHash)
}

// ComputeProposerIndex returns the index sampled by effective balance, which is used to calculate proposer.
//
// nolint:dupword
// Spec pseudocode definition:
//
//	def compute_proposer_index(state: BeaconState, indices: Sequence[ValidatorIndex], seed: Bytes32) -> ValidatorIndex:
//	  """
//	  Return from ``indices`` a random index sampled by effective balance.
//	  """
//	  assert len(indices) > 0
//	  MAX_RANDOM_BYTE = 2**8 - 1
//	  i = uint64(0)
//	  total = uint64(len(indices))
//	  while True:
//	      candidate_index = indices[compute_shuffled_index(i % total, total, seed)]
//	      random_byte = hash(seed + uint_to_bytes(uint64(i // 32)))[i % 32]
//	      effective_balance = state.validators[candidate_index].effective_balance
//	      if effective_balance * MAX_RANDOM_BYTE >= MAX_EFFECTIVE_BALANCE_ALPACA * random_byte: #[Modified in Electra:EIP7251]
//	          return candidate_index
//	      i += 1
func ComputeProposerIndex(bState state.ReadOnlyBeaconState, activeIndices []primitives.ValidatorIndex, seed [32]byte) (primitives.ValidatorIndex, error) {
	length := uint64(len(activeIndices))
	if length == 0 {
		return 0, errors.New("empty active indices list")
	}
	maxRandomByte := uint64(1<<8 - 1)
	hashFunc := hash.CustomSHA256Hasher()

	for i := uint64(0); ; i++ {
		candidateIndex, err := ComputeShuffledIndex(primitives.ValidatorIndex(i%length), length, seed, true /* shuffle */)
		if err != nil {
			return 0, err
		}
		candidateIndex = activeIndices[candidateIndex]
		if uint64(candidateIndex) >= uint64(bState.NumValidators()) {
			return 0, errors.New("active index out of range")
		}
		b := append(seed[:], bytesutil.Bytes8(i/32)...)
		randomByte := hashFunc(b)[i%32]
		v, err := bState.ValidatorAtIndexReadOnly(candidateIndex)
		if err != nil {
			return 0, err
		}
		effectiveBal := v.EffectiveBalance()

		maxEB := params.BeaconConfig().MaxEffectiveBalance
		if bState.Version() >= version.Alpaca {
			maxEB = params.BeaconConfig().MaxEffectiveBalanceAlpaca
		}

		if effectiveBal*maxRandomByte >= maxEB*uint64(randomByte) {
			return candidateIndex, nil
		}
	}
}

// IsEligibleForActivationQueue checks if the validator is eligible to
// be placed into the activation queue.
//
// Spec definition:
//
//	def is_eligible_for_activation_queue(validator: Validator) -> bool:
//	    """
//	    Check if ``validator`` is eligible to be placed into the activation queue.
//	    """
//	    return (
//	        validator.activation_eligibility_epoch == FAR_FUTURE_EPOCH
//	        and validator.effective_balance >= MIN_ACTIVATION_BALANCE  # [Modified in Electra:EIP7251]
//	    )
func IsEligibleForActivationQueue(validator state.ReadOnlyValidator, currentEpoch primitives.Epoch) bool {
	if currentEpoch >= params.BeaconConfig().AlpacaForkEpoch {
		return isEligibleForActivationQueueElectra(validator.ActivationEligibilityEpoch(), validator.EffectiveBalance())
	}
	return isEligibleForActivationQueue(validator.ActivationEligibilityEpoch(), validator.EffectiveBalance())
}

// isEligibleForActivationQueue carries out the logic for IsEligibleForActivationQueue
// Spec pseudocode definition:
//
//	def is_eligible_for_activation_queue(validator: Validator) -> bool:
//	  """
//	  Check if ``validator`` is eligible to be placed into the activation queue.
//	  """
//	  return (
//	      validator.activation_eligibility_epoch == FAR_FUTURE_EPOCH
//	      and validator.effective_balance == MAX_EFFECTIVE_BALANCE
//	  )
func isEligibleForActivationQueue(activationEligibilityEpoch primitives.Epoch, effectiveBalance uint64) bool {
	return activationEligibilityEpoch == params.BeaconConfig().FarFutureEpoch &&
		effectiveBalance == params.BeaconConfig().MaxEffectiveBalance
}

// IsEligibleForActivationQueue checks if the validator is eligible to
// be placed into the activation queue.
//
// Spec definition:
//
//	def is_eligible_for_activation_queue(validator: Validator) -> bool:
//	    """
//	    Check if ``validator`` is eligible to be placed into the activation queue.
//	    """
//	    return (
//	        validator.activation_eligibility_epoch == FAR_FUTURE_EPOCH
//	        and validator.effective_balance >= MIN_ACTIVATION_BALANCE  # [Modified in Electra:EIP7251]
//	    )
func isEligibleForActivationQueueElectra(activationEligibilityEpoch primitives.Epoch, effectiveBalance uint64) bool {
	return activationEligibilityEpoch == params.BeaconConfig().FarFutureEpoch &&
		effectiveBalance >= params.BeaconConfig().MinActivationBalance
}

// IsEligibleForActivation checks if the validator is eligible for activation.
//
// Spec pseudocode definition:
//
//	def is_eligible_for_activation(state: BeaconState, validator: Validator) -> bool:
//	  """
//	  Check if ``validator`` is eligible for activation.
//	  """
//	  return (
//	      # Placement in queue is finalized
//	      validator.activation_eligibility_epoch <= state.finalized_checkpoint.epoch
//	      # Has not yet been activated
//	      and validator.activation_epoch == FAR_FUTURE_EPOCH
//	  )
func IsEligibleForActivation(state state.ReadOnlyCheckpoint, validator *ethpb.Validator) bool {
	finalizedEpoch := state.FinalizedCheckpointEpoch()
	return isEligibleForActivation(validator.ActivationEligibilityEpoch, validator.ActivationEpoch, finalizedEpoch)
}

// IsEligibleForActivationUsingROVal checks if the validator is eligible for activation using the provided read only validator.
func IsEligibleForActivationUsingROVal(state state.ReadOnlyCheckpoint, validator state.ReadOnlyValidator) bool {
	return isEligibleForActivation(validator.ActivationEligibilityEpoch(), validator.ActivationEpoch(), state.FinalizedCheckpointEpoch())
}

// isEligibleForActivation carries out the logic for IsEligibleForActivation*
func isEligibleForActivation(activationEligibilityEpoch, activationEpoch, finalizedEpoch primitives.Epoch) bool {
	return activationEligibilityEpoch <= finalizedEpoch &&
		activationEpoch == params.BeaconConfig().FarFutureEpoch
}

// IsEligibleForBailOut checks if the validator is eligible for bailout.
func IsEligibleForBailOut(state state.ReadOnlyBeaconState, validator state.ReadOnlyValidator, idx int, leak bool) (bool, error) {
	currentEpoch := time.CurrentEpoch(state)
	isActive := IsActiveValidatorUsingTrie(validator, currentEpoch)
	pb := validator.PrincipalBalance()
	actualBalance, err := state.BalanceAtIndex(primitives.ValidatorIndex(idx))
	if err != nil {
		return false, err
	}

	inactivityScore, err := inactivityScoreAtIndex(state, idx)
	if err != nil {
		return false, err
	}

	if isActive && isBelowThresholdForBailOut(actualBalance, pb) {
		return true, nil
	} else if leak && inactivityScore > params.BeaconConfig().InactivityLeakBailoutScoreThreshold {
		return true, nil
	}

	return false, nil
}

// IsBailOut checks if the validator is bailed out.
func IsBailOut(state state.ReadOnlyBeaconState, validator *ethpb.Validator, idx int, leak bool) (bool, error) {
	actualBalance, err := state.BalanceAtIndex(primitives.ValidatorIndex(idx))
	if err != nil {
		return false, err
	}

	pb := validator.PrincipalBalance

	inactivityScore, err := inactivityScoreAtIndex(state, idx)
	if err != nil {
		return false, err
	}

	return isBelowThresholdForBailOut(actualBalance, pb) || (leak && inactivityScore > params.BeaconConfig().InactivityLeakBailoutScoreThreshold), nil
}

func isBelowThresholdForBailOut(actualBalance, principalBalance uint64) bool {
	bailoutBuffer := principalBalance * params.BeaconConfig().InactivityPenaltyRate / params.BeaconConfig().InactivityPenaltyRatePrecision
	return actualBalance+bailoutBuffer < principalBalance
}

func inactivityScoreAtIndex(state state.ReadOnlyBeaconState, idx int) (uint64, error) {
	if state.Version() < version.Altair {
		return 0, nil
	}
	inactivityScore, err := state.InactivityScoreAtIndex(primitives.ValidatorIndex(idx))
	if err != nil {
		return 0, err
	}

	return inactivityScore, nil
}

// LastActivatedValidatorIndex provides the last activated validator given a state
func LastActivatedValidatorIndex(ctx context.Context, st state.ReadOnlyBeaconState) (primitives.ValidatorIndex, error) {
	_, span := trace.StartSpan(ctx, "helpers.LastActivatedValidatorIndex")
	defer span.End()
	var lastActivatedvalidatorIndex primitives.ValidatorIndex
	// linear search because status are not sorted
	for j := st.NumValidators() - 1; j >= 0; j-- {
		val, err := st.ValidatorAtIndexReadOnly(primitives.ValidatorIndex(j))
		if err != nil {
			return 0, err
		}
		if IsActiveValidatorUsingTrie(val, time.CurrentEpoch(st)) {
			lastActivatedvalidatorIndex = primitives.ValidatorIndex(j)
			break
		}
	}
	return lastActivatedvalidatorIndex, nil
}

// IsSameWithdrawalCredentials returns true if both validators have the same withdrawal credentials.
//
//	return a.withdrawal_credentials[12:] == b.withdrawal_credentials[12:]
func IsSameWithdrawalCredentials(a, b *ethpb.Validator) bool {
	if a == nil || b == nil {
		return false
	}
	if len(a.WithdrawalCredentials) <= 12 || len(b.WithdrawalCredentials) <= 12 {
		return false
	}
	return bytes.Equal(a.WithdrawalCredentials[12:], b.WithdrawalCredentials[12:])
}

// IsFullyWithdrawableValidator returns whether the validator is able to perform a full
// withdrawal. This function assumes that the caller holds a lock on the state.
//
// Spec definition:
//
//	def is_fully_withdrawable_validator(validator: Validator, balance: Gwei, epoch: Epoch) -> bool:
//	    """
//	    Check if ``validator`` is fully withdrawable.
//	    """
//		return validator.withdrawable_epoch <= epoch and balance > 0  # [Modified in Electra:EIP7251]
func IsFullyWithdrawableValidator(val state.ReadOnlyValidator, balance uint64, epoch primitives.Epoch, fork int) bool {
	if val == nil || balance <= 0 {
		return false
	}
	withdrawableEpoch := GetWithdrawableEpoch(val.ExitEpoch(), val.Slashed())
	return withdrawableEpoch <= epoch
}

// IsPartiallyWithdrawableValidator returns whether the validator is able to perform a
// partial withdrawal. This function assumes that the caller has a lock on the state.
// This method conditionally calls the fork appropriate implementation based on the epoch argument.
func IsPartiallyWithdrawableValidator(val state.ReadOnlyValidator, balance uint64, epoch primitives.Epoch, fork int) bool {
	if val == nil {
		return false
	}
	return IsPartiallyWithdrawableValidatorAlpaca(val, balance)
}

// IsPartiallyWithdrawableValidatorAlpaca implements is_partially_withdrawable_validator in the
// alpaca fork.
//
// Spec definition:
// def is_partially_withdrawable_validator(validator: Validator, balance: Gwei) -> bool:
//
//	"""
//	Check if “validator“ is partially withdrawable.
//	"""
//	has_excess_balance = balance > validator.principal_balance  # [Modified in Electra:EIP7251]
//	return has_excess_balance  # [Modified in Electra:EIP7251]
func IsPartiallyWithdrawableValidatorAlpaca(val state.ReadOnlyValidator, balance uint64) bool {
	hasExcessBalance := balance > val.PrincipalBalance()
	return hasExcessBalance
}

// GetWithdrawableEpoch returns the epoch at which the validator can withdraw.
//
// Spec definition:
//
//	def get_withdrawable_epoch(validator: Validator) -> Epoch:
//		if validator.exit_epoch == FAR_FUTURE_EPOCH:
//			return FAR_FUTURE_EPOCH
//		elif validator.slashed:
//			return Epoch(validator.exit_epoch + MIN_SLASHING_WITHDRAWABLE_DELAY)
//		return Epoch(validator.exit_epoch + MIN_VALIDATOR_WITHDRAWABILITY_DELAY)
func GetWithdrawableEpoch(exitEpoch primitives.Epoch, slashed bool) primitives.Epoch {
	beaconConfig := params.BeaconConfig()

	if exitEpoch == beaconConfig.FarFutureEpoch {
		return beaconConfig.FarFutureEpoch
	} else if slashed {
		return exitEpoch + beaconConfig.MinSlashingWithdrawableDelay
	}
	return exitEpoch + beaconConfig.MinValidatorWithdrawabilityDelay
}
