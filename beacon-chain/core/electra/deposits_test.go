package electra_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	stateTesting "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/testing"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestProcessPendingDeposits(t *testing.T) {
	tests := []struct {
		name    string
		state   state.BeaconState
		wantErr bool
		check   func(*testing.T, state.BeaconState)
	}{
		{
			name:    "nil state fails",
			state:   nil,
			wantErr: true,
		},
		{
			name: "no deposits resets balance to consume",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 8_000)
				require.NoError(t, st.SetDepositBalanceToConsume(800))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
			},
		},
		{
			name: "more deposits than balance to consume processes partial deposits",
			state: func() state.BeaconState {
				amountAvailForProcessing := helpers.ActivationBalanceChurnLimit(8_000 * 1e9)
				depositAmount := uint64(amountAvailForProcessing) / 10
				st := stateWithPendingDeposits(t, 8_000, 20, depositAmount)
				require.NoError(t, st.SetDepositBalanceToConsume(800))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				amountAvailForProcessing := helpers.ActivationBalanceChurnLimit(8_000 * 1e9)
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(800), res)
				// Validators 0..9 should have their balance increased
				for i := primitives.ValidatorIndex(0); i < 10; i++ {
					b, err := st.BalanceAtIndex(i)
					require.NoError(t, err)
					require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(amountAvailForProcessing)/10, b)
				}

				// Half of the balance deposits should have been processed.
				remaining, err := st.PendingDeposits()
				require.NoError(t, err)
				require.Equal(t, 10, len(remaining))
			},
		},
		{
			name: "withdrawn validators should not consume churn",
			state: func() state.BeaconState {
				amountAvailForProcessing := helpers.ActivationBalanceChurnLimit(8_000 * 1e9)
				depositAmount := uint64(amountAvailForProcessing)
				// set the pending deposits to the maximum churn limit
				st := stateWithPendingDeposits(t, 8_000, 2, depositAmount)
				vals := st.Validators()
				vals[1].ExitEpoch = 0
				require.NoError(t, st.SetValidators(vals))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				amountAvailForProcessing := helpers.ActivationBalanceChurnLimit(8_000 * 1e9)
				// Validators 0..9 should have their balance increased
				for i := primitives.ValidatorIndex(0); i < 2; i++ {
					b, err := st.BalanceAtIndex(i)
					require.NoError(t, err)
					require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(amountAvailForProcessing), b)
				}

				// All pending deposits should have been processed
				remaining, err := st.PendingDeposits()
				require.NoError(t, err)
				require.Equal(t, 0, len(remaining))
			},
		},
		{
			name: "less deposits than balance to consume processes all deposits",
			state: func() state.BeaconState {
				amountAvailForProcessing := helpers.ActivationBalanceChurnLimit(8_000 * 1e9)
				depositAmount := uint64(amountAvailForProcessing) / 5
				st := stateWithPendingDeposits(t, 8_000, 5, depositAmount)
				require.NoError(t, st.SetDepositBalanceToConsume(0))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				amountAvailForProcessing := helpers.ActivationBalanceChurnLimit(8_000 * 1e9)
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
				// Validators 0..4 should have their balance increased
				for i := primitives.ValidatorIndex(0); i < 4; i++ {
					b, err := st.BalanceAtIndex(i)
					require.NoError(t, err)
					require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(amountAvailForProcessing)/5, b)
				}

				// All of the balance deposits should have been processed.
				remaining, err := st.PendingDeposits()
				require.NoError(t, err)
				require.Equal(t, 0, len(remaining))
			},
		},
		{
			name: "process pending deposit for unknown key, activates new key",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 0)
				sk, err := bls.RandKey()
				require.NoError(t, err)
				wc := make([]byte, 32)
				wc[31] = byte(0)
				dep := stateTesting.GeneratePendingDeposit(t, sk, params.BeaconConfig().MinActivationBalance, bytesutil.ToBytes32(wc), 0)
				require.NoError(t, st.SetPendingDeposits([]*eth.PendingDeposit{dep}))
				require.Equal(t, 0, len(st.Validators()))
				require.Equal(t, 0, len(st.Balances()))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
				b, err := st.BalanceAtIndex(0)
				require.NoError(t, err)
				require.Equal(t, params.BeaconConfig().MinActivationBalance, b)

				// All of the balance deposits should have been processed.
				remaining, err := st.PendingDeposits()
				require.NoError(t, err)
				require.Equal(t, 0, len(remaining))

				// validator becomes active
				require.Equal(t, 1, len(st.Validators()))
				require.Equal(t, 1, len(st.Balances()))
			},
		},
		{
			name: "process excess balance that uses a point to infinity signature, processed as a topup",
			state: func() state.BeaconState {
				excessBalance := uint64(800)
				st := stateWithActiveBalanceETH(t, 256)
				validators := st.Validators()
				sk, err := bls.RandKey()
				require.NoError(t, err)
				wc := make([]byte, 32)
				wc[31] = byte(0)
				validators[0].PublicKey = sk.PublicKey().Marshal()
				validators[0].WithdrawalCredentials = wc
				dep := stateTesting.GeneratePendingDeposit(t, sk, excessBalance, bytesutil.ToBytes32(wc), 0)
				dep.Signature = common.InfiniteSignature[:]
				require.NoError(t, st.SetValidators(validators))
				require.NoError(t, st.SetPendingDeposits([]*eth.PendingDeposit{dep}))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
				b, err := st.BalanceAtIndex(0)
				require.NoError(t, err)
				require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(800), b)

				// All of the balance deposits should have been processed.
				remaining, err := st.PendingDeposits()
				require.NoError(t, err)
				require.Equal(t, 0, len(remaining))
			},
		},
		{
			name: "exiting validator deposit postponed",
			state: func() state.BeaconState {
				amountAvailForProcessing := helpers.ActivationBalanceChurnLimit(8_000 * 1e9)
				depositAmount := uint64(amountAvailForProcessing) / 5
				st := stateWithPendingDeposits(t, 8_000, 5, depositAmount)
				require.NoError(t, st.SetDepositBalanceToConsume(0))
				v, err := st.ValidatorAtIndex(0)
				require.NoError(t, err)
				v.ExitEpoch = 10
				require.NoError(t, st.UpdateValidatorAtIndex(0, v))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				amountAvailForProcessing := helpers.ActivationBalanceChurnLimit(8_000 * 1e9)
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
				// Validators 1..4 should have their balance increased
				for i := primitives.ValidatorIndex(1); i < 4; i++ {
					b, err := st.BalanceAtIndex(i)
					require.NoError(t, err)
					require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(amountAvailForProcessing)/5, b)
				}

				// All of the balance deposits should have been processed, except validator index 0 was
				// added back to the pending deposits queue.
				remaining, err := st.PendingDeposits()
				require.NoError(t, err)
				require.Equal(t, 1, len(remaining))
			},
		},
		{
			name: "exited validator balance increased",
			state: func() state.BeaconState {
				st := stateWithPendingDeposits(t, 8_000, 1, 8_000_000)
				v, err := st.ValidatorAtIndex(0)
				require.NoError(t, err)
				v.ExitEpoch = 2
				require.NoError(t, st.UpdateValidatorAtIndex(0, v))
				require.NoError(t, st.UpdateBalancesAtIndex(0, 800_000))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
				b, err := st.BalanceAtIndex(0)
				require.NoError(t, err)
				require.Equal(t, uint64(8_800_000), b)

				// All of the balance deposits should have been processed.
				remaining, err := st.PendingDeposits()
				require.NoError(t, err)
				require.Equal(t, 0, len(remaining))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tab uint64
			var err error
			if tt.state != nil {
				// The caller of this method would normally have the precompute balance values for total
				// active balance for this epoch. For ease of test setup, we will compute total active
				// balance from the given state.
				tab, err = helpers.TotalActiveBalance(tt.state)
			}
			require.NoError(t, err)
			err = electra.ProcessPendingDeposits(context.TODO(), tt.state, primitives.Gwei(tab))
			require.Equal(t, tt.wantErr, err != nil, "wantErr=%v, got err=%s", tt.wantErr, err)
			if tt.check != nil {
				tt.check(t, tt.state)
			}
		})
	}
}

func TestBatchProcessNewPendingDeposits(t *testing.T) {
	t.Run("invalid batch initiates correct individual validation", func(t *testing.T) {
		st := stateWithActiveBalanceETH(t, 0)
		require.Equal(t, 0, len(st.Validators()))
		require.Equal(t, 0, len(st.Balances()))
		sk, err := bls.RandKey()
		require.NoError(t, err)
		wc := make([]byte, 32)
		wc[31] = byte(0)
		validDep := stateTesting.GeneratePendingDeposit(t, sk, params.BeaconConfig().MinActivationBalance, bytesutil.ToBytes32(wc), 0)
		invalidDep := &eth.PendingDeposit{}
		// have a combination of valid and invalid deposits
		deps := []*eth.PendingDeposit{validDep, invalidDep}
		require.NoError(t, electra.BatchProcessNewPendingDeposits(context.Background(), st, deps))
		// successfully added to register
		require.Equal(t, 1, len(st.Validators()))
		require.Equal(t, 1, len(st.Balances()))
	})
}

func TestProcessDepositRequests(t *testing.T) {
	st, _ := util.DeterministicGenesisStateElectra(t, 1)
	sk, err := bls.RandKey()
	require.NoError(t, err)

	t.Run("empty requests continues", func(t *testing.T) {
		newSt, err := electra.ProcessDepositRequests(context.Background(), st, []*enginev1.DepositRequest{})
		require.NoError(t, err)
		require.DeepEqual(t, newSt, st)
	})
	t.Run("nil request errors", func(t *testing.T) {
		_, err = electra.ProcessDepositRequests(context.Background(), st, []*enginev1.DepositRequest{nil})
		require.ErrorContains(t, "nil deposit request", err)
	})

	vals := st.Validators()
	vals[0].PublicKey = sk.PublicKey().Marshal()
	require.NoError(t, st.SetValidators(vals))
	bals := st.Balances()
	bals[0] = params.BeaconConfig().MinActivationBalance + 2000
	require.NoError(t, st.SetBalances(bals))
	require.NoError(t, st.SetPendingDeposits(make([]*eth.PendingDeposit, 0))) // reset pbd as the determinitstic state populates this already
	withdrawalCred := make([]byte, 32)
	depositMessage := &eth.DepositMessage{
		PublicKey:             sk.PublicKey().Marshal(),
		Amount:                1000,
		WithdrawalCredentials: withdrawalCred,
	}
	domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(depositMessage, domain)
	require.NoError(t, err)
	sig := sk.Sign(sr[:])
	requests := []*enginev1.DepositRequest{
		{
			Pubkey:                depositMessage.PublicKey,
			Index:                 0,
			WithdrawalCredentials: depositMessage.WithdrawalCredentials,
			Amount:                depositMessage.Amount,
			Signature:             sig.Marshal(),
		},
	}
	st, err = electra.ProcessDepositRequests(context.Background(), st, requests)
	require.NoError(t, err)

	pbd, err := st.PendingDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pbd))
	require.Equal(t, uint64(1000), pbd[0].Amount)
}

// stateWithActiveBalanceETH generates a mock beacon state given a balance in eth
func stateWithActiveBalanceETH(t *testing.T, balETH uint64) state.BeaconState {
	gwei := balETH * 1_000_000_000
	balPerVal := params.BeaconConfig().MinActivationBalance
	numVals := gwei / balPerVal

	vals := make([]*eth.Validator, numVals)
	bals := make([]uint64, numVals)
	for i := uint64(0); i < numVals; i++ {
		wc := make([]byte, 32)
		wc[31] = byte(i)
		vals[i] = &eth.Validator{
			ActivationEpoch:       0,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:      balPerVal,
			WithdrawalCredentials: wc,
		}
		bals[i] = balPerVal
	}
	st, err := state_native.InitializeFromProtoUnsafeElectra(&eth.BeaconStateElectra{
		Slot:       10 * params.BeaconConfig().SlotsPerEpoch,
		Validators: vals,
		Balances:   bals,
		Fork: &eth.Fork{
			CurrentVersion: params.BeaconConfig().ElectraForkVersion,
		},
	})
	require.NoError(t, err)
	// set some fake finalized checkpoint
	require.NoError(t, st.SetFinalizedCheckpoint(&eth.Checkpoint{
		Epoch: 0,
		Root:  make([]byte, 32),
	}))
	return st
}

// stateWithPendingDeposits with pending deposits and existing ethbalance
func stateWithPendingDeposits(t *testing.T, balETH uint64, numDeposits, amount uint64) state.BeaconState {
	st := stateWithActiveBalanceETH(t, balETH)
	deps := make([]*eth.PendingDeposit, numDeposits)
	validators := st.Validators()
	for i := 0; i < len(deps); i += 1 {
		sk, err := bls.RandKey()
		require.NoError(t, err)
		wc := make([]byte, 32)
		wc[31] = byte(i)
		validators[i].PublicKey = sk.PublicKey().Marshal()
		validators[i].WithdrawalCredentials = wc
		deps[i] = stateTesting.GeneratePendingDeposit(t, sk, amount, bytesutil.ToBytes32(wc), 0)
	}
	require.NoError(t, st.SetValidators(validators))
	require.NoError(t, st.SetPendingDeposits(deps))
	return st
}

func TestApplyPendingDeposit_TopUp(t *testing.T) {
	excessBalance := uint64(100)
	st := stateWithActiveBalanceETH(t, 256)
	validators := st.Validators()
	sk, err := bls.RandKey()
	require.NoError(t, err)
	wc := make([]byte, 32)
	wc[31] = byte(0)
	validators[0].PublicKey = sk.PublicKey().Marshal()
	validators[0].WithdrawalCredentials = wc
	dep := stateTesting.GeneratePendingDeposit(t, sk, excessBalance, bytesutil.ToBytes32(wc), 0)
	dep.Signature = common.InfiniteSignature[:]
	require.NoError(t, st.SetValidators(validators))

	require.NoError(t, electra.ApplyPendingDeposit(context.Background(), st, dep))

	b, err := st.BalanceAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(excessBalance), b)
}

func TestApplyPendingDeposit_UnknownKey(t *testing.T) {
	st := stateWithActiveBalanceETH(t, 0)
	sk, err := bls.RandKey()
	require.NoError(t, err)
	wc := make([]byte, 32)
	wc[31] = byte(0)
	dep := stateTesting.GeneratePendingDeposit(t, sk, params.BeaconConfig().MinActivationBalance, bytesutil.ToBytes32(wc), 0)
	require.Equal(t, 0, len(st.Validators()))
	require.NoError(t, electra.ApplyPendingDeposit(context.Background(), st, dep))
	// activates new validator
	require.Equal(t, 1, len(st.Validators()))
	b, err := st.BalanceAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MinActivationBalance, b)
}

func TestApplyPendingDeposit_InvalidSignature(t *testing.T) {
	st := stateWithActiveBalanceETH(t, 0)

	sk, err := bls.RandKey()
	require.NoError(t, err)
	wc := make([]byte, 32)
	wc[31] = byte(0)
	dep := &eth.PendingDeposit{
		PublicKey:             sk.PublicKey().Marshal(),
		WithdrawalCredentials: wc,
		Amount:                100,
	}
	require.Equal(t, 0, len(st.Validators()))
	require.NoError(t, electra.ApplyPendingDeposit(context.Background(), st, dep))
	// no validator added
	require.Equal(t, 0, len(st.Validators()))
	// no topup either
	require.Equal(t, 0, len(st.Balances()))
}
