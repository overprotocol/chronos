package helpers_test

import (
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	mathutil "github.com/prysmaticlabs/prysm/v5/math"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestTotalBalance_OK(t *testing.T) {
	helpers.ClearCache()

	state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: []*ethpb.Validator{
		{EffectiveBalance: 27 * 1e9}, {EffectiveBalance: 28 * 1e9},
		{EffectiveBalance: 32 * 1e9}, {EffectiveBalance: 40 * 1e9},
	}})
	require.NoError(t, err)

	balance := helpers.TotalBalance(state, []primitives.ValidatorIndex{0, 1, 2, 3})
	wanted := state.Validators()[0].EffectiveBalance + state.Validators()[1].EffectiveBalance +
		state.Validators()[2].EffectiveBalance + state.Validators()[3].EffectiveBalance
	assert.Equal(t, wanted, balance, "Incorrect TotalBalance")
}

func TestTotalBalance_ReturnsEffectiveBalanceIncrement(t *testing.T) {
	helpers.ClearCache()

	state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: []*ethpb.Validator{}})
	require.NoError(t, err)

	balance := helpers.TotalBalance(state, []primitives.ValidatorIndex{})
	wanted := params.BeaconConfig().EffectiveBalanceIncrement
	assert.Equal(t, wanted, balance, "Incorrect TotalBalance")
}

func TestGetBalance_OK(t *testing.T) {
	tests := []struct {
		i uint64
		b []uint64
	}{
		{i: 0, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 0, b: []uint64{0, 0, 0}},
		{i: 2, b: []uint64{0, 0, 0}},
	}
	for _, test := range tests {
		helpers.ClearCache()

		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Balances: test.b})
		require.NoError(t, err)
		assert.Equal(t, test.b[test.i], state.Balances()[test.i], "Incorrect Validator balance")
	}
}

func TestTotalActiveBalance(t *testing.T) {
	tests := []struct {
		vCount int
	}{
		{1},
		{10},
		{10000},
	}
	for _, test := range tests {
		helpers.ClearCache()

		validators := make([]*ethpb.Validator, 0)
		for i := 0; i < test.vCount; i++ {
			validators = append(validators, &ethpb.Validator{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: 1})
		}
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: validators})
		require.NoError(t, err)
		bal, err := helpers.TotalActiveBalance(state)
		require.NoError(t, err)
		require.Equal(t, uint64(test.vCount)*params.BeaconConfig().MaxEffectiveBalance, bal)
	}
}

func TestTotalBalanceWithQueue(t *testing.T) {
	tests := []struct {
		vCount int
		result int
	}{
		{vCount: 1, result: 1},
		{vCount: 10, result: 7},
		{vCount: 10000, result: 6667},
	}
	for _, test := range tests {
		validators := make([]*ethpb.Validator, 0)
		for i := 0; i < test.vCount; i++ {
			if i%3 == 1 {
				validators = append(validators, &ethpb.Validator{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: primitives.Epoch(i + 1)})
			} else if i%3 == 2 {
				validators = append(validators, &ethpb.Validator{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ActivationEpoch: primitives.Epoch(mathutil.MaxUint64), ExitEpoch: primitives.Epoch(mathutil.MaxUint64)})
			} else {
				validators = append(validators, &ethpb.Validator{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ActivationEpoch: 0, ExitEpoch: primitives.Epoch(mathutil.MaxUint64)})
			}
		}
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: validators})
		require.NoError(t, err)
		bal, err := helpers.TotalBalanceWithQueue(state)
		require.NoError(t, err)
		require.Equal(t, uint64(test.result)*params.BeaconConfig().MaxEffectiveBalance, bal)
	}
}

func TestTotalActiveBal_ReturnMin(t *testing.T) {
	tests := []struct {
		vCount int
	}{
		{1},
		{10},
		{10000},
	}
	for _, test := range tests {
		helpers.ClearCache()

		validators := make([]*ethpb.Validator, 0)
		for i := 0; i < test.vCount; i++ {
			validators = append(validators, &ethpb.Validator{EffectiveBalance: 1, ExitEpoch: 1})
		}
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: validators})
		require.NoError(t, err)
		bal, err := helpers.TotalActiveBalance(state)
		require.NoError(t, err)
		require.Equal(t, params.BeaconConfig().EffectiveBalanceIncrement, bal)
	}
}

func TestTotalActiveBalance_WithCache(t *testing.T) {
	tests := []struct {
		vCount    int
		wantCount int
	}{
		{vCount: 1, wantCount: 1},
		{vCount: 10, wantCount: 10},
		{vCount: 10000, wantCount: 10000},
	}
	for _, test := range tests {
		helpers.ClearCache()

		validators := make([]*ethpb.Validator, 0)
		for i := 0; i < test.vCount; i++ {
			validators = append(validators, &ethpb.Validator{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: 1})
		}
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: validators})
		require.NoError(t, err)
		bal, err := helpers.TotalActiveBalance(state)
		require.NoError(t, err)
		require.Equal(t, uint64(test.wantCount)*params.BeaconConfig().MaxEffectiveBalance, bal)
	}
}

func TestIncreaseBalance_OK(t *testing.T) {
	tests := []struct {
		i  primitives.ValidatorIndex
		b  []uint64
		nb uint64
		eb uint64
	}{
		{i: 0, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 1, eb: 27*1e9 + 1},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 0, eb: 28 * 1e9},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 33 * 1e9, eb: 65 * 1e9},
	}
	for _, test := range tests {
		helpers.ClearCache()

		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}},
			Balances: test.b,
		})
		require.NoError(t, err)
		require.NoError(t, helpers.IncreaseBalance(state, test.i, test.nb))
		assert.Equal(t, test.eb, state.Balances()[test.i], "Incorrect Validator balance")
	}
}

func TestIncreaseBalanceAndAdjustPrincipalBalance(t *testing.T) {
	// Define the test cases
	tests := []struct {
		name                     string
		initialEffectiveBalance  uint64
		initialPrincipalBalance  uint64
		initialBalance           uint64
		delta                    uint64
		expectedPrincipalBalance uint64
		expectedBalance          uint64
	}{
		{
			name:                     "principal balance < balance",
			initialEffectiveBalance:  10,
			initialPrincipalBalance:  8,
			initialBalance:           10,
			delta:                    6,
			expectedPrincipalBalance: 14,
			expectedBalance:          16,
		},
		{
			name:                     "balance < principal balance < balance + delta",
			initialEffectiveBalance:  8,
			initialPrincipalBalance:  10,
			initialBalance:           8,
			delta:                    6,
			expectedPrincipalBalance: 14,
			expectedBalance:          14,
		},
		{
			name:                     "balance + delta < principal balance",
			initialEffectiveBalance:  10,
			initialPrincipalBalance:  10,
			initialBalance:           3,
			delta:                    6,
			expectedPrincipalBalance: 10,
			expectedBalance:          9,
		},
	}

	// Iterate through test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup initial state for each test case
			protoState := &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{EffectiveBalance: tc.initialEffectiveBalance, PrincipalBalance: tc.initialPrincipalBalance},
				},
				Balances: []uint64{tc.initialBalance},
			}
			state, err := state_native.InitializeFromProtoPhase0(protoState)
			require.NoError(t, err)

			// Define the index
			idx := primitives.ValidatorIndex(0)

			// Call the function under test
			err = helpers.IncreaseBalanceAndAdjustPrincipalBalance(state, idx, tc.delta)
			require.NoError(t, err)

			// Fetch updated validator and balance
			validator, err := state.ValidatorAtIndex(idx)
			require.NoError(t, err)
			balance, err := state.BalanceAtIndex(idx)
			require.NoError(t, err)

			// Assert that the principal balance and balance are as expected
			assert.Equal(t, tc.expectedPrincipalBalance, validator.PrincipalBalance)
			assert.Equal(t, tc.expectedBalance, balance)
		})
	}
}

func TestDecreaseBalance_OK(t *testing.T) {
	tests := []struct {
		i  primitives.ValidatorIndex
		b  []uint64
		nb uint64
		eb uint64
	}{
		{i: 0, b: []uint64{2, 28 * 1e9, 32 * 1e9}, nb: 1, eb: 1},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 0, eb: 28 * 1e9},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 1}, nb: 2, eb: 0},
		{i: 3, b: []uint64{27 * 1e9, 28 * 1e9, 1, 28 * 1e9}, nb: 28 * 1e9, eb: 0},
	}
	for _, test := range tests {
		helpers.ClearCache()

		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 3}},
			Balances: test.b,
		})
		require.NoError(t, err)
		require.NoError(t, helpers.DecreaseBalance(state, test.i, test.nb))
		assert.Equal(t, test.eb, state.Balances()[test.i], "Incorrect Validator balance")
	}
}

func TestDecreaseBalanceAndAdjustPrincipalBalance(t *testing.T) {
	// Define the test cases
	tests := []struct {
		name                     string
		initialEffectiveBalance  uint64
		initialPrincipalBalance  uint64
		initialBalance           uint64
		delta                    uint64
		expectedPrincipalBalance uint64
		expectedBalance          uint64
	}{
		{
			name:                     "Test Case 1 - Decrease below Principal Balance",
			initialEffectiveBalance:  300_000_000_000,
			initialPrincipalBalance:  300_000_000_000,
			initialBalance:           400_000_000_000,
			delta:                    40_000_000_000,
			expectedPrincipalBalance: 270_000_000_000, // Still above min activation balance, so it shouldn't change
			expectedBalance:          360_000_000_000,
		},
		{
			name:                     "Test Case 2 - Decrease below Min Activation Balance",
			initialEffectiveBalance:  256_000_000_000,
			initialPrincipalBalance:  300_000_000_000,
			initialBalance:           255_000_000_000,
			delta:                    100_000_000_000,
			expectedPrincipalBalance: 256_000_000_000, // Principal balance set to min activation balance
			expectedBalance:          155_000_000_000,
		},
	}

	// Iterate through test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup initial state for each test case
			protoState := &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{EffectiveBalance: tc.initialEffectiveBalance, PrincipalBalance: tc.initialPrincipalBalance},
				},
				Balances: []uint64{tc.initialBalance},
			}
			state, err := state_native.InitializeFromProtoPhase0(protoState)
			require.NoError(t, err)

			// Define the index
			idx := primitives.ValidatorIndex(0)

			// Call the function under test
			err = helpers.DecreaseBalanceAndAdjustPrincipalBalance(state, idx, tc.delta)
			require.NoError(t, err)

			// Fetch updated validator and balance
			validator, err := state.ValidatorAtIndex(idx)
			require.NoError(t, err)
			balance, err := state.BalanceAtIndex(idx)
			require.NoError(t, err)

			// Assert that the principal balance and balance are as expected
			assert.Equal(t, tc.expectedPrincipalBalance, validator.PrincipalBalance)
			assert.Equal(t, tc.expectedBalance, balance)
		})
	}
}

func TestFinalityDelay(t *testing.T) {
	helpers.ClearCache()

	base := buildState(params.BeaconConfig().SlotsPerEpoch*10, 1)
	base.FinalizedCheckpoint = &ethpb.Checkpoint{Epoch: 3}
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)
	prevEpoch := primitives.Epoch(0)
	finalizedEpoch := primitives.Epoch(0)
	// Set values for each test case
	setVal := func() {
		prevEpoch = time.PrevEpoch(beaconState)
		finalizedEpoch = beaconState.FinalizedCheckpointEpoch()
	}
	setVal()
	d := helpers.FinalityDelay(prevEpoch, finalizedEpoch)
	w := time.PrevEpoch(beaconState) - beaconState.FinalizedCheckpointEpoch()
	assert.Equal(t, w, d, "Did not get wanted finality delay")

	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 4}))
	setVal()
	d = helpers.FinalityDelay(prevEpoch, finalizedEpoch)
	w = time.PrevEpoch(beaconState) - beaconState.FinalizedCheckpointEpoch()
	assert.Equal(t, w, d, "Did not get wanted finality delay")

	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 5}))
	setVal()
	d = helpers.FinalityDelay(prevEpoch, finalizedEpoch)
	w = time.PrevEpoch(beaconState) - beaconState.FinalizedCheckpointEpoch()
	assert.Equal(t, w, d, "Did not get wanted finality delay")
}

func TestIsInInactivityLeak(t *testing.T) {
	helpers.ClearCache()

	base := buildState(params.BeaconConfig().SlotsPerEpoch*10, 1)
	base.FinalizedCheckpoint = &ethpb.Checkpoint{Epoch: 3}
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)
	prevEpoch := primitives.Epoch(0)
	finalizedEpoch := primitives.Epoch(0)
	// Set values for each test case
	setVal := func() {
		prevEpoch = time.PrevEpoch(beaconState)
		finalizedEpoch = beaconState.FinalizedCheckpointEpoch()
	}
	setVal()
	assert.Equal(t, true, helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak true")
	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 4}))
	setVal()
	assert.Equal(t, true, helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak true")
	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 5}))
	setVal()
	assert.Equal(t, false, helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak false")
}

func buildState(slot primitives.Slot, validatorCount uint64) *ethpb.BeaconState {
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}
	return &ethpb.BeaconState{
		Slot:                        slot,
		Balances:                    validatorBalances,
		Validators:                  validators,
		RandaoMixes:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:                  make([][]byte, params.BeaconConfig().SlotsPerEpoch*10),
		FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, 32)},
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Root: make([]byte, 32)},
	}
}

func TestIncreaseBadBalance_NotOK(t *testing.T) {
	tests := []struct {
		i  primitives.ValidatorIndex
		b  []uint64
		nb uint64
	}{
		{i: 0, b: []uint64{math.MaxUint64, math.MaxUint64, math.MaxUint64}, nb: 1},
		{i: 2, b: []uint64{math.MaxUint64, math.MaxUint64, math.MaxUint64}, nb: 33 * 1e9},
	}
	for _, test := range tests {
		helpers.ClearCache()

		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}},
			Balances: test.b,
		})
		require.NoError(t, err)
		require.ErrorContains(t, "addition overflows", helpers.IncreaseBalance(state, test.i, test.nb))
	}
}

func TestEpochIssuance(t *testing.T) {
	tests := []struct {
		name string
		e    primitives.Epoch
		want uint64
	}{
		{name: "Issuance of Year 1", e: primitives.Epoch(0), want: 243531202435},
		{name: "Issuance of Year 2", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear + 1), want: 243531202435},
		{name: "Issuance of Year 3", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*2 + 1), want: 243531202435},
		{name: "Issuance of Year 4", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*3 + 1), want: 243531202435},
		{name: "Issuance of Year 5", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*4 + 1), want: 243531202435},
		{name: "Issuance of Year 6", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*5 + 1), want: 243531202435},
		{name: "Issuance of Year 7", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*6 + 1), want: 243531202435},
		{name: "Issuance of Year 8", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*7 + 1), want: 243531202435},
		{name: "Issuance of Year 9", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*8 + 1), want: 243531202435},
		{name: "Issuance of Year 10", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*9 + 1), want: 243531202435},
		{name: "Issuance of Year 11", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*10 + 1), want: 0},
		{name: "Issuance of Year 12", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*11 + 1), want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helpers.EpochIssuance(tt.e)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTargetDepositPlan(t *testing.T) {
	tests := []struct {
		name string
		e    primitives.Epoch
		want uint64
	}{
		{name: "TargetDepositPlan of Epoch 0 (year 1)", e: primitives.Epoch(0), want: 40000000000000000},
		{name: "TargetDepositPlan of Epoch 41063 (year 1)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear + 1) / 2), want: 60000243531176810},
		{name: "TargetDepositPlan of Epoch 82125 (Year 2)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear), want: 79999999999948750},
		{name: "TargetDepositPlan of Epoch 123188 (Year 2)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear*3 + 1) / 2), want: 100000243531125560},
		{name: "TargetDepositPlan of Epoch 164250 (Year 3)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 2), want: 119999999999897500},
		{name: "TargetDepositPlan of Epoch 205313 (Year 3)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear*5 + 1) / 2), want: 140000243531074310},
		{name: "TargetDepositPlan of Epoch 246375 (Year 4)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 3), want: 159999999999846250},
		{name: "TargetDepositPlan of Epoch 287438 (Year 4)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear*7 + 1) / 2), want: 180000243531023060},
		{name: "TargetDepositPlan of Epoch 328500 (Year 5)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 4), want: 200000000666636000},
		{name: "TargetDepositPlan of Epoch 369563 (Year 5)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear*9 + 1) / 2), want: 208333435471299848},
		{name: "TargetDepositPlan of Epoch 410625 (Year 6)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 5), want: 216666667333295000},
		{name: "TargetDepositPlan of Epoch 451688 (Year 6)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear*11 + 1) / 2), want: 225000102137958848},
		{name: "TargetDepositPlan of Epoch 492750 (Year 7)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 6), want: 233333333999954000},
		{name: "TargetDepositPlan of Epoch 533813 (Year 7)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear*13 + 1) / 2), want: 241666768804617848},
		{name: "TargetDepositPlan of Epoch 574875 (Year 8)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 7), want: 250000000666613000},
		{name: "TargetDepositPlan of Epoch 615938 (Year 8)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear*15 + 1) / 2), want: 258333435471276848},
		{name: "TargetDepositPlan of Epoch 657000 (Year 9)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 8), want: 266666667333272000},
		{name: "TargetDepositPlan of Epoch 698063 (Year 9)", e: primitives.Epoch((params.BeaconConfig().EpochsPerYear*17 + 1) / 2), want: 275000102137935848},
		{name: "TargetDepositPlan of Epoch 739125 (Year 10)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 9), want: 283333333999931000},
		{name: "TargetDepositPlan of Epoch 780188 (Year 10)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*19 + 1), want: 300000000 * 1e9},
		{name: "TargetDepositPlan of Epoch 821250 (Year 11)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear * 10), want: 300000000 * 1e9},
		{name: "TargetDepositPlan of Epoch 862313 (Year 11)", e: primitives.Epoch(params.BeaconConfig().EpochsPerYear*21 + 1), want: 300000000 * 1e9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helpers.TargetDepositPlan(tt.e)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTotalRewardWithReserveUsage_OK(t *testing.T) {
	tests := []struct {
		epoch   uint64
		factor  uint64
		reserve uint64
		want    uint64
		sign    int
		usage   uint64
	}{
		{epoch: 1, factor: 1030000000000, reserve: 0, want: 243531202435, usage: 0},
		{epoch: 1, factor: 0, reserve: 1000000000, want: 243531202435, usage: 0},
		{epoch: 1, factor: 100000000000, reserve: 1000000000000000, want: 47437686693262, usage: 47194155490827},
		{epoch: 1, factor: 1030000000000, reserve: 1000000000, want: 244531202435, usage: 1000000000},
		{epoch: 862313, factor: 1030000000000, reserve: 0, want: 0, usage: 0},
		{epoch: 862313, factor: 0, reserve: 1000000000, want: 0, usage: 0},
		{epoch: 862313, factor: 100000000000, reserve: 1000000000000000, want: 47194155490827, usage: 47194155490827},
		{epoch: 862313, factor: 100000000000, reserve: 1000000, want: 1000000, usage: 1000000},
	}
	for _, test := range tests {
		base := buildState(params.BeaconConfig().SlotsPerEpoch.Mul(test.epoch), 20000)
		base.RewardAdjustmentFactor = test.factor
		base.Reserves = test.reserve
		beaconState, err := state_native.InitializeFromProtoPhase0(base)
		require.NoError(t, err)

		got, usage := helpers.TotalRewardWithReserveUsage(beaconState)
		assert.Equal(t, test.want, got)
		assert.Equal(t, test.usage, usage)
	}
}

func TestProcessRewardAdjustmentFactor_OK(t *testing.T) {
	tests := []struct {
		name         string
		epoch        uint64
		valCnt       uint64
		rewardFactor uint64
		currReserve  uint64
		wantFactor   uint64
	}{
		{name: "Case 1 : first year, smaller than target, base factor", epoch: 1, valCnt: 10000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardAdjustmentFactorPrecision / 1000, wantFactor: 100150},
		{name: "Case 2 : first year, larger than target, base factor", epoch: 1, valCnt: 160000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardAdjustmentFactorPrecision / 1000, wantFactor: 99850},
		{name: "Case 3 : later year, smaller than target, base factor", epoch: 410625, valCnt: 500000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardAdjustmentFactorPrecision / 100, wantFactor: 1000000},
		{name: "Case 4 : later year, larger than target, base factor", epoch: 410625, valCnt: 1000000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardAdjustmentFactorPrecision / 100, wantFactor: 999850},
		{name: "Case 5 : last years, smaller valset, base factor", epoch: 862313, valCnt: 1000000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardAdjustmentFactorPrecision / 100, wantFactor: 1000000},
		{name: "Case 6 : last years, larger valset, base factor", epoch: 862313, valCnt: 1500000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardAdjustmentFactorPrecision / 100, wantFactor: 999850},
	}
	for _, test := range tests {
		base := buildState(params.BeaconConfig().SlotsPerEpoch.Mul(test.epoch), test.valCnt)
		base.RewardAdjustmentFactor = test.rewardFactor
		base.Reserves = test.currReserve
		beaconState, err := state_native.InitializeFromProtoPhase0(base)
		require.NoError(t, err)

		beaconState, err = helpers.ProcessRewardAdjustmentFactor(beaconState)
		require.NoError(t, err)
		assert.Equal(t, test.wantFactor, beaconState.RewardAdjustmentFactor(), test.name)
		assert.Equal(t, test.currReserve, beaconState.Reserves(), test.name)
	}
}

func TestDecreaseRewardAdjustmentFactor_OK(t *testing.T) {
	tests := []struct {
		name         string
		rewardFactor uint64
		want         uint64
	}{
		{name: "Rewardfactor smaller than delta", rewardFactor: 100, want: 0},
		{name: "Rewardfactor same as delta", rewardFactor: 150, want: 0},
		{name: "Rewardfactor larger than delta", rewardFactor: 200, want: 50},
	}
	for _, test := range tests {
		base := buildState(params.BeaconConfig().SlotsPerEpoch.Mul(1), 10000)
		base.RewardAdjustmentFactor = test.rewardFactor
		beaconState, err := state_native.InitializeFromProtoPhase0(base)
		require.NoError(t, err)
		err = helpers.DecreaseRewardAdjustmentFactor(beaconState)
		require.NoError(t, err)
		got := beaconState.RewardAdjustmentFactor()
		assert.Equal(t, test.want, got, test.name)
	}
}

func TestIncreaseRewardAdjustmentFactor_OK(t *testing.T) {
	tests := []struct {
		name         string
		rewardFactor uint64
		epoch        uint64
		want         uint64
	}{
		{name: "New rewardfactor less than MaxRewardAdjustmentFactors (year 1)", epoch: (params.BeaconConfig().EpochsPerYear + 1) / 2, rewardFactor: 100000, want: 100150},
		{name: "New rewardfactor same as MaxRewardAdjustmentFactors (year 1)", epoch: (params.BeaconConfig().EpochsPerYear + 1) / 2, rewardFactor: 999850, want: 1000000},
		{name: "New rewardfactor larger than MaxRewardAdjustmentFactors (year 1)", epoch: (params.BeaconConfig().EpochsPerYear + 1) / 2, rewardFactor: 1000001, want: 1000000},
		{name: "New rewardfactor less than MaxRewardAdjustmentFactors (year 2)", epoch: (params.BeaconConfig().EpochsPerYear*2 + 1) / 2, rewardFactor: 100000, want: 100150},
		{name: "New rewardfactor same as MaxRewardAdjustmentFactors (year 2)", epoch: (params.BeaconConfig().EpochsPerYear*2 + 1) / 2, rewardFactor: 999850, want: 1000000},
		{name: "New rewardfactor larger than MaxRewardAdjustmentFactors (year 2)", epoch: (params.BeaconConfig().EpochsPerYear*2 + 1) / 2, rewardFactor: 1000001, want: 1000000},
		{name: "New rewardfactor less than MaxRewardAdjustmentFactors (year 11)", epoch: (params.BeaconConfig().EpochsPerYear*20 + 1) / 2, rewardFactor: 100000, want: 100150},
		{name: "New rewardfactor same as MaxRewardAdjustmentFactors (year 11)", epoch: (params.BeaconConfig().EpochsPerYear*20 + 1) / 2, rewardFactor: 999850, want: 1000000},
		{name: "New rewardfactor larger than MaxRewardAdjustmentFactors (year 11)", epoch: (params.BeaconConfig().EpochsPerYear*20 + 1) / 2, rewardFactor: 1000001, want: 1000000},
	}
	for _, test := range tests {
		base := buildState(params.BeaconConfig().SlotsPerEpoch.Mul(test.epoch), 10000)
		base.RewardAdjustmentFactor = test.rewardFactor
		beaconState, err := state_native.InitializeFromProtoPhase0(base)
		require.NoError(t, err)
		err = helpers.IncreaseRewardAdjustmentFactor(beaconState)
		require.NoError(t, err)
		got := beaconState.RewardAdjustmentFactor()
		assert.Equal(t, test.want, got, test.name)
	}
}

func TestDecreaseReserves_OK(t *testing.T) {
	tests := []struct {
		r    uint64
		sub  uint64
		want uint64
	}{
		{r: 1000000000000000000, sub: 100000000000000000, want: 900000000000000000},
		{r: 100000000000000000, sub: 100000000000000001, want: 0},
	}
	for _, test := range tests {
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Reserves: test.r,
		})
		require.NoError(t, err)
		require.NoError(t, helpers.DecreaseReserves(state, test.sub))
		assert.Equal(t, test.want, state.Reserves())
	}
}
