package over

import (
	"net/http"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func (s *Server) GetWithdrawalEstimation(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "over.GetDepositEstimation")
	defer span.End()

	// Parse state_id and replay to the state
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

	// Partial withdrawal estimation is only supported for Electra and later versions.
	if st.Version() < version.Electra {
		httputil.HandleError(w, "Deposit estimation is not supported for pre-Electra.", http.StatusBadRequest)
		return
	}

	// Parse validator_id from URL params
	rawId := r.PathValue("validator_id")
	if rawId == "" {
		httputil.HandleError(w, "validator_id is required in URL params", http.StatusBadRequest)
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
		httputil.WriteError(w, handleWrapError(err, "could not calculate root of latest block header", http.StatusInternalServerError))
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	httputil.WriteJson(w, &structs.GetWithdrawalEstimationResponse{
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
		Data:                &structs.WithdrawalEstimationContainer{},
	})
}
