package precompute

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

type attesterRewardsFunc func(state.ReadOnlyBeaconState, *Balance, []*Validator) ([]uint64, []uint64, []uint64, error)
type proposerRewardsFunc func(state.ReadOnlyBeaconState, *Balance, []*Validator) ([]uint64, []uint64, error)

// ProcessRewardsAndPenaltiesPrecompute processes the rewards and penalties of individual validator.
// This is an optimized version by passing in precomputed validator attesting records and total epoch balances.
func ProcessRewardsAndPenaltiesPrecompute(
	state state.BeaconState,
	pBal *Balance,
	vp []*Validator,
	attRewardsFunc attesterRewardsFunc,
	proRewardsFunc proposerRewardsFunc,
) (state.BeaconState, error) {
	// Can't process rewards and penalties in genesis epoch.
	if time.CurrentEpoch(state) == 0 {
		return state, nil
	}

	numOfVals := state.NumValidators()
	// Guard against an out-of-bounds using validator balance precompute.
	if len(vp) != numOfVals || len(vp) != state.BalancesLength() {
		return state, errors.New("precomputed registries not the same length as state registries")
	}

	attsRewards, attsPenalties, attReserveUses, err := attRewardsFunc(state, pBal, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attester attestation delta")
	}
	proposerRewards, proposerReserveUses, err := proRewardsFunc(state, pBal, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not get proposer attestation delta")
	}
	validatorBals := state.Balances()
	reserveUsage := uint64(0)
	for i := 0; i < numOfVals; i++ {
		vp[i].BeforeEpochTransitionBalance = validatorBals[i]

		// Compute the post balance of the validator after accounting for the
		// attester and proposer rewards and penalties.
		validatorBals[i], err = helpers.IncreaseBalanceWithVal(validatorBals[i], attsRewards[i]+proposerRewards[i])
		if err != nil {
			return nil, err
		}
		validatorBals[i] = helpers.DecreaseBalanceWithVal(validatorBals[i], attsPenalties[i])

		vp[i].AfterEpochTransitionBalance = validatorBals[i]
		reserveUsage += attReserveUses[i] + proposerReserveUses[i]
	}

	if err := state.SetBalances(validatorBals); err != nil {
		return nil, errors.Wrap(err, "could not set validator balances")
	}
	if err := helpers.DecreaseReserves(state, reserveUsage); err != nil {
		return nil, errors.Wrap(err, "could not set current epoch reserve")
	}

	return state, nil
}

// AttestationsDelta computes and returns the rewards and penalties differences for individual validators based on the
// voting records.
func AttestationsDelta(state state.ReadOnlyBeaconState, pBal *Balance, vp []*Validator) ([]uint64, []uint64, []uint64, error) {
	numOfVals := state.NumValidators()
	rewards := make([]uint64, numOfVals)
	penalties := make([]uint64, numOfVals)
	reserveUsages := make([]uint64, numOfVals)
	prevEpoch := time.PrevEpoch(state)
	finalizedEpoch := state.FinalizedCheckpointEpoch()

	for i, v := range vp {
		rewards[i], penalties[i], reserveUsages[i] = attestationDelta(state, pBal, v, prevEpoch, finalizedEpoch)
	}
	return rewards, penalties, reserveUsages, nil
}

func attestationDelta(state state.ReadOnlyBeaconState, pBal *Balance, v *Validator, prevEpoch, finalizedEpoch primitives.Epoch) (uint64, uint64, uint64) {
	if !EligibleForRewards(v) || pBal.ActiveCurrentEpoch == 0 {
		return 0, 0, 0
	}

	baseRewardsPerEpoch := params.BeaconConfig().BaseRewardsPerEpoch
	effectiveBalanceIncrement := params.BeaconConfig().EffectiveBalanceIncrement
	currentEpochIncrement := pBal.ActiveCurrentEpoch / effectiveBalanceIncrement
	vb := v.CurrentEpochEffectiveBalance / effectiveBalanceIncrement
	totalReward, reserveUsage := helpers.TotalRewardWithReserveUsage(state)
	br := vb * totalReward / currentEpochIncrement / baseRewardsPerEpoch
	r, p := uint64(0), uint64(0)

	// Process source reward / penalty
	if v.IsPrevEpochAttester && !v.IsSlashed {
		proposerReward := br / params.BeaconConfig().ProposerRewardQuotient
		maxAttesterReward := br - proposerReward
		r += maxAttesterReward / uint64(v.InclusionDistance)

		if helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch) {
			// Since full base reward will be canceled out by inactivity penalty deltas,
			// optimal participation receives full base reward compensation here.
			r += br
		} else {
			rewardNumerator := br * (pBal.PrevEpochAttested / effectiveBalanceIncrement)
			r += rewardNumerator / currentEpochIncrement
		}
	} else {
		p += br
	}

	// Process target reward / penalty
	if v.IsPrevEpochTargetAttester && !v.IsSlashed {
		if helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch) {
			// Since full base reward will be canceled out by inactivity penalty deltas,
			// optimal participation receives full base reward compensation here.
			r += br
		} else {
			rewardNumerator := br * (pBal.PrevEpochTargetAttested / effectiveBalanceIncrement)
			r += rewardNumerator / currentEpochIncrement
		}
	} else {
		p += br
	}

	// Process head reward / penalty
	if v.IsPrevEpochHeadAttester && !v.IsSlashed {
		if helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch) {
			// Since full base reward will be canceled out by inactivity penalty deltas,
			// optimal participation receives full base reward compensation here.
			r += br
		} else {
			rewardNumerator := br * (pBal.PrevEpochHeadAttested / effectiveBalanceIncrement)
			r += rewardNumerator / currentEpochIncrement
		}
	} else {
		p += br
	}

	// Process finality delay penalty
	if helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch) {
		// If validator is performing optimally, this cancels all rewards for a neutral balance.
		proposerReward := br / params.BeaconConfig().ProposerRewardQuotient
		p += baseRewardsPerEpoch*br - proposerReward
		// Apply an additional penalty to validators that did not vote on the correct target or has been slashed.
		// Equivalent to the following condition from the spec:
		// `index not in get_unslashed_attesting_indices(state, matching_target_attestations)`
		if !v.IsPrevEpochTargetAttester || v.IsSlashed {
			finalityDelay := helpers.FinalityDelay(prevEpoch, finalizedEpoch)
			p += vb * uint64(finalityDelay) / params.BeaconConfig().InactivityPenaltyQuotient
		}
	}

	ru := r * reserveUsage / totalReward
	return r, p, ru
}

// ProposersDelta computes and returns the rewards and penalties differences for individual validators based on the
// proposer inclusion records.
func ProposersDelta(state state.ReadOnlyBeaconState, pBal *Balance, vp []*Validator) ([]uint64, []uint64, error) {
	numofVals := state.NumValidators()
	rewards := make([]uint64, numofVals)
	reserveUsages := make([]uint64, numofVals)

	baseRewardsPerEpoch := params.BeaconConfig().BaseRewardsPerEpoch
	proposerRewardQuotient := params.BeaconConfig().ProposerRewardQuotient
	effectiveBalanceIncrement := params.BeaconConfig().EffectiveBalanceIncrement
	totalReward, reserveUsage := helpers.TotalRewardWithReserveUsage(state)
	for _, v := range vp {
		if uint64(v.ProposerIndex) >= uint64(len(rewards)) {
			// This should never happen with a valid state / validator.
			return nil, nil, errors.New("proposer index out of range")
		}
		// Only apply inclusion rewards to proposer only if the attested hasn't been slashed.
		if v.IsPrevEpochAttester && !v.IsSlashed {
			currentEpochIncrement := pBal.ActiveCurrentEpoch / effectiveBalanceIncrement
			vBalanceIncrement := v.CurrentEpochEffectiveBalance / effectiveBalanceIncrement
			baseReward := vBalanceIncrement * totalReward / currentEpochIncrement / baseRewardsPerEpoch
			proposerReward := baseReward / proposerRewardQuotient
			rewards[v.ProposerIndex] += proposerReward
			reserveUsages[v.ProposerIndex] += proposerReward * reserveUsage / totalReward
		}
	}
	return rewards, reserveUsages, nil
}

// EligibleForRewards for validator.
//
// Spec code:
// if is_active_validator(v, previous_epoch) or (v.slashed and previous_epoch + 1 < v.withdrawable_epoch)
func EligibleForRewards(v *Validator) bool {
	return v.IsActivePrevEpoch || (v.IsSlashed && !v.IsWithdrawableCurrentEpoch)
}
