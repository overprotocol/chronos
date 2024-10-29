package altair

import (
	"context"
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestInitializeEpochValidators_Ok(t *testing.T) {
	ffe := params.BeaconConfig().FarFutureEpoch
	s, err := state_native.InitializeFromProtoAltair(&ethpb.BeaconStateAltair{
		Slot: params.BeaconConfig().SlotsPerEpoch,
		// Validator 0 is slashed
		// Validator 1 is withdrawable
		// Validator 2 is active prev epoch and current epoch
		// Validator 3 is active prev epoch
		Validators: []*ethpb.Validator{
			{Slashed: true, WithdrawableEpoch: ffe, EffectiveBalance: 100},
			{EffectiveBalance: 100},
			{WithdrawableEpoch: ffe, ExitEpoch: ffe, EffectiveBalance: 100},
			{WithdrawableEpoch: ffe, ExitEpoch: 1, EffectiveBalance: 100},
		},
		InactivityScores: []uint64{0, 1, 2, 3},
		Balances:         []uint64{100, 100, 100, 100},
	})
	require.NoError(t, err)
	v, b, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	assert.DeepEqual(t, &precompute.Validator{
		IsSlashed:                    true,
		CurrentEpochEffectiveBalance: 100,
		InactivityScore:              0,
		ActualBalance:                100,
	}, v[0], "Incorrect validator 0 status")
	assert.DeepEqual(t, &precompute.Validator{
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: 100,
		InactivityScore:              1,
		ActualBalance:                100,
	}, v[1], "Incorrect validator 1 status")
	assert.DeepEqual(t, &precompute.Validator{
		IsActivePrevEpoch:            true,
		IsActiveCurrentEpoch:         true,
		CurrentEpochEffectiveBalance: 100,
		InactivityScore:              2,
		ActualBalance:                100,
	}, v[2], "Incorrect validator 2 status")
	assert.DeepEqual(t, &precompute.Validator{
		IsActivePrevEpoch:            true,
		CurrentEpochEffectiveBalance: 100,
		InactivityScore:              3,
		ActualBalance:                100,
	}, v[3], "Incorrect validator 3 status")

	wantedBalances := &precompute.Balance{
		ActiveCurrentEpoch: 100,
		ActivePrevEpoch:    200,
	}
	assert.DeepEqual(t, wantedBalances, b, "Incorrect wanted balance")
}

func TestInitializeEpochValidators_Overflow(t *testing.T) {
	ffe := params.BeaconConfig().FarFutureEpoch
	s, err := state_native.InitializeFromProtoAltair(&ethpb.BeaconStateAltair{
		Slot: params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{WithdrawableEpoch: ffe, ExitEpoch: ffe, EffectiveBalance: math.MaxUint64},
			{WithdrawableEpoch: ffe, ExitEpoch: ffe, EffectiveBalance: math.MaxUint64},
		},
		InactivityScores: []uint64{0, 1},
		Balances:         []uint64{math.MaxUint64, math.MaxUint64},
	})
	require.NoError(t, err)
	_, _, err = InitializePrecomputeValidators(context.Background(), s)
	require.ErrorContains(t, "could not read every validator: addition overflows", err)
}

func TestInitializeEpochValidators_BadState(t *testing.T) {
	s, err := state_native.InitializeFromProtoAltair(&ethpb.BeaconStateAltair{
		Validators:       []*ethpb.Validator{{}},
		InactivityScores: []uint64{},
	})
	require.NoError(t, err)
	_, _, err = InitializePrecomputeValidators(context.Background(), s)
	require.ErrorContains(t, "num of validators is different than num of inactivity scores", err)
}

func TestProcessEpochParticipation(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	validators, balance, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	require.DeepEqual(t, &precompute.Validator{
		IsActiveCurrentEpoch:         true,
		IsActivePrevEpoch:            true,
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		PrincipalBalance:             params.BeaconConfig().MaxEffectiveBalance,
	}, validators[0])
	require.DeepEqual(t, &precompute.Validator{
		IsActiveCurrentEpoch:         true,
		IsActivePrevEpoch:            true,
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		IsCurrentEpochAttester:       true,
		IsPrevEpochAttester:          true,
		IsPrevEpochSourceAttester:    true,
		PrincipalBalance:             params.BeaconConfig().MaxEffectiveBalance,
	}, validators[1])
	require.DeepEqual(t, &precompute.Validator{
		IsActiveCurrentEpoch:         true,
		IsActivePrevEpoch:            true,
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		IsCurrentEpochAttester:       true,
		IsPrevEpochAttester:          true,
		IsPrevEpochSourceAttester:    true,
		IsCurrentEpochTargetAttester: true,
		IsPrevEpochTargetAttester:    true,
		PrincipalBalance:             params.BeaconConfig().MaxEffectiveBalance,
	}, validators[2])
	require.DeepEqual(t, &precompute.Validator{
		IsActiveCurrentEpoch:         true,
		IsActivePrevEpoch:            true,
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		IsCurrentEpochAttester:       true,
		IsPrevEpochAttester:          true,
		IsPrevEpochSourceAttester:    true,
		IsCurrentEpochTargetAttester: true,
		IsPrevEpochTargetAttester:    true,
		IsPrevEpochHeadAttester:      true,
		PrincipalBalance:             params.BeaconConfig().MaxEffectiveBalance,
	}, validators[3])
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance*3, balance.PrevEpochAttested)
	require.Equal(t, balance.CurrentEpochTargetAttested, params.BeaconConfig().MaxEffectiveBalance*2)
	require.Equal(t, balance.PrevEpochTargetAttested, params.BeaconConfig().MaxEffectiveBalance*2)
	require.Equal(t, balance.PrevEpochHeadAttested, params.BeaconConfig().MaxEffectiveBalance*1)
}

func TestProcessEpochParticipation_InactiveValidator(t *testing.T) {
	generateParticipation := func(flags ...uint8) byte {
		b := byte(0)
		var err error
		for _, flag := range flags {
			b, err = AddValidatorFlag(b, flag)
			require.NoError(t, err)
		}
		return b
	}
	eb := params.BeaconConfig().MaxEffectiveBalance
	st, err := state_native.InitializeFromProtoAltair(&ethpb.BeaconStateAltair{
		Slot: 2 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{EffectiveBalance: eb},               // Inactive
			{EffectiveBalance: eb, ExitEpoch: 2}, // Inactive current epoch, active previous epoch
			{EffectiveBalance: eb, ExitEpoch: params.BeaconConfig().FarFutureEpoch}, // Active
		},
		CurrentEpochParticipation: []byte{
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex),
		},
		PreviousEpochParticipation: []byte{
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex),
		},
		InactivityScores: []uint64{0, 0, 0},
		Balances:         []uint64{eb, eb, eb},
	})
	require.NoError(t, err)
	validators, balance, err := InitializePrecomputeValidators(context.Background(), st)
	require.NoError(t, err)
	validators, balance, err = ProcessEpochParticipation(context.Background(), st, balance, validators)
	require.NoError(t, err)
	require.DeepEqual(t, &precompute.Validator{
		IsActiveCurrentEpoch:         false,
		IsActivePrevEpoch:            false,
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: eb,
		ActualBalance:                eb,
	}, validators[0])
	require.DeepEqual(t, &precompute.Validator{
		IsActiveCurrentEpoch:         false,
		IsActivePrevEpoch:            true,
		IsPrevEpochAttester:          true,
		IsPrevEpochSourceAttester:    true,
		IsPrevEpochTargetAttester:    true,
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: eb,
		ActualBalance:                eb,
	}, validators[1])
	require.DeepEqual(t, &precompute.Validator{
		IsActiveCurrentEpoch:         true,
		IsActivePrevEpoch:            true,
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: eb,
		IsCurrentEpochAttester:       true,
		IsPrevEpochAttester:          true,
		IsPrevEpochSourceAttester:    true,
		IsCurrentEpochTargetAttester: true,
		IsPrevEpochTargetAttester:    true,
		IsPrevEpochHeadAttester:      true,
		ActualBalance:                eb,
	}, validators[2])
	require.Equal(t, balance.PrevEpochAttested, 2*eb)
	require.Equal(t, balance.CurrentEpochTargetAttested, eb)
	require.Equal(t, balance.PrevEpochTargetAttested, 2*eb)
	require.Equal(t, balance.PrevEpochHeadAttested, eb)
}

func TestAttestationsDelta(t *testing.T) {
	tests := []struct {
		name    string
		prepare func() state.BeaconState
		check   func(*testing.T, []*AttDelta)
	}{
		{
			name: "altair - zero penalty",
			prepare: func() state.BeaconState {
				s, err := testState()
				require.NoError(t, err)
				return s
			},
			check: func(t *testing.T, deltas []*AttDelta) {
				rewards := make([]uint64, len(deltas))
				penalties := make([]uint64, len(deltas))
				for i, d := range deltas {
					rewards[i] = d.HeadReward + d.SourceReward + d.TargetReward
					penalties[i] = d.SourcePenalty + d.TargetPenalty
				}

				// Reward amount should increase as validator index increases due to setup.
				for i := 1; i < len(rewards); i++ {
					require.Equal(t, true, rewards[i] > rewards[i-1])
				}

				// Penalty amount should decrease as validator index increases due to setup.
				for i := 1; i < len(penalties); i++ {
					require.Equal(t, true, penalties[i] <= penalties[i-1])
				}

				// First index should have 0 reward.
				require.Equal(t, uint64(0), rewards[0])
				// Last index should have 0 penalty.
				require.Equal(t, uint64(0), penalties[len(penalties)-1])

				want := []uint64{0, 8561643835, 19977168949, 24733637746}
				require.DeepEqual(t, want, rewards)
				want = []uint64{0, 0, 0, 0}
				require.DeepEqual(t, want, penalties)
			},
		},
		{
			name: "altair - penalty",
			prepare: func() state.BeaconState {
				s, err := testState()
				require.NoError(t, err)
				s, err = withAccumulatedInactivityScores(s)
				require.NoError(t, err)
				return s
			},
			check: func(t *testing.T, deltas []*AttDelta) {
				penalties := make([]uint64, len(deltas))
				for i, d := range deltas {
					penalties[i] = d.SourcePenalty + d.TargetPenalty
				}

				// Penalty amount should decrease as validator index increases due to setup.
				for i := 1; i < len(penalties); i++ {
					require.Equal(t, true, penalties[i] <= penalties[i-1])
				}

				// Last index should have 0 penalty.
				require.Equal(t, uint64(0), penalties[len(penalties)-1])

				want := []uint64{1625395, 1083597, 0, 0}
				require.DeepEqual(t, want, penalties)
			},
		},
		{
			name: "altair - zero penalty in inactivity leak",
			prepare: func() state.BeaconState {
				s, err := testState()
				require.NoError(t, err)
				s, err = withLeak(s)
				require.NoError(t, err)
				s, err = withAccumulatedInactivityScores(s)
				require.NoError(t, err)
				balance := params.BeaconConfig().MaxEffectiveBalance - uint64(1_000_000_000) // 255 OVER
				require.NoError(t, s.SetBalances([]uint64{balance, balance, balance, balance}))
				return s
			},
			check: func(t *testing.T, deltas []*AttDelta) {
				penalties := make([]uint64, len(deltas))
				for i, d := range deltas {
					penalties[i] = d.SourcePenalty + d.TargetPenalty
				}

				// Penalty amount should decrease as validator index increases due to setup.
				for i := 1; i < len(penalties); i++ {
					require.Equal(t, true, penalties[i] <= penalties[i-1])
				}

				// Last index should have 0 penalty.
				require.Equal(t, uint64(0), penalties[len(penalties)-1])

				want := []uint64{0, 0, 0, 0}
				require.DeepEqual(t, want, penalties)
			},
		},
		{
			name: "altair - penalty in inactivity leak",
			prepare: func() state.BeaconState {
				s, err := testState()
				require.NoError(t, err)
				s, err = withLeak(s)
				require.NoError(t, err)
				s, err = withAccumulatedInactivityScores(s)
				require.NoError(t, err)
				balance := params.BeaconConfig().MaxEffectiveBalance - uint64(100_000_000) // 255.9 OVER
				require.NoError(t, s.SetBalances([]uint64{balance, balance, balance, balance}))
				return s
			},
			check: func(t *testing.T, deltas []*AttDelta) {
				penalties := make([]uint64, len(deltas))
				for i, d := range deltas {
					penalties[i] = d.SourcePenalty + d.TargetPenalty
				}

				// Penalty amount should decrease as validator index increases due to setup.
				for i := 1; i < len(penalties); i++ {
					require.Equal(t, true, penalties[i] <= penalties[i-1])
				}

				// Last index should have 0 penalty.
				require.Equal(t, uint64(0), penalties[len(penalties)-1])

				want := []uint64{1625395, 1083597, 0, 0}
				require.DeepEqual(t, want, penalties)
			},
		},
		{
			name: "bellatrix - zero penalty",
			prepare: func() state.BeaconState {
				s, err := testStateBellatrix()
				require.NoError(t, err)
				return s
			},
			check: func(t *testing.T, deltas []*AttDelta) {
				rewards := make([]uint64, len(deltas))
				penalties := make([]uint64, len(deltas))
				for i, d := range deltas {
					rewards[i] = d.HeadReward + d.SourceReward + d.TargetReward
					penalties[i] = d.SourcePenalty + d.TargetPenalty
				}

				// Reward amount should increase as validator index increases due to setup.
				for i := 1; i < len(rewards); i++ {
					require.Equal(t, true, rewards[i] > rewards[i-1])
				}

				// Penalty amount should decrease as validator index increases due to setup.
				for i := 1; i < len(penalties); i++ {
					require.Equal(t, true, penalties[i] <= penalties[i-1])
				}

				// First index should have 0 reward.
				require.Equal(t, uint64(0), rewards[0])
				// Last index should have 0 penalty.
				require.Equal(t, uint64(0), penalties[len(penalties)-1])

				want := []uint64{0, 8561643835, 19977168949, 24733637746}
				require.DeepEqual(t, want, rewards)
				want = []uint64{0, 0, 0, 0}
				require.DeepEqual(t, want, penalties)
			},
		},
		{
			name: "bellatrix - penalty",
			prepare: func() state.BeaconState {
				s, err := testStateBellatrix()
				require.NoError(t, err)
				s, err = withAccumulatedInactivityScores(s)
				require.NoError(t, err)
				return s
			},
			check: func(t *testing.T, deltas []*AttDelta) {
				penalties := make([]uint64, len(deltas))
				for i, d := range deltas {
					penalties[i] = d.SourcePenalty + d.TargetPenalty
				}

				// Penalty amount should decrease as validator index increases due to setup.
				for i := 1; i < len(penalties); i++ {
					require.Equal(t, true, penalties[i] <= penalties[i-1])
				}

				// Last index should have 0 penalty.
				require.Equal(t, uint64(0), penalties[len(penalties)-1])

				want := []uint64{1625395, 1083597, 0, 0}
				require.DeepEqual(t, want, penalties)
			},
		},
		{
			name: "bellatrix - zero penalty in inactivity leak",
			prepare: func() state.BeaconState {
				s, err := testStateBellatrix()
				require.NoError(t, err)
				s, err = withLeak(s)
				require.NoError(t, err)
				s, err = withAccumulatedInactivityScores(s)
				require.NoError(t, err)
				balance := params.BeaconConfig().MaxEffectiveBalance - uint64(1_000_000_000) // 255 OVER
				require.NoError(t, s.SetBalances([]uint64{balance, balance, balance, balance}))
				return s
			},
			check: func(t *testing.T, deltas []*AttDelta) {
				penalties := make([]uint64, len(deltas))
				for i, d := range deltas {
					penalties[i] = d.SourcePenalty + d.TargetPenalty
				}

				// Penalty amount should decrease as validator index increases due to setup.
				for i := 1; i < len(penalties); i++ {
					require.Equal(t, true, penalties[i] <= penalties[i-1])
				}

				// Last index should have 0 penalty.
				require.Equal(t, uint64(0), penalties[len(penalties)-1])

				want := []uint64{0, 0, 0, 0}
				require.DeepEqual(t, want, penalties)
			},
		},
		{
			name: "bellatrix - penalty in inactivity leak",
			prepare: func() state.BeaconState {
				s, err := testStateBellatrix()
				require.NoError(t, err)
				s, err = withLeak(s)
				require.NoError(t, err)
				s, err = withAccumulatedInactivityScores(s)
				require.NoError(t, err)
				balance := params.BeaconConfig().MaxEffectiveBalance - uint64(100_000_000) // 255.9 OVER
				require.NoError(t, s.SetBalances([]uint64{balance, balance, balance, balance}))
				return s
			},
			check: func(t *testing.T, deltas []*AttDelta) {
				penalties := make([]uint64, len(deltas))
				for i, d := range deltas {
					penalties[i] = d.SourcePenalty + d.TargetPenalty
				}

				// Penalty amount should decrease as validator index increases due to setup.
				for i := 1; i < len(penalties); i++ {
					require.Equal(t, true, penalties[i] <= penalties[i-1])
				}

				// Last index should have 0 penalty.
				require.Equal(t, uint64(0), penalties[len(penalties)-1])

				want := []uint64{1625395, 1083597, 0, 0}
				require.DeepEqual(t, want, penalties)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconState := tt.prepare()

			validators, balance, err := InitializePrecomputeValidators(context.Background(), beaconState)
			require.NoError(t, err)
			validators, balance, err = ProcessEpochParticipation(context.Background(), beaconState, balance, validators)
			require.NoError(t, err)

			deltas, _, err := AttestationsDelta(beaconState, balance, validators)
			require.NoError(t, err)
			tt.check(t, deltas)
		})
	}
}

func TestProcessRewardsAndPenaltiesPrecompute_Ok(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	validators, balance, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	s, err = ProcessRewardsAndPenaltiesPrecompute(s, balance, validators)
	require.NoError(t, err)

	balances := s.Balances()
	// Reward amount should increase as validator index increases due to setup.
	for i := 1; i < len(balances); i++ {
		require.Equal(t, true, balances[i] >= balances[i-1])
	}

	wanted := make([]uint64, s.NumValidators())
	deltas, rDeltas, err := AttestationsDelta(s, balance, validators)
	require.NoError(t, err)

	rewards := make([]uint64, len(deltas))
	penalties := make([]uint64, len(deltas))
	totalReserve := uint64(0)
	for i, d := range deltas {
		rewards[i] = d.HeadReward + d.SourceReward + d.TargetReward
		penalties[i] = d.SourcePenalty + d.TargetPenalty
		totalReserve += rDeltas[i]
	}
	for i := range rewards {
		wanted[i] += rewards[i]
	}
	for i := range penalties {
		if wanted[i] > penalties[i] {
			wanted[i] -= penalties[i]
		} else {
			wanted[i] = 0
		}
	}
	require.DeepEqual(t, wanted, balances)
	require.Equal(t, uint64(0), totalReserve)
}

func TestProcessRewardsAndPenaltiesPrecompute_InactivityLeak(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	validators, balance, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	sCopy := s.Copy()
	s, err = ProcessRewardsAndPenaltiesPrecompute(s, balance, validators)
	require.NoError(t, err)

	// Copied state where finality happened long ago
	require.NoError(t, sCopy.SetSlot(params.BeaconConfig().SlotsPerEpoch*1000))
	sCopy, err = ProcessRewardsAndPenaltiesPrecompute(sCopy, balance, validators)
	require.NoError(t, err)

	balances := s.Balances()
	inactivityBalances := sCopy.Balances()
	// Balances decreased to 0 due to inactivity
	require.Equal(t, uint64(19977168949), balances[2])
	require.Equal(t, uint64(24733637746), balances[3])
	require.Equal(t, uint64(0), inactivityBalances[2])
	require.Equal(t, uint64(0), inactivityBalances[3])
}

func TestProcessInactivityScores_CanProcessInactivityLeak(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	defaultScore := uint64(5)
	require.NoError(t, s.SetInactivityScores([]uint64{defaultScore, defaultScore, defaultScore, defaultScore}))
	require.NoError(t, s.SetSlot(params.BeaconConfig().SlotsPerEpoch*primitives.Slot(params.BeaconConfig().MinEpochsToInactivityPenalty+2)))
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	validators, _, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	s, _, err = ProcessInactivityScores(context.Background(), s, validators)
	require.NoError(t, err)
	inactivityScores, err := s.InactivityScores()
	require.NoError(t, err)
	// V0 and V1 didn't vote head. V2 and V3 did.
	require.Equal(t, defaultScore+params.BeaconConfig().InactivityScoreBias, inactivityScores[0])
	require.Equal(t, defaultScore+params.BeaconConfig().InactivityScoreBias, inactivityScores[1])
	require.Equal(t, defaultScore-1, inactivityScores[2])
	require.Equal(t, defaultScore-1, inactivityScores[3])
}

func TestProcessInactivityScores_GenesisEpoch(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	defaultScore := uint64(10)
	require.NoError(t, s.SetInactivityScores([]uint64{defaultScore, defaultScore, defaultScore, defaultScore}))
	require.NoError(t, s.SetSlot(params.BeaconConfig().GenesisSlot))
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	validators, _, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	s, _, err = ProcessInactivityScores(context.Background(), s, validators)
	require.NoError(t, err)
	inactivityScores, err := s.InactivityScores()
	require.NoError(t, err)
	require.Equal(t, defaultScore, inactivityScores[0])
	require.Equal(t, defaultScore, inactivityScores[1])
	require.Equal(t, defaultScore, inactivityScores[2])
	require.Equal(t, defaultScore, inactivityScores[3])
}

func TestProcessInactivityScores_CanProcessNonInactivityLeak(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	defaultScore := uint64(5)
	require.NoError(t, s.SetInactivityScores([]uint64{defaultScore, defaultScore, defaultScore, defaultScore}))
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	validators, _, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	s, _, err = ProcessInactivityScores(context.Background(), s, validators)
	require.NoError(t, err)
	inactivityScores, err := s.InactivityScores()
	require.NoError(t, err)

	require.DeepEqual(t, []uint64{7, 7, 4, 4}, inactivityScores, "Incorrect inactivity scores")
}

func TestProcessRewardsAndPenaltiesPrecompute_GenesisEpoch(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	validators, balance, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(0))
	s, err = ProcessRewardsAndPenaltiesPrecompute(s, balance, validators)
	require.NoError(t, err)

	balances := s.Balances()
	// Nothing should happen at genesis epoch
	require.Equal(t, uint64(0), balances[0])
	for i := 1; i < len(balances); i++ {
		require.Equal(t, true, balances[i] == balances[i-1])
	}
}

func TestProcessRewardsAndPenaltiesPrecompute_BadState(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)
	_, balance, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	_, err = ProcessRewardsAndPenaltiesPrecompute(s, balance, []*precompute.Validator{})
	require.ErrorContains(t, "validator registries not the same length as state's validator registries", err)
}

func TestProcessInactivityScores_NonEligibleValidator(t *testing.T) {
	s, err := testState()
	require.NoError(t, err)
	defaultScore := uint64(5)
	require.NoError(t, s.SetInactivityScores([]uint64{defaultScore, defaultScore, defaultScore, defaultScore}))
	validators, balance, err := InitializePrecomputeValidators(context.Background(), s)
	require.NoError(t, err)

	// v0 is eligible (not active previous epoch, slashed and not withdrawable)
	validators[0].IsActivePrevEpoch = false
	validators[0].IsSlashed = true
	validators[0].IsWithdrawableCurrentEpoch = false

	// v1 is not eligible (not active previous epoch, not slashed and not withdrawable)
	validators[1].IsActivePrevEpoch = false
	validators[1].IsSlashed = false
	validators[1].IsWithdrawableCurrentEpoch = false

	// v2 is not eligible (not active previous epoch, slashed and withdrawable)
	validators[2].IsActivePrevEpoch = false
	validators[2].IsSlashed = true
	validators[2].IsWithdrawableCurrentEpoch = true

	// v3 is eligible (active previous epoch)
	validators[3].IsActivePrevEpoch = true

	validators, _, err = ProcessEpochParticipation(context.Background(), s, balance, validators)
	require.NoError(t, err)
	s, _, err = ProcessInactivityScores(context.Background(), s, validators)
	require.NoError(t, err)
	inactivityScores, err := s.InactivityScores()
	require.NoError(t, err)

	require.Equal(t, uint64(7), inactivityScores[0])
	require.Equal(t, defaultScore, inactivityScores[1]) // Should remain unchanged
	require.Equal(t, defaultScore, inactivityScores[2]) // Should remain unchanged
	require.Equal(t, uint64(4), inactivityScores[3])
}

func testState() (state.BeaconState, error) {
	generateParticipation := func(flags ...uint8) byte {
		b := byte(0)
		var err error
		for _, flag := range flags {
			b, err = AddValidatorFlag(b, flag)
			if err != nil {
				return 0
			}
		}
		return b
	}
	return state_native.InitializeFromProtoAltair(&ethpb.BeaconStateAltair{
		Slot: 2 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: params.BeaconConfig().FarFutureEpoch, PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance},
			{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: params.BeaconConfig().FarFutureEpoch, PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance},
			{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: params.BeaconConfig().FarFutureEpoch, PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance},
			{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: params.BeaconConfig().FarFutureEpoch, PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance},
		},
		CurrentEpochParticipation: []byte{
			0,
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex),
		},
		PreviousEpochParticipation: []byte{
			0,
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex),
		},
		InactivityScores: []uint64{0, 0, 0, 0},
		Balances:         []uint64{0, 0, 0, 0},
	})
}

func testStateBellatrix() (state.BeaconState, error) {
	generateParticipation := func(flags ...uint8) byte {
		b := byte(0)
		var err error
		for _, flag := range flags {
			b, err = AddValidatorFlag(b, flag)
			if err != nil {
				return 0
			}
		}
		return b
	}
	return state_native.InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{
		Slot: 2 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: params.BeaconConfig().FarFutureEpoch, PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance},
			{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: params.BeaconConfig().FarFutureEpoch, PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance},
			{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: params.BeaconConfig().FarFutureEpoch, PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance},
			{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: params.BeaconConfig().FarFutureEpoch, PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance},
		},
		CurrentEpochParticipation: []byte{
			0,
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex),
		},
		PreviousEpochParticipation: []byte{
			0,
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex),
			generateParticipation(params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex),
		},
		InactivityScores: []uint64{0, 0, 0, 0},
		Balances:         []uint64{0, 0, 0, 0},
	})
}

// withLeak is a wrapper function to make a state with inactivity leak.
func withLeak(s state.BeaconState) (state.BeaconState, error) {
	finalizedEpoch := primitives.Epoch(2)
	err := s.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: finalizedEpoch,
		Root:  make([]byte, fieldparams.RootLength),
	})
	if err != nil {
		return nil, err
	}
	currentEpoch := finalizedEpoch + params.BeaconConfig().MinEpochsToInactivityPenalty + 2
	slot := primitives.Slot(uint64(currentEpoch) * uint64(params.BeaconConfig().SlotsPerEpoch))
	err = s.SetSlot(slot)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// withAccumulatedInactivityScores is a wrapper function to make a state with accumulated inactivity scores,
// so that the penalty can be applied.
func withAccumulatedInactivityScores(s state.BeaconState) (state.BeaconState, error) {
	inactivityScore := params.BeaconConfig().InactivityScorePenaltyThreshold + 1
	err := s.SetInactivityScores([]uint64{inactivityScore, inactivityScore, inactivityScore, inactivityScore})
	if err != nil {
		return nil, err
	}
	return s, nil
}
