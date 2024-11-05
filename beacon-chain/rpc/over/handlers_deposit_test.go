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
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestGetDepositEstimation(t *testing.T) {
	pd, pubkeyBytes, err := generateValidPendingDeposit(321)
	require.NoError(t, err)
	pubkey := hexutil.Encode(pubkeyBytes)

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
			pubkey: pubkey,
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 1000)
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
				Pubkey:    pubkey,
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
			pubkey: pubkey,
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 1000)
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
				Pubkey: pubkey,
				Validator: structs.ValidatorFromConsensus(&ethpb.Validator{
					PublicKey:                  pubkeyBytes,
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
			pubkey: pubkey,
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 1000)
				require.NoError(t, st.SetSlot(384)) // current epoch = 12
				require.NoError(t, st.AppendPendingDeposit(pd))
				require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
					Epoch: 12,
					Root:  []byte("finalized"),
				}))
				st, err = transition.ProcessSlots(context.Background(), st, 14*params.BeaconConfig().SlotsPerEpoch)
				require.NoError(t, err)
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
				Pubkey: pubkey,
				Validator: structs.ValidatorFromConsensus(&ethpb.Validator{
					PublicKey:                  pubkeyBytes,
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

func generateValidPendingDeposit(slot uint64) (*ethpb.PendingDeposit, []byte, error) {
	sk, err := bls.RandKey()
	if err != nil {
		return nil, nil, err
	}
	domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	pd := &ethpb.PendingDeposit{
		PublicKey:             sk.PublicKey().Marshal(),
		WithdrawalCredentials: make([]byte, 32),
		Amount:                params.BeaconConfig().MinActivationBalance,
		Slot:                  primitives.Slot(slot),
	}
	sr, err := signing.ComputeSigningRoot(&ethpb.DepositMessage{
		PublicKey:             pd.PublicKey,
		WithdrawalCredentials: pd.WithdrawalCredentials,
		Amount:                params.BeaconConfig().MinActivationBalance,
	}, domain)
	if err != nil {
		return nil, nil, err
	}
	sig := sk.Sign(sr[:])
	pd.Signature = sig.Marshal()

	return pd, sk.PublicKey().Marshal(), nil
}
