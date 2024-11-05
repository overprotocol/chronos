package over

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls/common"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestGetDepositEstimation(t *testing.T) {
	// initial deposit keypair
	sk0, err := bls.RandKey()
	require.NoError(t, err)
	pubkey0 := hexutil.Encode(sk0.PublicKey().Marshal())

	// top-up deposit keypair
	st, privKeys := util.DeterministicGenesisStateElectra(t, 10)
	sk1 := privKeys[0]
	pubkey1 := hexutil.Encode(sk1.PublicKey().Marshal())

	val0, err := st.ValidatorAtIndex(0)
	require.NoError(t, err)

	tests := []struct {
		name     string
		pubkey   string
		state    state.BeaconState
		code     int
		wantData *structs.DepositEstimationContainer
		wantErr  string
	}{
		{
			name:   "[initial] initial deposit",
			pubkey: pubkey0,
			state: func() state.BeaconState {
				pd := &ethpb.PendingDeposit{
					PublicKey:             sk0.PublicKey().Marshal(),
					WithdrawalCredentials: make([]byte, 32),
					Amount:                params.BeaconConfig().MinActivationBalance,
					Slot:                  primitives.Slot(321),
				}
				pd, err = signedPendingDeposit(sk0, pd)
				require.NoError(t, err)

				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				require.NoError(t, st.SetSlot(384)) // current epoch = 12
				require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
					Epoch: 10,
					Root:  []byte("finalized"),
				}))
				require.NoError(t, st.AppendPendingDeposit(pd))
				return st
			}(),
			code: http.StatusOK,
			wantData: &structs.DepositEstimationContainer{
				Pubkey:    pubkey0,
				Validator: nil,
				PendingDeposits: []*structs.PendingDepositEstimationContainer{
					{
						Type: "initial",
						Data: &structs.PendingDepositEstimation{
							Amount:                  params.BeaconConfig().MinActivationBalance,
							Slot:                    321,
							ExpectedEpoch:           13,
							ExpectedActivationEpoch: 21,
						},
					},
				},
			},
		},
		{
			name:   "[initial] initial deposit was processed, validator's activation epoch is in the future",
			pubkey: pubkey0,
			state: func() state.BeaconState {
				pd := &ethpb.PendingDeposit{
					PublicKey:             sk0.PublicKey().Marshal(),
					WithdrawalCredentials: make([]byte, 32),
					Amount:                params.BeaconConfig().MinActivationBalance,
					Slot:                  primitives.Slot(321),
				}
				pd, err = signedPendingDeposit(sk0, pd)
				require.NoError(t, err)

				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				require.NoError(t, st.SetSlot(384)) // current epoch = 12
				require.NoError(t, st.AppendPendingDeposit(pd))
				require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
					Epoch: 12,
					Root:  []byte("finalized"),
				}))
				st, err = transition.ProcessSlots(context.Background(), st, 14*params.BeaconConfig().SlotsPerEpoch)
				require.NoError(t, err)
				return st
			}(),
			code: http.StatusOK,
			wantData: &structs.DepositEstimationContainer{
				Pubkey: pubkey0,
				Validator: structs.ValidatorFromConsensus(&ethpb.Validator{
					PublicKey:                  sk0.PublicKey().Marshal(),
					WithdrawalCredentials:      make([]byte, 32),
					EffectiveBalance:           params.BeaconConfig().MinActivationBalance,
					Slashed:                    false,
					ActivationEligibilityEpoch: 14,
					ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
					ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
					WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
					PrincipalBalance:           params.BeaconConfig().MinActivationBalance,
				}),
				PendingDeposits:         []*structs.PendingDepositEstimationContainer{},
				ExpectedActivationEpoch: 21,
			},
		},
		{
			name:   "[initial] initial deposit was processed, validator's activation epoch is set",
			pubkey: pubkey0,
			state: func() state.BeaconState {
				pd := &ethpb.PendingDeposit{
					PublicKey:             sk0.PublicKey().Marshal(),
					WithdrawalCredentials: make([]byte, 32),
					Amount:                params.BeaconConfig().MinActivationBalance,
					Slot:                  primitives.Slot(321),
				}
				pd, err = signedPendingDeposit(sk0, pd)
				require.NoError(t, err)

				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				require.NoError(t, st.SetSlot(384)) // current epoch = 12
				require.NoError(t, st.AppendPendingDeposit(pd))
				require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
					Epoch: 12,
					Root:  []byte("finalized"),
				}))
				st, err = transition.ProcessSlots(context.Background(), st, 16*params.BeaconConfig().SlotsPerEpoch)
				require.NoError(t, err)
				require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
					Epoch: 14,
					Root:  []byte("finalized"),
				}))
				st, err = transition.ProcessSlots(context.Background(), st, 17*params.BeaconConfig().SlotsPerEpoch)
				require.NoError(t, err)
				return st
			}(),
			code: http.StatusOK,
			wantData: &structs.DepositEstimationContainer{
				Pubkey: pubkey0,
				Validator: structs.ValidatorFromConsensus(&ethpb.Validator{
					PublicKey:                  sk0.PublicKey().Marshal(),
					WithdrawalCredentials:      make([]byte, 32),
					EffectiveBalance:           params.BeaconConfig().MinActivationBalance,
					Slashed:                    false,
					ActivationEligibilityEpoch: 14,
					ActivationEpoch:            21,
					ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
					WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
					PrincipalBalance:           params.BeaconConfig().MinActivationBalance,
				}),
				PendingDeposits:         []*structs.PendingDepositEstimationContainer{},
				ExpectedActivationEpoch: 21,
			},
		},
		{
			name:   "[top-up] top-up deposit",
			pubkey: pubkey1,
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)

				pd := &ethpb.PendingDeposit{
					PublicKey:             sk1.PublicKey().Marshal(),
					WithdrawalCredentials: make([]byte, 32),
					Amount:                1_000_000_000, // 1 OVER
					Slot:                  primitives.Slot(321),
				}
				pd, err = signedPendingDeposit(sk1, pd)
				require.NoError(t, err)

				require.NoError(t, st.SetSlot(384)) // current epoch = 12
				require.NoError(t, st.AppendPendingDeposit(pd))
				require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
					Epoch: 12,
					Root:  []byte("finalized"),
				}))
				return st
			}(),
			code: http.StatusOK,
			wantData: &structs.DepositEstimationContainer{
				Pubkey:    pubkey1,
				Validator: structs.ValidatorFromConsensus(val0),
				PendingDeposits: []*structs.PendingDepositEstimationContainer{{
					Type: "top-up",
					Data: &structs.PendingDepositEstimation{
						Amount:        1_000_000_000,
						Slot:          321,
						ExpectedEpoch: 12,
					},
				}},
			},
		},
		{
			name:   "[top-up] top-up deposit is processed",
			pubkey: pubkey1,
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)

				pd := &ethpb.PendingDeposit{
					PublicKey:             sk1.PublicKey().Marshal(),
					WithdrawalCredentials: make([]byte, 32),
					Amount:                1_000_000_000, // 1 OVER
					Slot:                  primitives.Slot(321),
				}
				pd, err = signedPendingDeposit(sk1, pd)
				require.NoError(t, err)

				require.NoError(t, st.SetSlot(384)) // current epoch = 12
				require.NoError(t, st.AppendPendingDeposit(pd))
				require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
					Epoch: 12,
					Root:  []byte("finalized"),
				}))
				st, err = transition.ProcessSlots(context.Background(), st, 13*params.BeaconConfig().SlotsPerEpoch)
				require.NoError(t, err)
				return st
			}(),
			code: http.StatusOK,
			wantData: func() *structs.DepositEstimationContainer {
				val := ethpb.CopyValidator(val0)
				val.PrincipalBalance += 1_000_000_000 // 1 OVER top-up
				return &structs.DepositEstimationContainer{
					Pubkey:          pubkey1,
					Validator:       structs.ValidatorFromConsensus(val),
					PendingDeposits: []*structs.PendingDepositEstimationContainer{},
				}
			}(),
		},
	}

	for _, tt := range tests {
		chainService := &chainMock.ChainService{}
		s := Server{
			Stater: &testutil.MockStater{
				BeaconState: tt.state,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
		}

		request := httptest.NewRequest(http.MethodGet,
			"http://example.com/over/v1/beacon/states/{state_id}/deposit_estimation/{pubkey}", nil)
		request.SetPathValue("state_id", "head")
		request.SetPathValue("pubkey", tt.pubkey)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetDepositEstimation(writer, request)
		assert.Equal(t, tt.code, writer.Code)
		if tt.wantErr != "" {
			require.StringContains(t, writer.Body.String(), tt.wantErr)
			continue
		}

		resp := &structs.GetDepositEstimationResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		if tt.wantData != nil {
			require.DeepEqual(t, tt.wantData, resp.Data)
		}
	}
}

func signedPendingDeposit(sk common.SecretKey, pd *ethpb.PendingDeposit) (*ethpb.PendingDeposit, error) {
	domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	if err != nil {
		return nil, err
	}
	sr, err := signing.ComputeSigningRoot(&ethpb.DepositMessage{
		PublicKey:             pd.PublicKey,
		WithdrawalCredentials: pd.WithdrawalCredentials,
		Amount:                pd.Amount,
	}, domain)
	if err != nil {
		return nil, err
	}
	sig := sk.Sign(sr[:])
	pd.Signature = sig.Marshal()

	return pd, nil
}
