package over

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

const (
	// pendingPartialWithdrawalResponseLimit is the maximum number of partial withdrawals that can be included in a response.
	pendingPartialWithdrawalResponseLimit = 100
)

// GetWithdrawalEstimation returns the estimated processing epoch for a validator's pending partial withdrawals.
// Accepts a validator ID as either a validator index or a public key.
// If there is no validator with the given ID, or if there are no pending partial withdrawals for the validator,
// the response will return a 404 status(not found) code.
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
	valId, ok := decodeId(w, rawId, st)
	if !ok {
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

	// Parse search_limit from URL params
	searchLimit := defaultSearchLimit
	rawSearchLimit := r.URL.Query().Get("search_limit")
	if rawSearchLimit != "" {
		_limit, err := strconv.Atoi(rawSearchLimit)
		if err != nil {
			httputil.HandleError(w, "search_limit must be a number", http.StatusBadRequest)
			return
		}
		if _limit < minSearchLimit {
			searchLimit = minSearchLimit
		} else if _limit > maxSearchLimit {
			searchLimit = maxSearchLimit
		} else {
			searchLimit = _limit
		}
	}

	ppws, err := st.PendingPartialWithdrawals()
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not get pending partial withdrawals from state", http.StatusInternalServerError))
		return
	}

	// Return early if there is no pending partial withdrawal
	if len(ppws) == 0 {
		httputil.WriteError(w, &httputil.DefaultJsonError{
			Message: errors.New("could not find pending partial withdrawals for requested validator").Error(),
			Code:    http.StatusNotFound,
		})
		return
	}

	// Limit the number of partial withdrawals to search
	ppws = ppws[:min(len(ppws), searchLimit)]

	val, err := st.ValidatorAtIndex(valId)
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not get validator at index", http.StatusInternalServerError))
		return
	}

	data := &structs.WithdrawalEstimationContainer{
		Pubkey: hexutil.Encode(val.PublicKey),
	}

	// Initialize variables
	currentEpoch := slots.ToEpoch(st.Slot())
	partialWithdrawalsCount := uint64(0)
	estimatedPendingPartialWithdrawals := make([]*structs.PendingPartialWithdrawalContainer, 0)

	// Iterate through pending partial withdrawals to estimate the expected epoch for requested validator
	for _, ppw := range ppws {
		if currentEpoch < ppw.WithdrawableEpoch {
			currentEpoch = ppw.WithdrawableEpoch
			partialWithdrawalsCount = 0
		}

		if partialWithdrawalsCount == params.BeaconConfig().MaxPendingPartialsPerWithdrawalsSweep {
			currentEpoch += 1
			partialWithdrawalsCount = 0
		}

		if ppw.Index == valId {
			cont := &structs.PendingPartialWithdrawalContainer{
				Amount:        ppw.Amount,
				ExpectedEpoch: uint64(currentEpoch),
			}
			estimatedPendingPartialWithdrawals = append(estimatedPendingPartialWithdrawals, cont)
			if len(estimatedPendingPartialWithdrawals) >= pendingPartialWithdrawalResponseLimit {
				// Limit the number of partial withdrawals to return
				break
			}
		}

		partialWithdrawalsCount++
	}

	if len(estimatedPendingPartialWithdrawals) == 0 {
		httputil.WriteError(w, &httputil.DefaultJsonError{
			Message: errors.New("could not find pending partial withdrawals for requested validator").Error(),
			Code:    http.StatusNotFound,
		})
		return
	}

	data.PendingPartialWithdrawals = estimatedPendingPartialWithdrawals

	httputil.WriteJson(w, &structs.GetWithdrawalEstimationResponse{
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
		Data:                data,
	})
}

// decodeId takes in a validator ID string (as either a pubkey or a validator index)
// and returns the corresponding validator index.
// From decodeIds in beacon-chain/rpc/eth/beacon/handlers_validator.go
func decodeId(w http.ResponseWriter, rawId string, st state.BeaconState) (primitives.ValidatorIndex, bool) {
	numVals := uint64(st.NumValidators())

	pubkey, err := hexutil.Decode(rawId)
	if err == nil {
		if len(pubkey) != fieldparams.BLSPubkeyLength {
			httputil.HandleError(w, fmt.Sprintf("Pubkey length is %d instead of %d", len(pubkey), fieldparams.BLSPubkeyLength), http.StatusBadRequest)
			return 0, false
		}
		valIndex, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
		if !ok {
			httputil.HandleError(w, fmt.Sprintf("Unknown validator: %s", hexutil.Encode(pubkey)), http.StatusNotFound)
			return 0, false
		}
		return valIndex, true
	}

	index, err := strconv.ParseUint(rawId, 10, 64)
	if err != nil {
		httputil.HandleError(w, fmt.Sprintf("Invalid validator index %s", rawId), http.StatusBadRequest)
		return 0, false
	}
	if index >= numVals {
		httputil.HandleError(w, fmt.Sprintf("Invalid validator index %d", index), http.StatusBadRequest)
		return 0, false
	}
	return primitives.ValidatorIndex(index), true
}
