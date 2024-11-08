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
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestGetWithdrawalEstimation(t *testing.T) {
	// first validator
	st, _ := util.DeterministicGenesisStateElectra(t, 10)
	firstVal, err := st.ValidatorAtIndex(0)
	require.NoError(t, err)

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
		{
			name:        "[error] pending partial withdrawals has zero length",
			validatorId: "0",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				return st
			}(),
			code:    http.StatusNotFound,
			wantErr: "could not find pending partial withdrawals for requested validator",
		},
		{
			name:        "[error] no pending partial withdrawals for corresponding validator",
			validatorId: "0",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				require.NoError(t, st.AppendPendingPartialWithdrawal(&ethpb.PendingPartialWithdrawal{
					Index:             1,
					Amount:            100,
					WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
				}))
				return st
			}(),
			code:    http.StatusNotFound,
			wantErr: "could not find pending partial withdrawals for requested validator",
		},
		{
			name:        "single pending partial withdrawal for corresponding validator",
			validatorId: "0",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				val, err := st.ValidatorAtIndex(primitives.ValidatorIndex(0))
				require.NoError(t, err)
				val.EffectiveBalance += 100_000_000_000 // 100 OVER
				val.PrincipalBalance += 100_000_000_000 // 100 OVER
				require.NoError(t, st.UpdateValidatorAtIndex(primitives.ValidatorIndex(0), val))

				bal, err := st.BalanceAtIndex(primitives.ValidatorIndex(0))
				require.NoError(t, err)

				bal += 100_000_000_000 // 100 OVER
				require.NoError(t, st.UpdateBalancesAtIndex(primitives.ValidatorIndex(0), bal))

				require.NoError(t, st.AppendPendingPartialWithdrawal(&ethpb.PendingPartialWithdrawal{
					Index:             0,
					Amount:            100_000_000_000, // 100 OVER
					WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
				}))
				return st
			}(),
			code: http.StatusOK,
			wantData: &structs.WithdrawalEstimationContainer{
				Pubkey: hexutil.Encode(firstVal.PublicKey),
				PendingPartialWithdrawals: []*structs.PendingPartialWithdrawalContainer{
					{
						Amount:        100_000_000_000, // 100 OVER
						ExpectedEpoch: uint64(params.BeaconConfig().MinValidatorWithdrawabilityDelay),
					},
				},
			},
		},
		{
			name:        "multiple pending partial withdrawal for corresponding validator",
			validatorId: "0",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)
				val, err := st.ValidatorAtIndex(primitives.ValidatorIndex(0))
				require.NoError(t, err)
				val.EffectiveBalance += 100_000_000_000 // 100 OVER
				val.PrincipalBalance += 100_000_000_000 // 100 OVER
				require.NoError(t, st.UpdateValidatorAtIndex(primitives.ValidatorIndex(0), val))

				bal, err := st.BalanceAtIndex(primitives.ValidatorIndex(0))
				require.NoError(t, err)

				bal += 100_000_000_000 // 100 OVER
				require.NoError(t, st.UpdateBalancesAtIndex(primitives.ValidatorIndex(0), bal))

				// Append same partial withdrawal twice
				require.NoError(t, st.AppendPendingPartialWithdrawal(&ethpb.PendingPartialWithdrawal{
					Index:             0,
					Amount:            50_000_000_000, // 50 OVER
					WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
				}))
				require.NoError(t, st.AppendPendingPartialWithdrawal(&ethpb.PendingPartialWithdrawal{
					Index:             0,
					Amount:            50_000_000_000, // 50 OVER
					WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
				}))
				return st
			}(),
			code: http.StatusOK,
			wantData: &structs.WithdrawalEstimationContainer{
				Pubkey: hexutil.Encode(firstVal.PublicKey),
				PendingPartialWithdrawals: []*structs.PendingPartialWithdrawalContainer{
					{
						Amount:        50_000_000_000, // 50 OVER
						ExpectedEpoch: uint64(params.BeaconConfig().MinValidatorWithdrawabilityDelay),
					},
					{
						Amount:        50_000_000_000, // 50 OVER
						ExpectedEpoch: uint64(params.BeaconConfig().MinValidatorWithdrawabilityDelay),
					},
				},
			},
		},
		{
			name:        "expected epoch is larger because of max pending partial withdrawals per sweep",
			validatorId: "0",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 10)

				// Increase validator 0's balance to 100 OVER
				val0, err := st.ValidatorAtIndex(primitives.ValidatorIndex(0))
				require.NoError(t, err)
				val0.EffectiveBalance += 100_000_000_000 // 100 OVER
				val0.PrincipalBalance += 100_000_000_000 // 100 OVER
				require.NoError(t, st.UpdateValidatorAtIndex(primitives.ValidatorIndex(0), val0))

				bal, err := st.BalanceAtIndex(primitives.ValidatorIndex(0))
				require.NoError(t, err)

				bal += 100_000_000_000 // 100 OVER
				require.NoError(t, st.UpdateBalancesAtIndex(primitives.ValidatorIndex(0), bal))

				// Increase validator 1's balance to 100 OVER
				val1, err := st.ValidatorAtIndex(primitives.ValidatorIndex(1))
				require.NoError(t, err)
				val1.EffectiveBalance += 100_000_000_000 // 100 OVER
				val1.PrincipalBalance += 100_000_000_000 // 100 OVER
				require.NoError(t, st.UpdateValidatorAtIndex(primitives.ValidatorIndex(1), val1))

				bal, err = st.BalanceAtIndex(primitives.ValidatorIndex(1))
				require.NoError(t, err)

				bal += 100_000_000_000 // 100 OVER
				require.NoError(t, st.UpdateBalancesAtIndex(primitives.ValidatorIndex(1), bal))

				// Append small partial withdrawal for validator 1, so that validator 0's partial withdrawal is not the first one
				maxPendingPartialsPerWithdrawalsSweep := int(params.BeaconConfig().MaxPendingPartialsPerWithdrawalsSweep) // casting is safe
				for i := 0; i < maxPendingPartialsPerWithdrawalsSweep; i++ {
					require.NoError(t, st.AppendPendingPartialWithdrawal(&ethpb.PendingPartialWithdrawal{
						Index:             1,
						Amount:            1_000_000_000, // 1 OVER
						WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
					}))
				}

				// Append target partial withdrawal for validator 0
				require.NoError(t, st.AppendPendingPartialWithdrawal(&ethpb.PendingPartialWithdrawal{
					Index:             0,
					Amount:            100_000_000_000, // 100 OVER
					WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
				}))
				return st
			}(),
			code: http.StatusOK,
			wantData: &structs.WithdrawalEstimationContainer{
				Pubkey: hexutil.Encode(firstVal.PublicKey),
				PendingPartialWithdrawals: []*structs.PendingPartialWithdrawalContainer{
					{
						Amount:        100_000_000_000,                                                    // 100 OVER
						ExpectedEpoch: uint64(params.BeaconConfig().MinValidatorWithdrawabilityDelay) + 1, // 1 epoch later
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
