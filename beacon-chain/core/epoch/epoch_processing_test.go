package epoch_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestProcessFinalUpdates_CanProcess(t *testing.T) {
	s := buildState(t, params.BeaconConfig().SlotsPerHistoricalRoot-1, uint64(params.BeaconConfig().SlotsPerEpoch))
	ce := time.CurrentEpoch(s)
	ne := ce + 1
	require.NoError(t, s.SetEth1DataVotes([]*ethpb.Eth1Data{}))
	balances := s.Balances()
	balances[0] = 255.75 * 1e9
	balances[1] = 250 * 1e9
	require.NoError(t, s.SetBalances(balances))

	mixes := s.RandaoMixes()
	mixes[ce] = []byte{'A'}
	require.NoError(t, s.SetRandaoMixes(mixes))
	newS, err := epoch.ProcessFinalUpdates(s)
	require.NoError(t, err)

	// Verify effective balance is correctly updated.
	assert.Equal(t, params.BeaconConfig().MaxEffectiveBalance, newS.Validators()[0].EffectiveBalance, "Effective balance incorrectly updated")
	assert.Equal(t, uint64(248*1e9), newS.Validators()[1].EffectiveBalance, "Effective balance incorrectly updated")

	// Verify randao is correctly updated in the right position.
	mix, err := newS.RandaoMixAtIndex(uint64(ne))
	assert.NoError(t, err)
	assert.DeepNotEqual(t, params.BeaconConfig().ZeroHash[:], mix, "latest RANDAO still zero hashes")

	currAtt, err := newS.CurrentEpochAttestations()
	require.NoError(t, err)
	assert.NotNil(t, currAtt, "Nil value stored in current epoch attestations instead of empty slice")
}

func TestProcessRegistryUpdates_NoRotation(t *testing.T) {
	base := &ethpb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{ExitEpoch: params.BeaconConfig().MaxSeedLookahead},
			{ExitEpoch: params.BeaconConfig().MaxSeedLookahead},
		},
		Balances: []uint64{
			params.BeaconConfig().MaxEffectiveBalance,
			params.BeaconConfig().MaxEffectiveBalance,
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
	}
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)
	newState, err := epoch.ProcessRegistryUpdates(context.Background(), beaconState)
	require.NoError(t, err)
	for i, validator := range newState.Validators() {
		assert.Equal(t, params.BeaconConfig().MaxSeedLookahead, validator.ExitEpoch, "Could not update registry %d", i)
	}
}

func TestProcessRegistryUpdates_EligibleToActivate(t *testing.T) {
	finalizedEpoch := primitives.Epoch(4)
	base := &ethpb.BeaconState{
		Slot:                5 * params.BeaconConfig().SlotsPerEpoch,
		FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: finalizedEpoch, Root: make([]byte, fieldparams.RootLength)},
	}
	limit := helpers.ValidatorActivationChurnLimit(0, params.BeaconConfig().EffectiveBalanceIncrement, 5)
	for i := uint64(0); i < limit+10; i++ {
		base.Validators = append(base.Validators, &ethpb.Validator{
			ActivationEligibilityEpoch: finalizedEpoch,
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
		})
	}
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)
	currentEpoch := time.CurrentEpoch(beaconState)
	newState, err := epoch.ProcessRegistryUpdates(context.Background(), beaconState)
	require.NoError(t, err)
	for i, validator := range newState.Validators() {
		if uint64(i) < limit && validator.ActivationEpoch != helpers.ActivationExitEpoch(currentEpoch) {
			t.Errorf("Could not update registry %d, validators failed to activate: wanted activation epoch %d, got %d",
				i, helpers.ActivationExitEpoch(currentEpoch), validator.ActivationEpoch)
		}
		if uint64(i) >= limit && validator.ActivationEpoch != params.BeaconConfig().FarFutureEpoch {
			t.Errorf("Could not update registry %d, validators should not have been activated, wanted activation epoch: %d, got %d",
				i, params.BeaconConfig().FarFutureEpoch, validator.ActivationEpoch)
		}
	}
}

func TestProcessRegistryUpdates_ActivationCompletes(t *testing.T) {
	base := &ethpb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{ExitEpoch: params.BeaconConfig().MaxSeedLookahead,
				ActivationEpoch: 5 + params.BeaconConfig().MaxSeedLookahead + 1},
			{ExitEpoch: params.BeaconConfig().MaxSeedLookahead,
				ActivationEpoch: 5 + params.BeaconConfig().MaxSeedLookahead + 1},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
	}
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)
	newState, err := epoch.ProcessRegistryUpdates(context.Background(), beaconState)
	require.NoError(t, err)
	for i, validator := range newState.Validators() {
		assert.Equal(t, params.BeaconConfig().MaxSeedLookahead, validator.ExitEpoch, "Could not update registry %d, unexpected exit slot", i)
	}
}

func TestProcessRegistryUpdates_ValidatorsBailedOut(t *testing.T) {
	principalBalance := params.BeaconConfig().MinActivationBalance
	bailoutBuffer := principalBalance * params.BeaconConfig().InactivityPenaltyRate / params.BeaconConfig().InactivityPenaltyRatePrecision
	actualBalance := principalBalance - bailoutBuffer - 1
	base := &ethpb.BeaconStateDeneb{
		Slot: 0,
		Validators: []*ethpb.Validator{
			{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: principalBalance,
				PrincipalBalance: principalBalance,
			},
			{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: principalBalance,
				PrincipalBalance: principalBalance,
			},
		},
		Balances:            []uint64{actualBalance, actualBalance},
		FinalizedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
	}
	beaconState, err := state_native.InitializeFromProtoDeneb(base)
	require.NoError(t, err)
	newState, err := epoch.ProcessRegistryUpdates(context.Background(), beaconState)
	require.NoError(t, err)
	for i, validator := range newState.Validators() {
		assert.Equal(t, params.BeaconConfig().MaxSeedLookahead+1, validator.ExitEpoch, "Could not update registry %d, unexpected exit slot", i)
	}
}

func TestProcessRegistryUpdates_CanExits(t *testing.T) {
	e := primitives.Epoch(5)
	exitEpoch := helpers.ActivationExitEpoch(e)
	minWithdrawalDelay := params.BeaconConfig().MinValidatorWithdrawabilityDelay
	base := &ethpb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch.Mul(uint64(e)),
		Validators: []*ethpb.Validator{
			{
				ExitEpoch:         exitEpoch,
				WithdrawableEpoch: exitEpoch + minWithdrawalDelay},
			{
				ExitEpoch:         exitEpoch,
				WithdrawableEpoch: exitEpoch + minWithdrawalDelay},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
	}
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)
	newState, err := epoch.ProcessRegistryUpdates(context.Background(), beaconState)
	require.NoError(t, err)
	for i, validator := range newState.Validators() {
		assert.Equal(t, exitEpoch, validator.ExitEpoch, "Could not update registry %d, unexpected exit slot", i)
	}
}

func buildState(t testing.TB, slot primitives.Slot, validatorCount uint64) state.BeaconState {
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
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	if err := s.SetSlot(slot); err != nil {
		t.Error(err)
	}
	if err := s.SetBalances(validatorBalances); err != nil {
		t.Error(err)
	}
	if err := s.SetValidators(validators); err != nil {
		t.Error(err)
	}
	return s
}

func TestProcessHistoricalDataUpdate(t *testing.T) {
	tests := []struct {
		name     string
		st       func() state.BeaconState
		verifier func(state.BeaconState)
	}{
		{
			name: "no change",
			st: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 1)
				return st
			},
			verifier: func(st state.BeaconState) {
				roots, err := st.HistoricalSummaries()
				require.NoError(t, err)
				require.Equal(t, 0, len(roots))
			},
		},
		{
			name: "after capella can process and get historical summary",
			st: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateCapella(t, 1)
				st, err := transition.ProcessSlots(context.Background(), st, params.BeaconConfig().SlotsPerHistoricalRoot-1)
				require.NoError(t, err)
				return st
			},
			verifier: func(st state.BeaconState) {
				summaries, err := st.HistoricalSummaries()
				require.NoError(t, err)
				require.Equal(t, 1, len(summaries))

				br, err := stateutil.ArraysRoot(st.BlockRoots(), fieldparams.BlockRootsLength)
				require.NoError(t, err)
				sr, err := stateutil.ArraysRoot(st.StateRoots(), fieldparams.StateRootsLength)
				require.NoError(t, err)
				b := &ethpb.HistoricalSummary{
					BlockSummaryRoot: br[:],
					StateSummaryRoot: sr[:],
				}
				require.DeepEqual(t, b, summaries[0])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := epoch.ProcessHistoricalDataUpdate(tt.st())
			require.NoError(t, err)
			tt.verifier(got)
		})
	}
}
