package helpers

import (
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	mathutil "github.com/prysmaticlabs/prysm/v5/math"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

var balanceCache = cache.NewEffectiveBalanceCache()
var balanceWithQueueCache = cache.NewEffectiveBalanceCache()

// TotalBalance returns the total amount at stake in Gwei
// of input validators.
//
// Spec pseudocode definition:
//
//	def get_total_balance(state: BeaconState, indices: Set[ValidatorIndex]) -> Gwei:
//	 """
//	 Return the combined effective balance of the ``indices``.
//	 ``EFFECTIVE_BALANCE_INCREMENT`` Gwei minimum to avoid divisions by zero.
//	 Math safe up to ~10B ETH, after which this overflows uint64.
//	 """
//	 return Gwei(max(EFFECTIVE_BALANCE_INCREMENT, sum([state.validators[index].effective_balance for index in indices])))
func TotalBalance(state state.ReadOnlyValidators, indices []primitives.ValidatorIndex) uint64 {
	total := uint64(0)

	for _, idx := range indices {
		val, err := state.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			continue
		}
		total += val.EffectiveBalance()
	}

	// EFFECTIVE_BALANCE_INCREMENT is the lower bound for total balance.
	if total < params.BeaconConfig().EffectiveBalanceIncrement {
		return params.BeaconConfig().EffectiveBalanceIncrement
	}

	return total
}

// TotalActiveBalance returns the total amount at stake in Gwei
// of active validators.
//
// Spec pseudocode definition:
//
//	def get_total_active_balance(state: BeaconState) -> Gwei:
//	 """
//	 Return the combined effective balance of the active validators.
//	 Note: ``get_total_balance`` returns ``EFFECTIVE_BALANCE_INCREMENT`` Gwei minimum to avoid divisions by zero.
//	 """
//	 return get_total_balance(state, set(get_active_validator_indices(state, get_current_epoch(state))))
func TotalActiveBalance(s state.ReadOnlyBeaconState) (uint64, error) {
	bal, err := balanceCache.Get(s)
	switch {
	case err == nil:
		return bal, nil
	case errors.Is(err, cache.ErrNotFound):
		// Do nothing if we receive a not found error.
	default:
		// In the event, we encounter another error we return it.
		return 0, err
	}

	total := uint64(0)
	epoch := slots.ToEpoch(s.Slot())
	if err := s.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		if IsActiveValidatorUsingTrie(val, epoch) {
			total += val.EffectiveBalance()
		}
		return nil
	}); err != nil {
		return 0, err
	}

	// Spec defines `EffectiveBalanceIncrement` as min to avoid divisions by zero.
	total = mathutil.Max(params.BeaconConfig().EffectiveBalanceIncrement, total)
	if err := balanceCache.AddTotalEffectiveBalance(s, total); err != nil {
		return 0, err
	}

	return total, nil
}

// TotalBalanceWithQueue returns the total amount at stake in Gwei
// of active validators + pending validators - exiting validators.
func TotalBalanceWithQueue(s state.ReadOnlyBeaconState) (uint64, error) {
	bal, err := balanceWithQueueCache.Get(s)
	switch {
	case err == nil:
		return bal, nil
	case errors.Is(err, cache.ErrNotFound):
		// Do nothing if we receive a not found error.
	default:
		// In the event, we encounter another error we return it.
		return 0, err
	}

	total := uint64(0)
	totalExit := uint64(0)
	epoch := slots.ToEpoch(s.Slot())
	if err := s.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		if IsPendingValidatorUsingTrie(val, epoch) {
			total += val.EffectiveBalance()
		} else if IsActiveValidatorUsingTrie(val, epoch) {
			total += val.EffectiveBalance()
		}
		if IsExitingValidatorUsingTrie(val, epoch) {
			totalExit += val.EffectiveBalance()
		}
		return nil
	}); err != nil {
		return 0, err
	}
	if total > totalExit {
		total -= totalExit
	} else {
		total = 0
	}

	// Spec defines `EffectiveBalanceIncrement` as min to avoid divisions by zero.
	total = mathutil.Max(params.BeaconConfig().EffectiveBalanceIncrement, total)
	if err := balanceWithQueueCache.AddTotalEffectiveBalance(s, total); err != nil {
		return 0, err
	}

	return total, nil
}

// IncreaseBalance increases validator with the given 'index' balance by 'delta' in Gwei.
//
// Spec pseudocode definition:
//
//	def increase_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//	  """
//	  Increase the validator balance at index ``index`` by ``delta``.
//	  """
//	  state.balances[index] += delta
func IncreaseBalance(state state.BeaconState, idx primitives.ValidatorIndex, delta uint64) error {
	balAtIdx, err := state.BalanceAtIndex(idx)
	if err != nil {
		return err
	}
	newBal, err := IncreaseBalanceWithVal(balAtIdx, delta)
	if err != nil {
		return err
	}
	return state.UpdateBalancesAtIndex(idx, newBal)
}

// IncreaseBalanceAndAdjustPrincipalBalance increase validator with the given 'index' balance by 'delta' in Gwei
// and adjust the principal balance.
//
// Spec pseudocode definition:
//
// def increase_balance_and_adjust_principal_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//
//	increase_balance(state, index, delta)
//
//	validator = state.validators[index]
//
//	if state.balances[index] >= validator.principal_balance + delta:
//
//		validator.principal_balance += delta
//
//	elif state.balances[index] >= validator.principal_balance:
//
//		validator.principal_balance = state.balances[index]
func IncreaseBalanceAndAdjustPrincipalBalance(state state.BeaconState, idx primitives.ValidatorIndex, delta uint64) error {
	prevBalance, err := state.BalanceAtIndex(idx)
	if err != nil {
		return err
	}
	newBal, err := IncreaseBalanceWithVal(prevBalance, delta)
	if err != nil {
		return err
	}

	validator, err := state.ValidatorAtIndex(idx)
	if err != nil {
		return err
	}

	pbUpdated := false
	if newBal >= validator.PrincipalBalance+delta {
		validator.PrincipalBalance += delta
		pbUpdated = true
	} else if newBal >= validator.PrincipalBalance {
		validator.PrincipalBalance = newBal
		pbUpdated = true
	}

	// Update the balance in the state
	if err := state.UpdateBalancesAtIndex(idx, newBal); err != nil {
		return err
	}

	// If principal balance was updated, update the validator as well
	if pbUpdated {
		if err := state.UpdateValidatorAtIndex(idx, validator); err != nil {
			// Rollback balance if validator update fails
			if rollbackErr := state.UpdateBalancesAtIndex(idx, prevBalance); rollbackErr != nil {
				// If rollback fails, log or handle it as needed, but return the original error
				return fmt.Errorf("validator update failed: %w, and rollback failed: %v", err, rollbackErr)
			}
			return err
		}
	}

	return nil
}

// IncreaseBalanceWithVal increases validator with the given 'index' balance by 'delta' in Gwei.
// This method is flattened version of the spec method, taking in the raw balance and returning
// the post balance.
//
// Spec pseudocode definition:
//
//	def increase_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//	  """
//	  Increase the validator balance at index ``index`` by ``delta``.
//	  """
//	  state.balances[index] += delta
func IncreaseBalanceWithVal(currBalance, delta uint64) (uint64, error) {
	return mathutil.Add64(currBalance, delta)
}

// DecreaseBalance decreases validator with the given 'index' balance by 'delta' in Gwei.
//
// Spec pseudocode definition:
//
//	def decrease_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//	  """
//	  Decrease the validator balance at index ``index`` by ``delta``, with underflow protection.
//	  """
//	  state.balances[index] = 0 if delta > state.balances[index] else state.balances[index] - delta
func DecreaseBalance(state state.BeaconState, idx primitives.ValidatorIndex, delta uint64) error {
	balAtIdx, err := state.BalanceAtIndex(idx)
	if err != nil {
		return err
	}
	return state.UpdateBalancesAtIndex(idx, DecreaseBalanceWithVal(balAtIdx, delta))
}

// DecreaseBalanceAndAdjustPrincipalBalance decreases validator with the given 'index' balance by 'delta' in Gwei.
// and adjust the principal balance.
//
// def decrease_balance_and_adjust_principal_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//
//	prev_balance = state.balances[index]
//	decrease_balance(state, index, delta)
//
//	validator = state.validators[index]
//	if prev_balance >= MIN_ACTIVATION_BALANCE:
//		validator.principal_balance = max(validator.principal_balance * (state.balances[index] / prev_balance), MIN_ACTIVATION_BALANCE)
//	elif validator.principal_balance != MIN_ACTIVATION_BALANCE:
//		validator.principal_balance = MIN_ACTIVATION_BALANCE
func DecreaseBalanceAndAdjustPrincipalBalance(state state.BeaconState, idx primitives.ValidatorIndex, delta uint64) error {
	prevBalance, err := state.BalanceAtIndex(idx)
	if err != nil {
		return err
	}

	newBal := DecreaseBalanceWithVal(prevBalance, delta)

	validator, err := state.ValidatorAtIndex(idx)
	if err != nil {
		return err
	}

	pbUpdated := false
	if prevBalance >= params.BeaconConfig().MinActivationBalance {
		validator.PrincipalBalance = max(validator.PrincipalBalance*(newBal/prevBalance), params.BeaconConfig().MinActivationBalance)
		pbUpdated = true
	} else if validator.PrincipalBalance != params.BeaconConfig().MinActivationBalance {
		validator.PrincipalBalance = params.BeaconConfig().MinActivationBalance
		pbUpdated = true
	}

	// Update the balance in the state
	if err := state.UpdateBalancesAtIndex(idx, newBal); err != nil {
		return err
	}

	// If principal balance was updated, update the validator as well
	if pbUpdated {
		if err := state.UpdateValidatorAtIndex(idx, validator); err != nil {
			// Rollback balance if validator update fails
			if rollbackErr := state.UpdateBalancesAtIndex(idx, prevBalance); rollbackErr != nil {
				// If rollback fails, log or handle it as needed, but return the original error
				return rollbackErr
			}
			return err
		}
	}

	return nil
}

// DecreaseBalanceWithVal decreases validator with the given 'index' balance by 'delta' in Gwei.
// This method is flattened version of the spec method, taking in the raw balance and returning
// the post balance.
//
// Spec pseudocode definition:
//
//	def decrease_balance(state: BeaconState, index: ValidatorIndex, delta: Gwei) -> None:
//	  """
//	  Decrease the validator balance at index ``index`` by ``delta``, with underflow protection.
//	  """
//	  state.balances[index] = 0 if delta > state.balances[index] else state.balances[index] - delta
func DecreaseBalanceWithVal(currBalance, delta uint64) uint64 {
	if delta > currBalance {
		return 0
	}
	return currBalance - delta
}

// IsInInactivityLeak returns true if the state is experiencing inactivity leak.
//
// Spec code:
// def is_in_inactivity_leak(state: BeaconState) -> bool:
//
//	return get_finality_delay(state) > MIN_EPOCHS_TO_INACTIVITY_PENALTY
func IsInInactivityLeak(prevEpoch, finalizedEpoch primitives.Epoch) bool {
	return FinalityDelay(prevEpoch, finalizedEpoch) > params.BeaconConfig().MinEpochsToInactivityPenalty
}

// FinalityDelay returns the finality delay using the beacon state.
//
// Spec code:
// def get_finality_delay(state: BeaconState) -> uint64:
//
//	return get_previous_epoch(state) - state.finalized_checkpoint.epoch
func FinalityDelay(prevEpoch, finalizedEpoch primitives.Epoch) primitives.Epoch {
	return prevEpoch - finalizedEpoch
}

// EpochIssuance returns the total amount of OVER(in Gwei) to be issued in the given epoch.
func EpochIssuance(epoch primitives.Epoch) uint64 {
	cfg := params.BeaconConfig()
	year := EpochToYear(epoch)

	// After year 10, no more issuance left.
	if year >= len(cfg.IssuanceRate) {
		year = len(cfg.IssuanceRate) - 1
	}
	return cfg.MaxTokenSupply / cfg.IssuancePrecision * cfg.IssuanceRate[year] / cfg.EpochsPerYear
}

// TargetDepositPlan returns the target deposit plan for the given epoch.
func TargetDepositPlan(epoch primitives.Epoch) uint64 {
	cfg := params.BeaconConfig()
	e := uint64(epoch)
	if e < cfg.EpochsPerYear*cfg.DepositPlanEarlyEnd {
		return cfg.DepositPlanEarlySlope*e + cfg.DepositPlanEarlyOffset
	} else if e < cfg.EpochsPerYear*cfg.DepositPlanLaterEnd {
		return cfg.DepositPlanLaterSlope*e + cfg.DepositPlanLaterOffset
	} else {
		return cfg.DepositPlanFinal
	}
}

// TotalRewardWithReserveUsage returns the total reward and total reserve usage in the given epoch.
// Reserve is always depleted in Over tokenomics.
func TotalRewardWithReserveUsage(s state.ReadOnlyBeaconState) (uint64, uint64) {
	epochIssuance := EpochIssuance(time.CurrentEpoch(s))
	feedbackBoost := EpochFeedbackBoost(s)

	return epochIssuance + feedbackBoost, feedbackBoost
}

// EpochFeedbackBoost returns the boost reward from feedback model in tokenomics.
func EpochFeedbackBoost(s state.ReadOnlyBeaconState) uint64 {
	cfg := params.BeaconConfig()
	rewardAdjustmentFactor := s.RewardAdjustmentFactor()
	feedbackBoost := cfg.MaxTokenSupply / cfg.RewardAdjustmentFactorPrecision * rewardAdjustmentFactor / cfg.EpochsPerYear

	if reserves := s.Reserves(); feedbackBoost > reserves {
		return reserves
	}
	return feedbackBoost
}

// ProcessRewardAdjustmentFactor sets the adjustment factor for the next epoch.
// This is pseudo code from the spec.
//
// Spec code:
//
// def process_reward_adjustment_factor(state: BeaconState) -> None:
// _, future_total_active_balance = get_balance_with_queue(state)
// target_deposit = get_target_deposit(state)
//
// if future_total_active_balance > target_deposit:
// decrease_reward_adjustment_factor(state, REWARD_ADJUSTMENT_FACTOR_DELTA)
// elif future_total_active_balance < target_deposit:
// increase_reward_adjustment_factor(state, REWARD_ADJUSTMENT_FACTOR_DELTA)
func ProcessRewardAdjustmentFactor(state state.BeaconState) (state.BeaconState, error) {
	futureDeposit, err := TotalBalanceWithQueue(state)
	if err != nil {
		return nil, err
	}
	targetDeposit := TargetDepositPlan(time.NextEpoch(state))

	if futureDeposit > targetDeposit {
		err = DecreaseRewardAdjustmentFactor(state)
		if err != nil {
			return nil, err
		}
	} else if futureDeposit < targetDeposit {
		err = IncreaseRewardAdjustmentFactor(state)
		if err != nil {
			return nil, err
		}
	}

	return state, nil
}

// DecreaseRewardAdjustmentFactor reduces the RewardAdjustmentFactor with fixed amount.
// If the RewardAdjustmentFactor is less than the given amount, it sets the RewardAdjustmentFactor to 0.
func DecreaseRewardAdjustmentFactor(state state.BeaconState) error {
	delta := params.BeaconConfig().RewardAdjustmentFactorDelta
	factor := state.RewardAdjustmentFactor()
	if factor < delta {
		err := state.SetRewardAdjustmentFactor(0)
		if err != nil {
			return err
		}
	} else {
		err := state.SetRewardAdjustmentFactor(factor - delta)
		if err != nil {
			return err
		}
	}

	return nil
}

// IncreaseRewardAdjustmentFactor increases the RewardAdjustmentFactor with fixed amount.
// If the RewardAdjustmentFactor is larger than the MaxRewardAdjustmentFactors[year],
// it sets the RewardAdjustmentFactor to MaxRewardAdjustmentFactors[year].
func IncreaseRewardAdjustmentFactor(state state.BeaconState) error {
	epoch := slots.ToEpoch(state.Slot())
	newFactor := state.RewardAdjustmentFactor() + params.BeaconConfig().RewardAdjustmentFactorDelta

	maxBoostYield := MaxRewardAdjustmentFactor(epoch)
	if maxBoostYield < newFactor {
		err := state.SetRewardAdjustmentFactor(maxBoostYield)
		if err != nil {
			return err
		}
	} else {
		err := state.SetRewardAdjustmentFactor(newFactor)
		if err != nil {
			return err
		}
	}

	return nil
}

// MaxRewardAdjustmentFactor gets the maximum reward adjustment factor of corresponding year for the given epoch.
func MaxRewardAdjustmentFactor(epoch primitives.Epoch) uint64 {
	cfg := params.BeaconConfig()
	year := EpochToYear(epoch)
	if year >= len(cfg.MaxRewardAdjustmentFactors) {
		year = len(cfg.MaxRewardAdjustmentFactors) - 1
	}

	return cfg.MaxRewardAdjustmentFactors[year]
}

// EpochToYear converts an epoch to a year.
func EpochToYear(epoch primitives.Epoch) int {
	cfg := params.BeaconConfig()
	year := int(epoch.Div(cfg.EpochsPerYear)) // lint:ignore uintcast -- Time is always positive.
	return year
}

// DecreaseReserves reduces the reserve by the given amount.
// If the reserve is less than the given amount, it sets the reserve to 0.
func DecreaseReserves(state state.BeaconState, delta uint64) error {
	reserve := state.Reserves()
	if reserve < delta {
		err := state.SetReserves(0)
		if err != nil {
			return err
		}
	} else {
		err := state.SetReserves(reserve - delta)
		if err != nil {
			return err
		}
	}
	return nil
}
