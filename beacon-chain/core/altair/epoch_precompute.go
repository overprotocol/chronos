package altair

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/math"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
)

// AttDelta contains rewards and penalties for a single attestation.
type AttDelta struct {
	HeadReward        uint64
	SourceReward      uint64
	SourcePenalty     uint64
	TargetReward      uint64
	TargetPenalty     uint64
	InactivityPenalty uint64
}

// InitializePrecomputeValidators precomputes individual validator for its attested balances and the total sum of validators attested balances of the epoch.
func InitializePrecomputeValidators(ctx context.Context, beaconState state.BeaconState) ([]*precompute.Validator, *precompute.Balance, error) {
	_, span := trace.StartSpan(ctx, "altair.InitializePrecomputeValidators")
	defer span.End()
	vals := make([]*precompute.Validator, beaconState.NumValidators())
	bal := &precompute.Balance{}
	prevEpoch := time.PrevEpoch(beaconState)
	currentEpoch := time.CurrentEpoch(beaconState)
	inactivityScores, err := beaconState.InactivityScores()
	if err != nil {
		return nil, nil, err
	}
	bailoutScores, err := beaconState.BailOutScores()
	if err != nil {
		return nil, nil, err
	}

	// This shouldn't happen with a correct beacon state,
	// but rather be safe to defend against index out of bound panics.
	if beaconState.NumValidators() != len(inactivityScores) {
		return nil, nil, errors.New("num of validators is different than num of inactivity scores")
	}
	if beaconState.NumValidators() != len(bailoutScores) {
		return nil, nil, errors.New("num of validators is different than num of bail out scores")
	}

	if err := beaconState.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		// Set validator's balance, inactivity score and slashed/withdrawable status.
		v := &precompute.Validator{
			CurrentEpochEffectiveBalance: val.EffectiveBalance(),
			InactivityScore:              inactivityScores[idx],
			BailOutScore:                 bailoutScores[idx],
			IsSlashed:                    val.Slashed(),
			IsWithdrawableCurrentEpoch:   currentEpoch >= val.WithdrawableEpoch(),
			IsWaitingForExit:             val.ExitEpoch() != params.BeaconConfig().FarFutureEpoch && currentEpoch < val.ExitEpoch(),
		}
		// Set validator's active status for current epoch.
		if helpers.IsActiveValidatorUsingTrie(val, currentEpoch) {
			v.IsActiveCurrentEpoch = true
			bal.ActiveCurrentEpoch, err = math.Add64(bal.ActiveCurrentEpoch, val.EffectiveBalance())
			if err != nil {
				return err
			}
		}
		// Set validator's active status for previous epoch.
		if helpers.IsActiveValidatorUsingTrie(val, prevEpoch) {
			v.IsActivePrevEpoch = true
			bal.ActivePrevEpoch, err = math.Add64(bal.ActivePrevEpoch, val.EffectiveBalance())
			if err != nil {
				return err
			}
		}
		vals[idx] = v
		return nil
	}); err != nil {
		return nil, nil, errors.Wrap(err, "could not read every validator")
	}
	return vals, bal, nil
}

// ProcessInactivityAndBailOutScores of beacon chain. This updates inactivity scores of beacon chain and
// updates the precompute validator struct for later processing. The inactivity scores work as following:
// For fully inactive validators and perfect active validators, the effect is the same as before Altair.
// For a validator is inactive and the chain fails to finalize, the inactivity score increases by a fixed number, the total loss after N epochs is proportional to N**2/2.
// For imperfectly active validators. The inactivity score's behavior is specified by this function:
//
//	If a validator fails to submit an attestation with the correct target, their inactivity score goes up by 4.
//	If they successfully submit an attestation with the correct source and target, their inactivity score drops by 1
//	If the chain has recently finalized, each validator's score drops by 16.
func ProcessInactivityAndBailOutScores(
	ctx context.Context,
	beaconState state.BeaconState,
	vals []*precompute.Validator,
) (state.BeaconState, []*precompute.Validator, error) {
	_, span := trace.StartSpan(ctx, "altair.ProcessInactivityAndBailOutScores")
	defer span.End()

	cfg := params.BeaconConfig()
	if time.CurrentEpoch(beaconState) == cfg.GenesisEpoch {
		return beaconState, vals, nil
	}

	inactivityScores, err := beaconState.InactivityScores()
	if err != nil {
		return nil, nil, err
	}
	bailoutScores, err := beaconState.BailOutScores()
	if err != nil {
		return nil, nil, err
	}

	bias := cfg.InactivityScoreBias
	recoveryRate := cfg.InactivityScoreRecoveryRate
	prevEpoch := time.PrevEpoch(beaconState)
	finalizedEpoch := beaconState.FinalizedCheckpointEpoch()
	recovery := helpers.BailOutRecoveryScore(len(vals))
	for i, v := range vals {
		if !precompute.EligibleForRewards(v) {
			continue
		}

		isUpdated := false
		if v.IsPrevEpochTargetAttester && !v.IsSlashed {
			// Decrease inactivity score when validator gets target correct.
			if v.InactivityScore > 0 {
				v.InactivityScore -= 1
			}
			// Decrease bailout score when validator gets target correct.
			if v.BailOutScore > recovery && v.BailOutScore < cfg.BailOutScoreThreshold &&
				!v.IsWaitingForExit {
				v.BailOutScore, err = math.Sub64(v.BailOutScore, recovery)
				if err != nil {
					return nil, nil, err
				}
			}
		} else {
			v.InactivityScore, err = math.Add64(v.InactivityScore, bias)
			if err != nil {
				return nil, nil, err
			}
			if !v.IsWaitingForExit && v.IsActivePrevEpoch && v.BailOutScore < cfg.BailOutScoreThreshold {
				v.BailOutScore, err = math.Add64(v.BailOutScore, cfg.BailOutScoreBias)
				if err != nil {
					return nil, nil, err
				}
				isUpdated = true
			}
		}

		if !v.IsWaitingForExit && !isUpdated && v.BailOutScore >= cfg.BailOutScoreThreshold && v.BailOutScore < math.MaxUint64-cfg.BailOutScoreBias {
			v.BailOutScore, err = math.Add64(v.BailOutScore, cfg.BailOutScoreBias)
			if err != nil {
				return nil, nil, err
			}
		}

		if !helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch) {
			score := recoveryRate
			// Prevents underflow below 0.
			if score > v.InactivityScore {
				score = v.InactivityScore
			}
			v.InactivityScore -= score
		}
		inactivityScores[i] = v.InactivityScore
		bailoutScores[i] = v.BailOutScore
	}

	if err := beaconState.SetInactivityScores(inactivityScores); err != nil {
		return nil, nil, err
	}
	if err := beaconState.SetBailOutScores(bailoutScores); err != nil {
		return nil, nil, err
	}

	return beaconState, vals, nil
}

// ProcessEpochParticipation processes the epoch participation in state and updates individual validator's pre computes,
// it also tracks and updates epoch attesting balances.
// Spec code:
// if epoch == get_current_epoch(state):
//
//	    epoch_participation = state.current_epoch_participation
//	else:
//	    epoch_participation = state.previous_epoch_participation
//	active_validator_indices = get_active_validator_indices(state, epoch)
//	participating_indices = [i for i in active_validator_indices if has_flag(epoch_participation[i], flag_index)]
//	return set(filter(lambda index: not state.validators[index].slashed, participating_indices))
func ProcessEpochParticipation(
	ctx context.Context,
	beaconState state.BeaconState,
	bal *precompute.Balance,
	vals []*precompute.Validator,
) ([]*precompute.Validator, *precompute.Balance, error) {
	_, span := trace.StartSpan(ctx, "altair.ProcessEpochParticipation")
	defer span.End()

	cp, err := beaconState.CurrentEpochParticipation()
	if err != nil {
		return nil, nil, err
	}
	cfg := params.BeaconConfig()
	targetIdx := cfg.TimelyTargetFlagIndex
	sourceIdx := cfg.TimelySourceFlagIndex
	headIdx := cfg.TimelyHeadFlagIndex
	for i, b := range cp {
		has, err := HasValidatorFlag(b, sourceIdx)
		if err != nil {
			return nil, nil, err
		}
		if has && vals[i].IsActiveCurrentEpoch {
			vals[i].IsCurrentEpochAttester = true
		}
		has, err = HasValidatorFlag(b, targetIdx)
		if err != nil {
			return nil, nil, err
		}
		if has && vals[i].IsActiveCurrentEpoch {
			vals[i].IsCurrentEpochAttester = true
			vals[i].IsCurrentEpochTargetAttester = true
		}
	}
	pp, err := beaconState.PreviousEpochParticipation()
	if err != nil {
		return nil, nil, err
	}
	for i, b := range pp {
		has, err := HasValidatorFlag(b, sourceIdx)
		if err != nil {
			return nil, nil, err
		}
		if has && vals[i].IsActivePrevEpoch {
			vals[i].IsPrevEpochAttester = true
			vals[i].IsPrevEpochSourceAttester = true
		}
		has, err = HasValidatorFlag(b, targetIdx)
		if err != nil {
			return nil, nil, err
		}
		if has && vals[i].IsActivePrevEpoch {
			vals[i].IsPrevEpochAttester = true
			vals[i].IsPrevEpochTargetAttester = true
		}
		has, err = HasValidatorFlag(b, headIdx)
		if err != nil {
			return nil, nil, err
		}
		if has && vals[i].IsActivePrevEpoch {
			vals[i].IsPrevEpochHeadAttester = true
		}
	}
	bal = precompute.UpdateBalance(vals, bal, beaconState.Version())
	return vals, bal, nil
}

// ProcessRewardsAndPenaltiesPrecompute processes the rewards and penalties of individual validator.
// This is an optimized version by passing in precomputed validator attesting records and total epoch balances.
func ProcessRewardsAndPenaltiesPrecompute(
	beaconState state.BeaconState,
	bal *precompute.Balance,
	vals []*precompute.Validator,
) (state.BeaconState, error) {
	// Don't process rewards and penalties in genesis epoch.
	cfg := params.BeaconConfig()
	if time.CurrentEpoch(beaconState) == cfg.GenesisEpoch {
		return beaconState, nil
	}

	numOfVals := beaconState.NumValidators()
	// Guard against an out-of-bounds using validator balance precompute.
	if len(vals) != numOfVals || len(vals) != beaconState.BalancesLength() {
		return beaconState, errors.New("validator registries not the same length as state's validator registries")
	}

	attDeltas, attReserves, err := AttestationsDelta(beaconState, bal, vals)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}

	balances := beaconState.Balances()
	reserveUsage := uint64(0)
	for i := 0; i < numOfVals; i++ {
		vals[i].BeforeEpochTransitionBalance = balances[i]

		// Compute the post balance of the validator after accounting for the
		// attester and proposer rewards and penalties.
		delta := attDeltas[i]
		balances[i], err = helpers.IncreaseBalanceWithVal(balances[i], delta.HeadReward+delta.SourceReward+delta.TargetReward)
		if err != nil {
			return nil, err
		}
		balances[i] = helpers.DecreaseBalanceWithVal(balances[i], delta.SourcePenalty+delta.TargetPenalty+delta.InactivityPenalty)

		vals[i].AfterEpochTransitionBalance = balances[i]
		reserveUsage += attReserves[i]
	}

	if err := beaconState.SetBalances(balances); err != nil {
		return nil, errors.Wrap(err, "could not set validator balances")
	}

	if err := helpers.DecreaseCurrentReserve(beaconState, reserveUsage); err != nil {
		return nil, errors.Wrap(err, "could not set current epoch reserve")
	}

	return beaconState, nil
}

// AttestationsDelta computes and returns the rewards and penalties differences for individual validators based on the
// voting records.
func AttestationsDelta(beaconState state.BeaconState, bal *precompute.Balance, vals []*precompute.Validator) ([]*AttDelta, []uint64, error) {
	attDeltas := make([]*AttDelta, len(vals))
	attReserveDeltas := make([]uint64, len(vals))

	cfg := params.BeaconConfig()
	prevEpoch := time.PrevEpoch(beaconState)
	finalizedEpoch := beaconState.FinalizedCheckpointEpoch()
	baseRewardPerIncrement, reserveUsagePerIncrement, err := BaseRewardPerIncrement(beaconState, bal.ActiveCurrentEpoch)
	if err != nil {
		return nil, nil, err
	}
	leak := helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch)

	// Modified in Altair and Bellatrix.
	bias := cfg.InactivityScoreBias
	inactivityPenaltyQuotient, err := beaconState.InactivityPenaltyQuotient()
	if err != nil {
		return nil, nil, err
	}
	inactivityDenominator := bias * inactivityPenaltyQuotient

	for i, v := range vals {
		attDeltas[i], err = attestationDelta(bal, v, baseRewardPerIncrement, inactivityDenominator, leak)
		if err != nil {
			return nil, nil, err
		}
		attReserveDeltas[i] = (attDeltas[i].SourceReward + attDeltas[i].TargetReward + attDeltas[i].HeadReward) * reserveUsagePerIncrement / baseRewardPerIncrement
	}

	return attDeltas, attReserveDeltas, nil
}

func attestationDelta(
	bal *precompute.Balance,
	val *precompute.Validator,
	baseRewardPerIncrement, inactivityDenominator uint64,
	inactivityLeak bool) (*AttDelta, error) {
	eligible := val.IsActivePrevEpoch || (val.IsSlashed && !val.IsWithdrawableCurrentEpoch)
	// Per spec `ActiveCurrentEpoch` can't be 0 to process attestation delta.
	if !eligible || bal.ActiveCurrentEpoch == 0 {
		return &AttDelta{}, nil
	}

	cfg := params.BeaconConfig()
	increment := cfg.EffectiveBalanceIncrement
	effectiveBalance := val.CurrentEpochEffectiveBalance
	baseReward := (effectiveBalance / increment) * baseRewardPerIncrement
	activeIncrement := bal.ActiveCurrentEpoch / increment

	weightDenominator := cfg.WeightDenominator
	srcWeight := cfg.TimelySourceWeight
	tgtWeight := cfg.TimelyTargetWeight
	headWeight := cfg.TimelyHeadWeight
	lightLayerWeight := cfg.LightLayerWeight
	attDelta := &AttDelta{}
	// Process source reward / penalty
	if val.IsPrevEpochSourceAttester && !val.IsSlashed {
		if !inactivityLeak {
			n := baseReward * srcWeight * (bal.PrevEpochAttested / increment)
			attDelta.SourceReward += n / (activeIncrement * weightDenominator)
		}
	} else {
		attDelta.SourcePenalty += baseReward * srcWeight / weightDenominator
	}

	// Process target reward / penalty
	if val.IsPrevEpochTargetAttester && !val.IsSlashed {
		if !inactivityLeak {
			n := baseReward * tgtWeight * (bal.PrevEpochTargetAttested / increment)
			attDelta.TargetReward += n / (activeIncrement * weightDenominator)
		}
	} else {
		attDelta.TargetPenalty += baseReward * tgtWeight / weightDenominator
	}

	// Process head reward / penalty with light layer reward
	if val.IsPrevEpochHeadAttester && !val.IsSlashed {
		if !inactivityLeak {
			n := baseReward * (headWeight + lightLayerWeight) * (bal.PrevEpochHeadAttested / increment)
			attDelta.HeadReward += n / (activeIncrement * weightDenominator)
		}
	}

	// Process finality delay penalty
	// Apply an additional penalty to validators that did not vote on the correct target or slashed
	if !val.IsPrevEpochTargetAttester || val.IsSlashed {
		n, err := math.Mul64(effectiveBalance, val.InactivityScore)
		if err != nil {
			return &AttDelta{}, err
		}
		attDelta.InactivityPenalty = n / inactivityDenominator
	}

	return attDelta, nil
}
