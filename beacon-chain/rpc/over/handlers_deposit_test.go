package over

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestGetDepositEstimation(t *testing.T) {
	tests := []struct {
		name     string
		pubkey   string
		state    state.BeaconState
		code     int
		wantData *structs.DepositEstimationContainer
		wantErr  string
	}{
		{
			name:   "initial deposit",
			pubkey: "0x93fc14e0e90ff2053dfe0543b70ed8c945b15133d9d75785d2452eff5ef1ef36d2ca07f0fba71562b6803c40c6b2ff43",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 1000)
				require.NoError(t, st.SetSlot(322)) // current epoch = 10
				require.NoError(t, st.SetFinalizedCheckpoint(&ethpb.Checkpoint{
					Epoch: 10,
					Root:  []byte("finalized"),
				}))
				pubkey, err := hexutil.Decode("0x93fc14e0e90ff2053dfe0543b70ed8c945b15133d9d75785d2452eff5ef1ef36d2ca07f0fba71562b6803c40c6b2ff43")
				require.NoError(t, err)
				pd := &ethpb.PendingDeposit{
					PublicKey:             pubkey,
					WithdrawalCredentials: []byte("wc"),
					Amount:                params.BeaconConfig().MinActivationBalance,
					Slot:                  321, // second slot of Epoch 10
					Signature:             []byte("signature"),
				}
				require.NoError(t, st.AppendPendingDeposit(pd))
				return st
			}(),
			code: http.StatusOK,
			wantData: &structs.DepositEstimationContainer{
				Pubkey:    "0x93fc14e0e90ff2053dfe0543b70ed8c945b15133d9d75785d2452eff5ef1ef36d2ca07f0fba71562b6803c40c6b2ff43",
				Validator: nil,
				PendingDeposits: []*structs.PendingDepositEstimationContainer{
					{
						Type: "initial",
						Data: &structs.PendingDepositEstimation{
							Amount:                  params.BeaconConfig().MinActivationBalance,
							Slot:                    321,
							ExpectedEpoch:           11,
							ExpectedActivationEpoch: 20,
						},
					},
				},
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
