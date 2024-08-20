package helpers

import (
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	mathutil "github.com/prysmaticlabs/prysm/v4/math"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestTotalBalance_OK(t *testing.T) {
	state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: []*ethpb.Validator{
		{EffectiveBalance: 27 * 1e9},
		{EffectiveBalance: 28 * 1e9},
		{EffectiveBalance: 32 * 1e9},
		{EffectiveBalance: 40 * 1e9},
	}})
	require.NoError(t, err)

	balance := TotalBalance(state, []primitives.ValidatorIndex{0, 1, 2, 3})
	wanted := state.Validators()[0].EffectiveBalance + state.Validators()[1].EffectiveBalance +
		state.Validators()[2].EffectiveBalance + state.Validators()[3].EffectiveBalance
	assert.Equal(t, wanted, balance, "Incorrect TotalBalance")
}

func TestTotalBalance_ReturnsEffectiveBalanceIncrement(t *testing.T) {
	state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: []*ethpb.Validator{}})
	require.NoError(t, err)

	balance := TotalBalance(state, []primitives.ValidatorIndex{})
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
		validators := make([]*ethpb.Validator, 0)
		for i := 0; i < test.vCount; i++ {
			validators = append(validators, &ethpb.Validator{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: 1})
		}
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: validators})
		require.NoError(t, err)
		bal, err := TotalActiveBalance(state)
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
		bal, err := TotalBalanceWithQueue(state)
		require.NoError(t, err)
		require.Equal(t, uint64(test.result)*params.BeaconConfig().MaxEffectiveBalance, bal)
	}
}

func TestTotalActiveBal_ReturnMin(t *testing.T) {
	ClearCache()
	defer ClearCache()
	tests := []struct {
		vCount int
	}{
		{1},
		{10},
		{10000},
	}
	for _, test := range tests {
		validators := make([]*ethpb.Validator, 0)
		for i := 0; i < test.vCount; i++ {
			validators = append(validators, &ethpb.Validator{EffectiveBalance: 1, ExitEpoch: 1})
		}
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: validators})
		require.NoError(t, err)
		bal, err := TotalActiveBalance(state)
		require.NoError(t, err)
		require.Equal(t, params.BeaconConfig().EffectiveBalanceIncrement, bal)
	}
}

func TestTotalActiveBalance_WithCache(t *testing.T) {
	ClearCache()
	defer ClearCache()
	tests := []struct {
		vCount    int
		wantCount int
	}{
		{vCount: 1, wantCount: 1},
		{vCount: 10, wantCount: 10},
		{vCount: 10000, wantCount: 10000},
	}
	for _, test := range tests {
		validators := make([]*ethpb.Validator, 0)
		for i := 0; i < test.vCount; i++ {
			validators = append(validators, &ethpb.Validator{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance, ExitEpoch: 1})
		}
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: validators})
		require.NoError(t, err)
		bal, err := TotalActiveBalance(state)
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
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4},
			},
			Balances: test.b,
		})
		require.NoError(t, err)
		require.NoError(t, IncreaseBalance(state, test.i, test.nb))
		assert.Equal(t, test.eb, state.Balances()[test.i], "Incorrect Validator balance")
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
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 3},
			},
			Balances: test.b,
		})
		require.NoError(t, err)
		require.NoError(t, DecreaseBalance(state, test.i, test.nb))
		assert.Equal(t, test.eb, state.Balances()[test.i], "Incorrect Validator balance")
	}
}

func TestFinalityDelay(t *testing.T) {
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
	d := FinalityDelay(prevEpoch, finalizedEpoch)
	w := time.PrevEpoch(beaconState) - beaconState.FinalizedCheckpointEpoch()
	assert.Equal(t, w, d, "Did not get wanted finality delay")

	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 4}))
	setVal()
	d = FinalityDelay(prevEpoch, finalizedEpoch)
	w = time.PrevEpoch(beaconState) - beaconState.FinalizedCheckpointEpoch()
	assert.Equal(t, w, d, "Did not get wanted finality delay")

	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 5}))
	setVal()
	d = FinalityDelay(prevEpoch, finalizedEpoch)
	w = time.PrevEpoch(beaconState) - beaconState.FinalizedCheckpointEpoch()
	assert.Equal(t, w, d, "Did not get wanted finality delay")
}

func TestIsInInactivityLeak(t *testing.T) {
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
	assert.Equal(t, true, IsInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak true")
	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 4}))
	setVal()
	assert.Equal(t, true, IsInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak true")
	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 5}))
	setVal()
	assert.Equal(t, false, IsInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak false")
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
		Slashings:                   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
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
		state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4},
			},
			Balances: test.b,
		})
		require.NoError(t, err)
		require.ErrorContains(t, "addition overflows", IncreaseBalance(state, test.i, test.nb))
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
			got := EpochIssuance(tt.e)
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
			got := TargetDepositPlan(tt.e)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestProcessRewardfactorUpdate_OK(t *testing.T) {
	tests := []struct {
		name         string
		epoch        uint64
		valCnt       uint64
		rewardFactor uint64
		currReserve  uint64
		wantFactor   uint64
	}{
		{name: "Case 1 : first year, smaller valset, base factor", epoch: 1, valCnt: 10000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 0},
		{name: "Case 2 : first year, slight small valset, base factor", epoch: 1, valCnt: 150000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 0},
		{name: "Case 3 : first year, slight large valset, base factor", epoch: 1, valCnt: 160000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 0},
		{name: "Case 4 : first year, larger valset, base factor", epoch: 1, valCnt: 200000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 0},
		{name: "Case 5 : later year, smaller valset, base factor", epoch: 410625, valCnt: 500000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 1001500000},
		{name: "Case 6 : later year, slight small valset, base factor", epoch: 410625, valCnt: 840000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 1000112629},
		{name: "Case 7 : later year, slight large valset, base factor", epoch: 410625, valCnt: 850000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 999935399},
		{name: "Case 8 : later year, larger valset, base factor", epoch: 410625, valCnt: 1000000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 998500000},
		{name: "Case 9 : last years, smaller valset, base factor", epoch: 862313, valCnt: 1000000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 1001500000},
		{name: "Case 10 : last years, slight small valset, base factor", epoch: 862313, valCnt: 1100000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 1000919999},
		{name: "Case 11 : last years, slight large valset, base factor", epoch: 862313, valCnt: 1230000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 999256000},
		{name: "Case 12 : last years, larger valset, base factor", epoch: 862313, valCnt: 1500000, currReserve: 1000000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, wantFactor: 998500000},
	}
	for _, test := range tests {
		base := buildState(params.BeaconConfig().SlotsPerEpoch.Mul(test.epoch), test.valCnt)
		base.RewardAdjustmentFactor = test.rewardFactor
		base.CurrentEpochReserve = test.currReserve
		beaconState, err := state_native.InitializeFromProtoPhase0(base)
		require.NoError(t, err)

		err = ProcessRewardfactorUpdate(beaconState)
		require.NoError(t, err)
		assert.Equal(t, test.wantFactor, beaconState.RewardAdjustmentFactor(), test.name)
		assert.Equal(t, test.currReserve, beaconState.PreviousEpochReserve(), test.name)
	}
}

func TestTotalRewardWithReserveUsage_OK(t *testing.T) {
	tests := []struct {
		epoch   uint64
		factor  uint64
		reserve uint64
		want    uint64
		usage   uint64
	}{
		{epoch: 1, factor: 1030000000000, reserve: 0, want: 243531202435, usage: 0},
		{epoch: 1, factor: 0, reserve: 1000000000, want: 243531202435, usage: 0},
		{epoch: 1, factor: 100000000000, reserve: 1000000000000000, want: 1461187214611, usage: 1217656012176},
		{epoch: 1, factor: 1030000000000, reserve: 1000000000, want: 244531202435, usage: 1000000000},
		{epoch: 862313, factor: 1030000000000, reserve: 0, want: 0, usage: 0},
		{epoch: 862313, factor: 0, reserve: 1000000000, want: 0, usage: 0},
		{epoch: 862313, factor: 100000000000, reserve: 1000000000000000, want: 1217656012176, usage: 1217656012176},
		{epoch: 862313, factor: 100000000000, reserve: 1000000, want: 1000000, usage: 1000000},
	}
	for _, test := range tests {
		base := buildState(params.BeaconConfig().SlotsPerEpoch.Mul(test.epoch), 20000)
		base.RewardAdjustmentFactor = test.factor
		base.PreviousEpochReserve = test.reserve
		base.CurrentEpochReserve = test.reserve
		beaconState, err := state_native.InitializeFromProtoPhase0(base)
		require.NoError(t, err)

		got, usage := TotalRewardWithReserveUsage(beaconState)
		assert.Equal(t, test.want, got)
		assert.Equal(t, test.usage, usage)
	}
}

func TestCalculateRewardAdjustmentFactor_OK(t *testing.T) {
	tests := []struct {
		name         string
		epoch        uint64
		valCnt       uint64
		rewardFactor uint64
		want         uint64
	}{
		{name: "Case 1 : early year, smaller valset, 0 factor", epoch: 1, valCnt: 10000, want: 0},
		{name: "Case 2 : early year, slight small valset, 0 factor", epoch: 1, valCnt: 150000, want: 0},
		{name: "Case 3 : early year, slight large valset, 0 factor", epoch: 1, valCnt: 160000, want: 0},
		{name: "Case 4 : early year, larger valset, 0 factor", epoch: 1, valCnt: 200000, want: 0},
		{name: "Case 1-1 : early year, smaller valset, base factor", epoch: 1, valCnt: 10000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 0},
		{name: "Case 2-1 : early year, slight small valset, base factor", epoch: 1, valCnt: 150000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 0},
		{name: "Case 3-1 : early year, slight large valset, base factor", epoch: 1, valCnt: 160000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 0},
		{name: "Case 4-1 : early year, larger valset, base factor", epoch: 1, valCnt: 200000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 0},
		{name: "Case 5 : later year, smaller valset, 0 factor", epoch: 410625, valCnt: 500000, want: 1500000},
		{name: "Case 6 : later year, slight small valset, 0 factor", epoch: 410625, valCnt: 840000, want: 112629},
		{name: "Case 7 : later year, slight large valset, 0 factor", epoch: 410625, valCnt: 850000, want: 0},
		{name: "Case 8 : later year, larger valset, 0 factor", epoch: 410625, valCnt: 1000000, want: 0},
		{name: "Case 5-1 : later year, smaller valset, base factor", epoch: 410625, valCnt: 500000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 1001500000},
		{name: "Case 6-1 : later year, slight small valset, base factor", epoch: 410625, valCnt: 840000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 1000112629},
		{name: "Case 7-1 : later year, slight large valset, base factor", epoch: 410625, valCnt: 850000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 999935399},
		{name: "Case 8-1 : later year, larger valset, base factor", epoch: 410625, valCnt: 1000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 998500000},
		{name: "Case 9 : last years, smaller valset, 0 factor", epoch: 862313, valCnt: 1000000, want: 1500000},
		{name: "Case 10 : last years, slight small valset, 0 factor", epoch: 862313, valCnt: 1100000, want: 919999},
		{name: "Case 11 : last years, slight large valset, 0 factor", epoch: 862313, valCnt: 1230000, want: 0},
		{name: "Case 12 : last years, larger valset, 0 factor", epoch: 862313, valCnt: 1500000, want: 0},
		{name: "Case 9-1 : last years, smaller valset, base factor", epoch: 862313, valCnt: 1000000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 1001500000},
		{name: "Case 10-1 : last years, slight small valset, base factor", epoch: 862313, valCnt: 1100000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 1000919999},
		{name: "Case 11-1 : last years, slight large valset, base factor", epoch: 862313, valCnt: 1230000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 999256000},
		{name: "Case 12-1 : last years, larger valset, base factor", epoch: 862313, valCnt: 1500000, rewardFactor: params.BeaconConfig().RewardFeedbackPrecision / 1000, want: 998500000},
	}
	for _, test := range tests {
		base := buildState(params.BeaconConfig().SlotsPerEpoch.Mul(test.epoch), test.valCnt)
		base.RewardAdjustmentFactor = test.rewardFactor
		beaconState, err := state_native.InitializeFromProtoPhase0(base)
		require.NoError(t, err)

		got, err := CalculateRewardAdjustmentFactor(beaconState)
		require.NoError(t, err)
		assert.Equal(t, test.want, got, test.name)
	}
}

func TestTruncateRewardAdjustmentFactor_OK(t *testing.T) {
	tests := []struct {
		name  string
		in    uint64
		epoch uint64
		want  uint64
	}{
		{name: "TruncateRewardAdjustmentFactor of Epoch 41063 (year 1)", epoch: (params.BeaconConfig().EpochsPerYear + 1) / 2, in: 0, want: 0},
		{name: "TruncateRewardAdjustmentFactor of Epoch 41063 (year 1)", epoch: (params.BeaconConfig().EpochsPerYear + 1) / 2, in: 5000000000, want: 0},
		{name: "TruncateRewardAdjustmentFactor of Epoch 41063 (year 1)", epoch: (params.BeaconConfig().EpochsPerYear + 1) / 2, in: 10000000000, want: 0},
		{name: "TruncateRewardAdjustmentFactor of Epoch 41063 (year 1)", epoch: (params.BeaconConfig().EpochsPerYear + 1) / 2, in: 10000000001, want: 0},
		{name: "TruncateRewardAdjustmentFactor of Epoch 82125 (Year 2)", epoch: params.BeaconConfig().EpochsPerYear, in: 0, want: 0},
		{name: "TruncateRewardAdjustmentFactor of Epoch 82125 (Year 2)", epoch: params.BeaconConfig().EpochsPerYear, in: 5000000000, want: 5000000000},
		{name: "TruncateRewardAdjustmentFactor of Epoch 82125 (Year 2)", epoch: params.BeaconConfig().EpochsPerYear, in: 10000000000, want: 10000000000},
		{name: "TruncateRewardAdjustmentFactor of Epoch 82125 (Year 2)", epoch: params.BeaconConfig().EpochsPerYear, in: 10000000001, want: 10000000000},
		{name: "TruncateRewardAdjustmentFactor of Epoch 739125 (Year 10)", epoch: params.BeaconConfig().EpochsPerYear * 9, in: 0, want: 0},
		{name: "TruncateRewardAdjustmentFactor of Epoch 739125 (Year 10)", epoch: params.BeaconConfig().EpochsPerYear * 9, in: 5000000000, want: 5000000000},
		{name: "TruncateRewardAdjustmentFactor of Epoch 739125 (Year 10)", epoch: params.BeaconConfig().EpochsPerYear * 9, in: 10000000000, want: 10000000000},
		{name: "TruncateRewardAdjustmentFactor of Epoch 739125 (Year 10)", epoch: params.BeaconConfig().EpochsPerYear * 9, in: 10000000001, want: 10000000000},
		{name: "TruncateRewardAdjustmentFactor of Epoch 862313 (Year 11)", epoch: params.BeaconConfig().EpochsPerYear*21 + 1, in: 0, want: 0},
		{name: "TruncateRewardAdjustmentFactor of Epoch 862313 (Year 11)", epoch: params.BeaconConfig().EpochsPerYear*21 + 1, in: 5000000000, want: 5000000000},
		{name: "TruncateRewardAdjustmentFactor of Epoch 862313 (Year 11)", epoch: params.BeaconConfig().EpochsPerYear*21 + 1, in: 10000000000, want: 10000000000},
		{name: "TruncateRewardAdjustmentFactor of Epoch 862313 (Year 11)", epoch: params.BeaconConfig().EpochsPerYear*21 + 1, in: 10000000001, want: 10000000000},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, TruncateRewardAdjustmentFactor(test.in, primitives.Epoch(test.epoch)))
	}
}

func TestDecreaseCurrentReserve_OK(t *testing.T) {
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
			CurrentEpochReserve: test.r,
		})
		require.NoError(t, err)
		require.NoError(t, DecreaseCurrentReserve(state, test.sub))
		assert.Equal(t, test.want, state.CurrentEpochReserve())
	}
}
