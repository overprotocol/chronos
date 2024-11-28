package over

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"go.opencensus.io/trace"
)

// GetEpochReward returns total reward at given epoch.
func (s *Server) GetEpochReward(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "over.GetEpochReward")
	defer span.End()

	var requestedEpoch primitives.Epoch
	epochId := r.PathValue("epoch")
	curEpoch := slots.ToEpoch(s.GenesisTimeFetcher.CurrentSlot())

	if epochId == "latest" {
		requestedEpoch = curEpoch
	} else {
		uintEpoch, err := strconv.ParseUint(epochId, 10, 64)
		if err != nil {
			httputil.HandleError(w, "Could not parse uint: "+err.Error(), http.StatusBadRequest)
			return
		}
		if uintEpoch > uint64(curEpoch) {
			httputil.HandleError(w, fmt.Sprintf("Cannot retrieve information for an future epoch, current epoch %d, requesting %d", curEpoch, uintEpoch), http.StatusBadRequest)
			return
		}
		requestedEpoch = primitives.Epoch(uintEpoch)
	}

	slot, err := slots.EpochStart(requestedEpoch)
	if err != nil {
		httputil.HandleError(w, "Could not get start slot of requested : "+err.Error(), http.StatusBadRequest)
		return
	}

	reqState, err := s.Stater.StateBySlot(ctx, slot)
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not retrieve state", http.StatusNotFound))
		return
	}

	reward, _ := helpers.TotalRewardWithReserveUsage(reqState)

	httputil.WriteJson(w, &structs.EpochReward{
		Reward: strconv.FormatUint(reward, 10),
	})
}

// GetReserves get reserves data from the requested state.
// e.g. RewardAdjustmentFactor, Reserves, etc.
func (s *Server) GetReserves(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "over.GetReserves")
	defer span.End()

	// Retrieve beacon state
	stateId := r.PathValue("state_id")
	if stateId == "" {
		httputil.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}
	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not retrieve state", http.StatusNotFound))
		return
	}

	// Get metadata for response
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not get optimistic mode info", http.StatusInternalServerError))
		return
	}
	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		httputil.HandleError(w, errors.Wrap(err, "Could not calculate root of latest block header: ").Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	rewardAdjustmentFactor := st.RewardAdjustmentFactor()
	reserves := st.Reserves()

	httputil.WriteJson(w, &structs.GetReservesResponse{
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
		Data: &structs.Reserves{
			RewardAdjustmentFactor: strconv.FormatUint(rewardAdjustmentFactor, 10),
			Reserves:               strconv.FormatUint(reserves, 10),
		},
	})
}

func handleWrapError(err error, message string, code int) *httputil.DefaultJsonError {
	return &httputil.DefaultJsonError{
		Message: errors.Wrapf(err, message).Error(),
		Code:    code,
	}
}

func append0x(input string) string {
	if has0xPrefix(input) {
		return input
	}
	return "0x" + input
}

func has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}
