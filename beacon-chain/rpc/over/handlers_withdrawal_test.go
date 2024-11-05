package over

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestGetWithdrawalEstimation(t *testing.T) {
	tests := []struct {
		name        string
		state       state.BeaconState
		validatorId string
		code        int
		wantData    *structs.WithdrawalEstimationContainer
		wantErr     string
	}{
		{
			name:        "[error] pre-electra is not supported",
			validatorId: "0",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateDeneb(t, 10)
				return st
			}(),
			code:    http.StatusBadRequest,
			wantErr: "Deposit estimation is not supported for pre-Electra.",
		},
		{
			name:        "[error] validator pubkey not found",
			validatorId: "0x93fc14e0e90ff2053dfe0543b70ed8c945b15133d9d75785d2452eff5ef1ef36d2ca07f0fba71562b6803c40c6b2ff43",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				return st
			}(),
			code:    http.StatusNotFound,
			wantErr: "Unknown validator",
		},
		{
			name:        "[error] validator id exceeds max validator index",
			validatorId: "11",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				return st
			}(),
			code:    http.StatusBadRequest,
			wantErr: "Invalid validator index",
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
			"http://example.com/over/v1/beacon/states/{state_id}/withdrawal_estimation/{validator_id}", nil)
		request.SetPathValue("state_id", "head")
		request.SetPathValue("validator_id", tt.validatorId)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetWithdrawalEstimation(writer, request)
		assert.Equal(t, tt.code, writer.Code)
		if tt.wantErr != "" {
			require.StringContains(t, tt.wantErr, writer.Body.String())
			continue
		}

		resp := &structs.GetWithdrawalEstimationResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		if tt.wantData != nil {
			require.DeepEqual(t, tt.wantData, resp.Data)
		}
	}
}
