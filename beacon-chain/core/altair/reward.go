package altair

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

// BaseReward takes state and validator index and calculate
// individual validator's base reward.
//
// Spec code:
//
//	def get_base_reward(state: BeaconState, index: ValidatorIndex) -> Gwei:
//	  """
//	  Return the base reward for the validator defined by ``index`` with respect to the current ``state``.
//
//	  Note: An optimally performing validator can earn one base reward per epoch over a long time horizon.
//	  This takes into account both per-epoch (e.g. attestation) and intermittent duties (e.g. block proposal).
//	  """
//	  increments = state.validators[index].effective_balance // EFFECTIVE_BALANCE_INCREMENT
//	  return Gwei(increments * get_base_reward_per_increment(state))
func BaseReward(s state.ReadOnlyBeaconState, index primitives.ValidatorIndex) (uint64, error) {
	totalBalance, err := helpers.TotalActiveBalance(s)
	if err != nil {
		return 0, errors.Wrap(err, "could not calculate active balance")
	}
	baseReward, _, err := BaseRewardWithTotalBalance(s, index, totalBalance)
	return baseReward, err
}

// BaseRewardWithTotalBalance calculates the base reward with the provided total balance.
func BaseRewardWithTotalBalance(s state.ReadOnlyBeaconState, index primitives.ValidatorIndex, totalBalance uint64) (uint64, uint64, error) {
	val, err := s.ValidatorAtIndexReadOnly(index)
	if err != nil {
		return 0, 0, err
	}
	cfg := params.BeaconConfig()
	increments := val.EffectiveBalance() / cfg.EffectiveBalanceIncrement
	baseRewardPerInc, reserveUsagePerInc, err := BaseRewardPerIncrement(s, totalBalance)
	if err != nil {
		return 0, 0, err
	}
	return increments * baseRewardPerInc, increments * reserveUsagePerInc, nil
}

// BaseRewardPerIncrement of the beacon state
//
// Spec code:
// def get_base_reward_per_increment(epoch, activeBalance) -> Gwei:
//
//	return Gwei(Issuance_Per_Epoch * EffectiveBalanceIncrement // get_total_active_balance(state))
func BaseRewardPerIncrement(s state.ReadOnlyBeaconState, activeBalance uint64) (uint64, uint64, error) {
	if activeBalance == 0 {
		return 0, 0, errors.New("active balance can't be 0")
	}

	cfg := params.BeaconConfig()
	if activeBalance < cfg.EffectiveBalanceIncrement {
		return 0, 0, errors.New("active balance can't be lower than effective balance increment")
	}
	totalActiveIncrement := activeBalance / cfg.EffectiveBalanceIncrement
	totalReward, reserveUsage := helpers.TotalRewardWithReserveUsage(s)
	if reserveUsage == 0 {
		return totalReward / totalActiveIncrement, 0, nil
	}
	return totalReward / totalActiveIncrement, (reserveUsage / totalActiveIncrement) + 1, nil
}
