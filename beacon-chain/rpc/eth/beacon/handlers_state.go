package beacon

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

// GetStateRoot calculates HashTreeRoot for state with given 'stateId'. If stateId is root, same value will be returned.
func (s *Server) GetStateRoot(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetStateRoot")
	defer span.End()

	stateId := r.PathValue("state_id")
	if stateId == "" {
		httputil.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}

	stateRoot, err := s.Stater.StateRoot(ctx, []byte(stateId))
	if err != nil {
		var rootNotFoundErr *lookup.StateRootNotFoundError
		if errors.As(err, &rootNotFoundErr) {
			httputil.HandleError(w, "State root not found: "+rootNotFoundErr.Error(), http.StatusNotFound)
			return
		}
		httputil.HandleError(w, "Could not get state root: "+err.Error(), http.StatusInternalServerError)
		return
	}
	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	isOptimistic, err := helpers.IsOptimistic(ctx, []byte(stateId), s.OptimisticModeFetcher, s.Stater, s.ChainInfoFetcher, s.BeaconDB)
	if err != nil {
		httputil.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
		return
	}
	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		httputil.HandleError(w, "Could not calculate root of latest block header: "+err.Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	resp := &structs.GetStateRootResponse{
		Data: &structs.StateRoot{
			Root: hexutil.Encode(stateRoot),
		},
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
	}
	httputil.WriteJson(w, resp)
}

// GetRandao fetches the RANDAO mix for the requested epoch from the state identified by state_id.
// If an epoch is not specified then the RANDAO mix for the state's current epoch will be returned.
// By adjusting the state_id parameter you can query for any historic value of the RANDAO mix.
// Ordinarily states from the same epoch will mutate the RANDAO mix for that epoch as blocks are applied.
func (s *Server) GetRandao(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetRandao")
	defer span.End()

	stateId := r.PathValue("state_id")
	if stateId == "" {
		httputil.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}
	rawEpoch, e, ok := shared.UintFromQuery(w, r, "epoch", false)
	if !ok {
		return
	}

	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}

	stEpoch := slots.ToEpoch(st.Slot())
	epoch := stEpoch
	if rawEpoch != "" {
		epoch = primitives.Epoch(e)
	}

	// future epochs and epochs too far back are not supported.
	randaoEpochLowerBound := uint64(0)
	// Lower bound should not underflow.
	if uint64(stEpoch) > uint64(st.RandaoMixesLength()) {
		randaoEpochLowerBound = uint64(stEpoch) - uint64(st.RandaoMixesLength())
	}
	if epoch > stEpoch || uint64(epoch) < randaoEpochLowerBound+1 {
		httputil.HandleError(w, "Epoch is out of range for the randao mixes of the state", http.StatusBadRequest)
		return
	}
	idx := epoch % params.BeaconConfig().EpochsPerHistoricalVector
	randao, err := st.RandaoMixAtIndex(uint64(idx))
	if err != nil {
		httputil.HandleError(w, fmt.Sprintf("Could not get randao mix at index %d: %v", idx, err), http.StatusInternalServerError)
		return
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, []byte(stateId), s.OptimisticModeFetcher, s.Stater, s.ChainInfoFetcher, s.BeaconDB)
	if err != nil {
		httputil.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		httputil.HandleError(w, "Could not calculate root of latest block header: "+err.Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	resp := &structs.GetRandaoResponse{
		Data:                &structs.Randao{Randao: hexutil.Encode(randao)},
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
	}
	httputil.WriteJson(w, resp)
}
