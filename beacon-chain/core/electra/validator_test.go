package electra_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestQueueEntireBalanceAndResetValidator(t *testing.T) {
	s, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{
		Validators: []*eth.Validator{
			{
				EffectiveBalance:           params.BeaconConfig().MinActivationBalance + 100_000,
				ActivationEligibilityEpoch: primitives.Epoch(100),
			},
		},
		Balances: []uint64{
			params.BeaconConfig().MinActivationBalance + 100_000,
		},
	})
	require.NoError(t, err)
	require.NoError(t, electra.QueueEntireBalanceAndResetValidator(s, 0))
	b, err := s.BalanceAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), b, "balance was not changed")
	v, err := s.ValidatorAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), v.EffectiveBalance, "effective balance was not reset")
	require.Equal(t, params.BeaconConfig().FarFutureEpoch, v.ActivationEligibilityEpoch, "activation eligibility epoch was not reset")
	pbd, err := s.PendingDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pbd), "pending balance deposits should have one element")
	require.Equal(t, params.BeaconConfig().MinActivationBalance+100_000, pbd[0].Amount, "pending balance deposit amount is incorrect")
}

func TestQueueExcessActiveBalance_Ok(t *testing.T) {
	st, _ := util.DeterministicGenesisStateElectra(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	bals := st.Balances()
	bals[0] = params.BeaconConfig().MinActivationBalance + 1000
	require.NoError(t, st.SetBalances(bals))

	err := electra.QueueExcessActiveBalance(st, 0)
	require.NoError(t, err)

	pbd, err := st.PendingDeposits()
	require.NoError(t, err)
	require.Equal(t, uint64(1000), pbd[0].Amount) // appends it at the end

	bals = st.Balances()
	require.Equal(t, params.BeaconConfig().MinActivationBalance, bals[0])
}

func TestQueueEntireBalanceAndResetValidator_Ok(t *testing.T) {
	st, _ := util.DeterministicGenesisStateElectra(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	// need to manually set this to 0 as after 6110 these balances are now 0 and instead populates pending balance deposits
	bals := st.Balances()
	bals[0] = params.BeaconConfig().MinActivationBalance - 1000
	require.NoError(t, st.SetBalances(bals))
	err := electra.QueueEntireBalanceAndResetValidator(st, 0)
	require.NoError(t, err)

	pbd, err := st.PendingDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pbd))
	require.Equal(t, params.BeaconConfig().MinActivationBalance-1000, pbd[0].Amount)
	bal, err := st.BalanceAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), bal)
}
