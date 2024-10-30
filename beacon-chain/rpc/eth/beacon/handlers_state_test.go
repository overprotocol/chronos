package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/v5/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestGetStateRoot(t *testing.T) {
	ctx := context.Background()
	fakeState, err := util.NewBeaconState()
	require.NoError(t, err)
	stateRoot, err := fakeState.HashTreeRoot(ctx)
	require.NoError(t, err)
	db := dbTest.SetupDB(t)
	parentRoot := [32]byte{'a'}
	blk := util.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, db, blk)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	chainService := &chainMock.ChainService{}
	s := &Server{
		Stater: &testutil.MockStater{
			BeaconStateRoot: stateRoot[:],
			BeaconState:     fakeState,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		FinalizationFetcher:   chainService,
		BeaconDB:              db,
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com//eth/v1/beacon/states/{state_id}/root", nil)
	request.SetPathValue("state_id", "head")
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetStateRoot(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	resp := &structs.GetStateRootResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	assert.Equal(t, hexutil.Encode(stateRoot[:]), resp.Data.Root)

	t.Run("execution optimistic", func(t *testing.T) {
		chainService := &chainMock.ChainService{Optimistic: true}
		s := &Server{
			Stater: &testutil.MockStater{
				BeaconStateRoot: stateRoot[:],
				BeaconState:     fakeState,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		request := httptest.NewRequest(http.MethodGet, "http://example.com//eth/v1/beacon/states/{state_id}/root", nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetStateRoot(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetStateRootResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.DeepEqual(t, true, resp.ExecutionOptimistic)
	})

	t.Run("finalized", func(t *testing.T) {
		headerRoot, err := fakeState.LatestBlockHeader().HashTreeRoot()
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			FinalizedRoots: map[[32]byte]bool{
				headerRoot: true,
			},
		}
		s := &Server{
			Stater: &testutil.MockStater{
				BeaconStateRoot: stateRoot[:],
				BeaconState:     fakeState,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		request := httptest.NewRequest(http.MethodGet, "http://example.com//eth/v1/beacon/states/{state_id}/root", nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetStateRoot(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetStateRootResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.DeepEqual(t, true, resp.Finalized)
	})
}

func TestGetRandao(t *testing.T) {
	mixCurrent := bytesutil.ToBytes32([]byte("current"))
	mixOld := bytesutil.ToBytes32([]byte("old"))
	epochCurrent := primitives.Epoch(100000)
	epochOld := 100000 - params.BeaconConfig().EpochsPerHistoricalVector + 1

	ctx := context.Background()
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	// Set slot to epoch 100000
	require.NoError(t, st.SetSlot(params.BeaconConfig().SlotsPerEpoch*100000))
	require.NoError(t, st.UpdateRandaoMixesAtIndex(uint64(epochCurrent%params.BeaconConfig().EpochsPerHistoricalVector), mixCurrent))
	require.NoError(t, st.UpdateRandaoMixesAtIndex(uint64(epochOld%params.BeaconConfig().EpochsPerHistoricalVector), mixOld))

	headEpoch := primitives.Epoch(1)
	headSt, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headSt.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	headRandao := bytesutil.ToBytes32([]byte("head"))
	require.NoError(t, headSt.UpdateRandaoMixesAtIndex(uint64(headEpoch), headRandao))

	db := dbTest.SetupDB(t)
	chainService := &chainMock.ChainService{}
	s := &Server{
		Stater: &testutil.MockStater{
			BeaconState: st,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		FinalizationFetcher:   chainService,
		BeaconDB:              db,
	}

	t.Run("no epoch requested", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://example.com//eth/v1/beacon/states/{state_id}/randao", nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetRandao(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetRandaoResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.Equal(t, hexutil.Encode(mixCurrent[:]), resp.Data.Randao)
	})
	t.Run("current epoch requested", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com//eth/v1/beacon/states/{state_id}/randao?epoch=%d", epochCurrent), nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetRandao(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetRandaoResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.Equal(t, hexutil.Encode(mixCurrent[:]), resp.Data.Randao)
	})
	t.Run("old epoch requested", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com//eth/v1/beacon/states/{state_id}/randao?epoch=%d", epochOld), nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetRandao(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetRandaoResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.Equal(t, hexutil.Encode(mixOld[:]), resp.Data.Randao)
	})
	t.Run("head state below `EpochsPerHistoricalVector`", func(t *testing.T) {
		s.Stater = &testutil.MockStater{
			BeaconState: headSt,
		}

		request := httptest.NewRequest(http.MethodGet, "http://example.com//eth/v1/beacon/states/{state_id}/randao", nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetRandao(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetRandaoResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.Equal(t, hexutil.Encode(headRandao[:]), resp.Data.Randao)
	})
	t.Run("epoch too old", func(t *testing.T) {
		epochTooOld := primitives.Epoch(100000 - st.RandaoMixesLength())
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com//eth/v1/beacon/states/{state_id}/randao?epoch=%d", epochTooOld), nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetRandao(writer, request)
		require.Equal(t, http.StatusBadRequest, writer.Code)
		e := &httputil.DefaultJsonError{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		require.StringContains(t, "Epoch is out of range for the randao mixes of the state", e.Message)
	})
	t.Run("epoch in the future", func(t *testing.T) {
		futureEpoch := primitives.Epoch(100000 + 1)
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com//eth/v1/beacon/states/{state_id}/randao?epoch=%d", futureEpoch), nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetRandao(writer, request)
		require.Equal(t, http.StatusBadRequest, writer.Code)
		e := &httputil.DefaultJsonError{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		require.StringContains(t, "Epoch is out of range for the randao mixes of the state", e.Message)
	})
	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		s := &Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		request := httptest.NewRequest(http.MethodGet, "http://example.com//eth/v1/beacon/states/{state_id}/randao", nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetRandao(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetRandaoResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.DeepEqual(t, true, resp.ExecutionOptimistic)
	})
	t.Run("finalized", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		headerRoot, err := headSt.LatestBlockHeader().HashTreeRoot()
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			FinalizedRoots: map[[32]byte]bool{
				headerRoot: true,
			},
		}
		s := &Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		request := httptest.NewRequest(http.MethodGet, "http://example.com//eth/v1/beacon/states/{state_id}/randao", nil)
		request.SetPathValue("state_id", "head")
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetRandao(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetRandaoResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		assert.DeepEqual(t, true, resp.Finalized)
	})
}
