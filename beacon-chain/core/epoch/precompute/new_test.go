package precompute_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch/precompute"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestNew(t *testing.T) {
	ffe := params.BeaconConfig().FarFutureEpoch
	params.BeaconConfig().MinSlashingWithdrawableDelay = 2
	params.BeaconConfig().MinValidatorWithdrawabilityDelay = 1
	s, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch,
		// Validator 0 is slashed
		// Validator 1 is withdrawable
		// Validator 2 is active prev epoch and current epoch
		// Validator 3 is active prev epoch
		Validators: []*ethpb.Validator{
			{Slashed: true, ExitEpoch: 0, EffectiveBalance: 100},
			{ExitEpoch: 0, EffectiveBalance: 100},
			{ExitEpoch: ffe, EffectiveBalance: 100},
			{ExitEpoch: 1, EffectiveBalance: 100},
		},
	})
	require.NoError(t, err)
	e := params.BeaconConfig().FarFutureSlot
	v, b, err := precompute.New(context.Background(), s)
	require.NoError(t, err)
	assert.DeepEqual(t, &precompute.Validator{
		IsSlashed:                    true,
		CurrentEpochEffectiveBalance: 100,
		InclusionDistance:            e,
		InclusionSlot:                e,
	}, v[0], "Incorrect validator 0 status")
	assert.DeepEqual(t, &precompute.Validator{
		IsWithdrawableCurrentEpoch:   true,
		CurrentEpochEffectiveBalance: 100,
		InclusionDistance:            e,
		InclusionSlot:                e,
	}, v[1], "Incorrect validator 1 status")
	assert.DeepEqual(t, &precompute.Validator{
		IsActiveCurrentEpoch:         true,
		IsActivePrevEpoch:            true,
		CurrentEpochEffectiveBalance: 100,
		InclusionDistance:            e,
		InclusionSlot:                e,
	}, v[2], "Incorrect validator 2 status")
	assert.DeepEqual(t, &precompute.Validator{
		IsActivePrevEpoch:            true,
		CurrentEpochEffectiveBalance: 100,
		InclusionDistance:            e,
		InclusionSlot:                e,
	}, v[3], "Incorrect validator 3 status")

	wantedBalances := &precompute.Balance{
		ActiveCurrentEpoch: 100,
		ActivePrevEpoch:    200,
	}
	assert.DeepEqual(t, wantedBalances, b, "Incorrect wanted balance")
}
