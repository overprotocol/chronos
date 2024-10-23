package over

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func TestEstimatedActivation(t *testing.T) {
	st, _ := util.DeterministicGenesisState(t, 10)

	// two pending_queued validators
	const (
		pendingValidatorIndex1 = 8 // Status 2
		pendingValidatorIndex2 = 9 // Status 1

		activationEpoch            = 12
		activationEligibilityEpoch = 10
	)

	vals := st.Validators()
	vals[pendingValidatorIndex1].ActivationEpoch = activationEpoch
	vals[pendingValidatorIndex1].ActivationEligibilityEpoch = activationEligibilityEpoch
	vals[pendingValidatorIndex2].ActivationEpoch = params.BeaconConfig().FarFutureEpoch
	vals[pendingValidatorIndex2].ActivationEligibilityEpoch = activationEligibilityEpoch
	require.NoError(t, st.SetValidators(vals))

	chainService := &chainMock.ChainService{
		State: st,
	}

	s := &Server{
		HeadFetcher: chainService,
	}

	tests := []struct {
		name      string
		input     string
		want      *structs.EstimatedActivationResponse
		wantedErr string
	}{
		{
			name:  "empty request",
			input: "",
			want: &structs.EstimatedActivationResponse{
				WaitingEpoch:  uint64(5),
				EligibleEpoch: uint64(97),
				Status:        uint64(0),
			},
		},
		{
			name: "active validator (pubkey)",
			input: func(idx primitives.ValidatorIndex) string {
				pubkey0 := st.PubkeyAtIndex(idx)
				return hexutil.Encode(pubkey0[:])
			}(primitives.ValidatorIndex(0)),
			want: &structs.EstimatedActivationResponse{
				WaitingEpoch:  uint64(0),
				EligibleEpoch: uint64(0),
				Status:        uint64(3),
			},
		},
		{
			name: "active validator (pubkey without 0x prefix)",
			input: func(idx primitives.ValidatorIndex) string {
				pubkey0 := st.PubkeyAtIndex(idx)
				return hexutil.Encode(pubkey0[:])[2:]
			}(primitives.ValidatorIndex(0)),
			want: &structs.EstimatedActivationResponse{
				WaitingEpoch:  uint64(0),
				EligibleEpoch: uint64(0),
				Status:        uint64(3),
			},
		},
		{
			name:  "active validator (index)",
			input: "1",
			want: &structs.EstimatedActivationResponse{
				WaitingEpoch:  uint64(0),
				EligibleEpoch: uint64(0),
				Status:        uint64(3),
			},
		},
		{
			name:  "status one pending validator",
			input: strconv.Itoa(pendingValidatorIndex2),
			want: &structs.EstimatedActivationResponse{
				WaitingEpoch:  uint64(5),
				EligibleEpoch: uint64(activationEligibilityEpoch),
				Status:        uint64(1),
			},
		},
		{
			name:  "status two pending validator",
			input: strconv.Itoa(pendingValidatorIndex1),
			want: &structs.EstimatedActivationResponse{
				WaitingEpoch:  uint64(activationEpoch),
				EligibleEpoch: uint64(activationEligibilityEpoch),
				Status:        uint64(2),
			},
		},
		{
			name:  "unknown validator with appropriate input",
			input: strings.Repeat("0", 96),
			want: &structs.EstimatedActivationResponse{
				WaitingEpoch:  uint64(5),
				EligibleEpoch: uint64(97),
				Status:        uint64(0),
			},
		},
		{
			name:      "error while decoding validator id",
			input:     "dummy",
			wantedErr: "could not decode validator id from raw id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://example.com//chronos/validator/estimated_activation/{validator_id}", nil)
			request = mux.SetURLVars(request, map[string]string{"validator_id": tt.input})
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.EstimatedActivation(writer, request)

			if tt.wantedErr != "" {
				e := &httputil.DefaultJsonError{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
				assert.Equal(t, http.StatusBadRequest, e.Code)
				assert.StringContains(t, tt.wantedErr, e.Message)
			} else {
				require.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.EstimatedActivationResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				assert.DeepEqual(t, tt.want, resp)
			}
		})
	}
}

func TestEstimatedActivation_NoPendingValidators(t *testing.T) {
	st, _ := util.DeterministicGenesisState(t, 10)
	chainService := &chainMock.ChainService{
		State: st,
	}
	s := &Server{
		HeadFetcher: chainService,
	}

	t.Run("empty request", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "http://example.com//chronos/validator/estimated_activation/{validator_id}", nil)
		request = mux.SetURLVars(request, map[string]string{"validator_id": ""})
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.EstimatedActivation(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.EstimatedActivationResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		want := &structs.EstimatedActivationResponse{
			WaitingEpoch:  uint64(0),
			EligibleEpoch: uint64(97),
			Status:        uint64(0),
		}
		assert.DeepEqual(t, want, resp)
	})
}

func TestCalculateEligibleEpoch(t *testing.T) {
	tests := []struct {
		headSlot primitives.Slot
		want     uint64
	}{
		{headSlot: 93122, want: 2977},
		{headSlot: 93287, want: 3041},
		{headSlot: 107802, want: 3489},
		{headSlot: 120101, want: 3873},
		{headSlot: 141829, want: 4513},
		{headSlot: 156671, want: 4961},
		{headSlot: 156680, want: 5025},
		{headSlot: 156693, want: 5025},
	}
	for _, test := range tests {
		got := calculateEligibleEpoch(test.headSlot)
		assert.Equal(t, test.want, got, "Incorrect Eligible Epoch")
	}
}

func TestGetEpochReward(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	currentEpoch := primitives.Epoch(100)
	currentSlot, err := slots.EpochStart(currentEpoch)
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(currentSlot))

	t.Run("correctly get epoch reward when no boost", func(t *testing.T) {
		chainService := &chainMock.ChainService{Slot: &currentSlot, State: st, Optimistic: true}

		s := &Server{
			GenesisTimeFetcher: chainService,
			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
				currentSlot: st,
			}},
		}

		request := httptest.NewRequest(
			"GET", "/chronos/states/epoch_reward/{epoch}", nil)
		request = mux.SetURLVars(request, map[string]string{"epoch": "100"})
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetEpochReward(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.EpochReward{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))

		// minimum epoch issuance = 243531202435
		reward, _ := helpers.TotalRewardWithReserveUsage(st)
		want := strconv.Itoa(int(reward))
		assert.Equal(t, want, resp.Reward)
	})

	t.Run("handle latest epoch", func(t *testing.T) {
		chainService := &chainMock.ChainService{Slot: &currentSlot, State: st, Optimistic: true}

		s := &Server{
			GenesisTimeFetcher: chainService,
			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
				currentSlot: st,
			}},
		}

		request := httptest.NewRequest(
			"GET", "/chronos/states/epoch_reward/{epoch}", nil)
		request = mux.SetURLVars(request, map[string]string{"epoch": "latest"})
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetEpochReward(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.EpochReward{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))

		// minimum epoch issuance = 243531202435
		reward, _ := helpers.TotalRewardWithReserveUsage(st)
		want := strconv.Itoa(int(reward))
		assert.Equal(t, want, resp.Reward)
	})

	t.Run("correctly get epoch reward when boost", func(t *testing.T) {
		require.NoError(t, st.SetRewardAdjustmentFactor(uint64(20)))
		require.NoError(t, st.SetReserves(uint64(10000000)))

		chainService := &chainMock.ChainService{Slot: &currentSlot, State: st, Optimistic: true}

		s := &Server{
			GenesisTimeFetcher: chainService,
			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
				currentSlot: st,
			}},
		}

		request := httptest.NewRequest(
			"GET", "/chronos/states/epoch_reward/{epoch}", nil)
		request = mux.SetURLVars(request, map[string]string{"epoch": "100"})
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetEpochReward(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.EpochReward{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		// 243531202435 + feedback boost
		assert.Equal(t, "243533637747", resp.Reward)
	})

}

func TestGetReserves(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	currentSlot := primitives.Slot(5000)
	require.NoError(t, st.SetSlot(currentSlot))
	mockChainService := &chainMock.ChainService{Optimistic: true}

	t.Run("get correct reserves data", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)

		var (
			wantRewardAdjustmentFactor = uint64(200000)
			wantReserves               = uint64(10000000)
		)

		require.NoError(t, st.SetRewardAdjustmentFactor(wantRewardAdjustmentFactor))
		require.NoError(t, st.SetReserves(wantReserves))

		s := &Server{
			FinalizationFetcher:   mockChainService,
			OptimisticModeFetcher: mockChainService,
			Stater:                &testutil.MockStater{BeaconState: st},
		}
		request := httptest.NewRequest(
			"GET", "/over/v1/beacon/states/{state_id}/reserves", nil)
		request = mux.SetURLVars(request, map[string]string{"state_id": "head"})
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetReserves(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetReservesResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.Equal(t, true, resp.ExecutionOptimistic)
		assert.Equal(t, false, resp.Finalized)
		expectedReserves := &structs.Reserves{
			RewardAdjustmentFactor: strconv.FormatUint(wantRewardAdjustmentFactor, 10),
			Reserves:               strconv.FormatUint(wantReserves, 10),
		}
		require.DeepEqual(t, expectedReserves, resp.Data)
	})

	t.Run("return bad request", func(t *testing.T) {
		s := &Server{
			FinalizationFetcher:   mockChainService,
			OptimisticModeFetcher: mockChainService,
			Stater:                &testutil.MockStater{},
		}
		request := httptest.NewRequest(
			"GET", "/over/v1/beacon/states/{state_id}/reserves", nil)
		request = mux.SetURLVars(request, map[string]string{})
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetReserves(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &httputil.DefaultJsonError{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.StringContains(t, "state_id is required in URL params", e.Message)
	})
}

func Test_decodeValidatorId(t *testing.T) {
	st, _ := util.DeterministicGenesisState(t, 4)
	pubkey0 := st.PubkeyAtIndex(0)
	firstValPubkey := hexutil.Encode(pubkey0[:])

	tests := []struct {
		name    string
		rawId   string
		want    primitives.ValidatorIndex
		wantErr string
	}{
		{name: "Valid input (pubkey)", rawId: firstValPubkey, want: primitives.ValidatorIndex(0)},
		{name: "Valid input (validator index)", rawId: "0", want: primitives.ValidatorIndex(0)},
		{name: "Valid input (empty)", rawId: "", want: primitives.ValidatorIndex(math.MaxUint64)},
		{name: "Valid input (exceeding current validator set length)", rawId: "10", want: primitives.ValidatorIndex(math.MaxUint64)},
		{name: "Wrong input (wrong pubkey size)", rawId: firstValPubkey[:10] /*string of 4 bytes*/, wantErr: "Pubkey length is 4 instead of 48"},
		{name: "Wrong input (parse uint)", rawId: "dummy", wantErr: "could not parse validator id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valIndex, err := decodeValidatorId(st, tt.rawId)
			if tt.wantErr != "" {
				require.StringContains(t, tt.wantErr, err.Error())
				return
			}
			assert.Equal(t, tt.want, valIndex)
		})
	}
}
