package over

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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
		request.SetPathValue("epoch", "100")
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
		request.SetPathValue("epoch", "latest")
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
		request.SetPathValue("epoch", "100")
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
		request.SetPathValue("state_id", "head")
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
