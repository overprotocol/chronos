package over

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
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

// GetExitQueueEpoch calculates exit_queue_epoch for given exit_balance.
func (s *Server) GetExitQueueEpoch(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "over.GetExitQueueEpoch")
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

	// Parse exit_balance from URL params
	rawExitBalance := r.URL.Query().Get("exit_balance")
	if st.Version() >= version.Alpaca && rawExitBalance == "" {
		httputil.HandleError(w, "exit_balance is required for post-alpaca in query params", http.StatusBadRequest)
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

	copiedSt := st.Copy()

	var exitQueueEpoch primitives.Epoch
	var churn uint64
	// code is originated from InitiatorValidatorExit (beacon-chain/core/validators/validator.go)
	if copiedSt.Version() < version.Alpaca {
		exitQueueEpoch, churn = validators.MaxExitEpochAndChurn(copiedSt)
		exitableEpoch := helpers.ActivationExitEpoch(time.CurrentEpoch(copiedSt))
		if exitableEpoch > exitQueueEpoch {
			exitQueueEpoch = exitableEpoch
			churn = 0
		}
		activeValidatorCount, err := helpers.ActiveValidatorCount(ctx, copiedSt, time.CurrentEpoch(copiedSt))
		if err != nil {
			httputil.WriteError(w, handleWrapError(err, "could not get active validator count", http.StatusInternalServerError))
			return
		}
		currentChurn := helpers.ValidatorExitChurnLimit(activeValidatorCount)

		if churn >= currentChurn {
			exitQueueEpoch, err = exitQueueEpoch.SafeAdd(1)
			if err != nil {
				httputil.WriteError(w, handleWrapError(err, "could not add 1 to exit queue epoch", http.StatusInternalServerError))
				return
			}
		}
	} else {
		exitBalance, err := strconv.ParseUint(rawExitBalance, 10, 64)
		if err != nil {
			httputil.HandleError(w, "exit_balance must be a number", http.StatusBadRequest)
			return
		}

		exitQueueEpoch, err = calculateExitEpochForAlpaca(st, primitives.Gwei(exitBalance))
		if err != nil {
			httputil.WriteError(w, handleWrapError(err, "could not update exit epoch and churn", http.StatusInternalServerError))
			return
		}
	}

	httputil.WriteJson(w, &structs.GetExitQueueEpochResponse{
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
		Data: &structs.ExitQueueEpochContainer{
			ExitQueueEpoch: uint64(exitQueueEpoch),
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

// calculateExitEpochForAlpaca calculates exit queue epoch for post-alpaca state.
// This function has same logic with ExitEpochAndUpdateChurn in beacon-chain/state/state-native/setters_churn.go,
// but it doesn't mutate the state.
func calculateExitEpochForAlpaca(st state.ReadOnlyBeaconState, exitBalance primitives.Gwei) (primitives.Epoch, error) {
	if st.Version() < version.Alpaca {
		return 0, errors.New("exit epoch calculation is not supported for pre-alpaca states")
	}

	activeBal, err := helpers.TotalActiveBalance(st)
	if err != nil {
		return 0, err
	}
	earliestExitEpochFromState, err := st.EarliestExitEpoch()
	if err != nil {
		return 0, err
	}
	exitBalanceToConsumeFromState, err := st.ExitBalanceToConsume()
	if err != nil {
		return 0, err
	}
	earliestExitEpoch := max(earliestExitEpochFromState, helpers.ActivationExitEpoch(slots.ToEpoch(st.Slot())))
	perEpochChurn := helpers.ExitBalanceChurnLimit(primitives.Gwei(activeBal)) // Guaranteed to be non-zero.

	var exitBalanceToConsume primitives.Gwei
	if earliestExitEpochFromState < earliestExitEpoch {
		exitBalanceToConsume = perEpochChurn
	} else {
		exitBalanceToConsume = exitBalanceToConsumeFromState
	}

	if exitBalance > exitBalanceToConsume {
		balanceToProcess := exitBalance - exitBalanceToConsume
		additionalEpochs := primitives.Epoch((balanceToProcess-1)/perEpochChurn + 1)
		earliestExitEpoch += additionalEpochs
	}

	return earliestExitEpoch, nil
}
