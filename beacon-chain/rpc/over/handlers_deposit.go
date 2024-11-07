package over

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	valhelpers "github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

const (
	// Beacon chain is expected to be finalized after 2 epochs.
	// Use this constant to calculate estimation.
	expectedFinalityDelay = primitives.Epoch(2)
)

// GetDepositEstimation returns the deposit estimation for a given pubkey.
// By iterating through the validators and pending deposit queue, it calculates the expected epoch that
// the deposit will be processed.
// For initial deposit, it also calculates the expected activation epoch.
// If the validator is already in the registry, it will include the validator info in the response.
func (s *Server) GetDepositEstimation(w http.ResponseWriter, r *http.Request) {
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

	// Deposit estimation is only supported for Electra and later versions.
	if st.Version() < version.Electra {
		httputil.HandleError(w, "Deposit estimation is not supported for pre-Electra.", http.StatusBadRequest)
		return
	}

	// Parse pubkey from URL params
	rawPubkey := r.PathValue("pubkey")
	if rawPubkey == "" {
		httputil.HandleError(w, "pubkey is required in URL params", http.StatusBadRequest)
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

	// Decode pubkey
	hexId := append0x(rawPubkey)
	pubkey, err := hexutil.Decode(hexId)
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not decode pubkey", http.StatusInternalServerError))
		return
	}
	if len(pubkey) != fieldparams.BLSPubkeyLength {
		httputil.WriteError(w, handleWrapError(err, "invalid pubkey length", http.StatusInternalServerError))
		return
	}

	epoch := slots.ToEpoch(st.Slot())
	valIndex, found := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))

	data := &structs.DepositEstimationContainer{
		Pubkey: hexId,
	}

	// if the validator is found in registry, add the validator data to the response
	if found {
		val, err := st.ValidatorAtIndexReadOnly(valIndex)
		if err != nil {
			httputil.WriteError(w, handleWrapError(err, "could not get validator at index", http.StatusBadRequest))
			return
		}
		valSubStatus, err := valhelpers.ValidatorSubStatus(val, epoch)
		if err != nil {
			httputil.WriteError(w, handleWrapError(err, "could not get validator sub status", http.StatusBadRequest))
			return
		}

		expectedActivationEpoch := params.BeaconConfig().FarFutureEpoch
		if valSubStatus == validator.PendingInitialized && helpers.IsEligibleForActivationQueue(val, epoch) {
			estimatedActivationEligibilityEpoch := epoch + 1
			estimatedEligibleEpochForActivation := estimatedActivationEligibilityEpoch + expectedFinalityDelay
			expectedActivationEpoch = helpers.ActivationExitEpoch(estimatedEligibleEpochForActivation)
		} else if valSubStatus == validator.PendingQueued {
			if val.ActivationEpoch() == params.BeaconConfig().FarFutureEpoch {
				estimatedEligibleEpochForActivation := val.ActivationEligibilityEpoch() + expectedFinalityDelay
				expectedActivationEpoch = helpers.ActivationExitEpoch(estimatedEligibleEpochForActivation)
			} else {
				expectedActivationEpoch = val.ActivationEpoch()
			}
		}

		data.Validator = validatorFromROVal(val)
		// If validator is already in the registry and ready to be activated,
		// expectedActivationEpoch will be set to the epoch when the validator is assigned/expected to be activated.
		if expectedActivationEpoch < params.BeaconConfig().FarFutureEpoch {
			data.ExpectedActivationEpoch = uint64(expectedActivationEpoch)
		}
	}

	pdes, err := buildPendingDepositEstimations(st, pubkey, !found /* initial */)
	if err != nil {
		httputil.WriteError(w, handleWrapError(err, "could not build pending deposit estimations", http.StatusBadRequest))
		return
	}
	data.PendingDeposits = pdes

	httputil.WriteJson(w, &structs.GetDepositEstimationResponse{
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
		Data:                data,
	})
}

// buildPendingDepositEstimations iterates through the pending deposits and calculates the expected epoch.
func buildPendingDepositEstimations(st state.BeaconState, pubkey []byte, initial bool) ([]*structs.PendingDepositEstimationContainer, error) {
	activeBalance, err := helpers.TotalActiveBalance(st)
	if err != nil {
		return nil, errors.Wrap(err, "could not get total active balance")
	}
	balanceChurnLimit := helpers.ActivationBalanceChurnLimit(primitives.Gwei(activeBalance))

	pdes := make([]*structs.PendingDepositEstimationContainer, 0)
	pds, err := st.PendingDeposits()
	if err != nil {
		return nil, errors.Wrap(err, "could not get pending deposits from state")
	}

	// Return early if there are no pending deposits
	if len(pds) == 0 {
		return pdes, nil
	}

	finalizedEpoch := st.FinalizedCheckpointEpoch()
	finalizedSlot, err := slots.EpochStart(finalizedEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get finalized slot")
	}

	depBalToConsume, err := st.DepositBalanceToConsume()
	if err != nil {
		return nil, errors.Wrap(err, "could not get deposit balance to consume")
	}
	availableForProcessing := depBalToConsume + balanceChurnLimit

	currentEpoch := slots.ToEpoch(st.Slot())
	processedAmount := uint64(0)
	depositCount := uint64(0)

	// state.pending_deposits is guaranteed to be sorted by slot, so we can iterate through it in order
	// Mostly the same logic as in ProcessPendingDeposits, but it iterates through all pending deposits.
	for _, pd := range pds {
		// A pending deposit is processed when its slot is larger than finalized slot.
		// In this case, moving to the next epoch is needed. This will update and initialize the variables.
		// Premise: finalizedSlot will always be incremented by 32 slots(= 1 epoch).
		for pd.Slot > finalizedSlot {
			currentEpoch += 1

			availableForProcessing = balanceChurnLimit
			finalizedSlot += params.BeaconConfig().SlotsPerEpoch

			processedAmount = uint64(0)
			depositCount = uint64(0)
		}

		// Move to next epoch if max pending deposits per epoch is reached.
		if depositCount >= params.BeaconConfig().MaxPendingDepositsPerEpoch {
			currentEpoch += 1

			availableForProcessing = balanceChurnLimit
			finalizedSlot += params.BeaconConfig().SlotsPerEpoch

			processedAmount = uint64(0)
			depositCount = uint64(0)
		}

		var isValidatorExited bool
		var isValidatorWithdrawn bool
		index, found := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pd.PublicKey))
		if found {
			val, err := st.ValidatorAtIndexReadOnly(index)
			if err != nil {
				return nil, errors.Wrap(err, "could not get validator")
			}
			isValidatorExited = val.ExitEpoch() < params.BeaconConfig().FarFutureEpoch
			isValidatorWithdrawn = val.WithdrawableEpoch() < currentEpoch+1
		}

		if !isValidatorWithdrawn && !isValidatorExited {
			isChurnLimitReached := primitives.Gwei(processedAmount+pd.Amount) > availableForProcessing
			// If churn limit is reached, move to the next epoch.
			if isChurnLimitReached {
				currentEpoch += 1

				finalizedSlot += params.BeaconConfig().SlotsPerEpoch

				depBalToConsume = availableForProcessing - primitives.Gwei(processedAmount)
				// deposit_balance_to_consume only matters when the churn limit is reached.
				availableForProcessing = depBalToConsume + balanceChurnLimit
				processedAmount = uint64(0)
				depositCount = uint64(0)
			}

			processedAmount += pd.Amount
		}

		// Regardless of how the pendingDeposit was handled, we move on in the queue.
		depositCount++

		// If the pending deposit has same pubkey as the one we are looking for
		// append it to the response
		if bytes.Equal(pd.PublicKey, pubkey) {
			pde := &structs.PendingDepositEstimationContainer{}
			if initial {
				pde.Type = "initial"
			} else {
				pde.Type = "top-up"
			}

			data := &structs.PendingDepositEstimation{
				Amount: pd.Amount,
				Slot:   uint64(pd.Slot),
			}
			if isValidatorExited {
				// if the validator is already exited, it is hard to predict the expected epoch(postponed).
				data.ExpectedEpoch = uint64(params.BeaconConfig().FarFutureEpoch)
			} else {
				data.ExpectedEpoch = uint64(currentEpoch)
			}
			if initial && pd.Amount >= params.BeaconConfig().MinActivationBalance {
				estimatedActivationEligibilityEpoch := currentEpoch + 1
				estimatedEligibleEpochForActivation := estimatedActivationEligibilityEpoch + expectedFinalityDelay
				data.ExpectedActivationEpoch = uint64(helpers.ActivationExitEpoch(estimatedEligibleEpochForActivation))
			}
			pde.Data = data

			pdes = append(pdes, pde)
			initial = false // only one initial deposit is expected in the queue.
		}
	}

	return pdes, nil
}

func validatorFromROVal(val state.ReadOnlyValidator) *structs.Validator {
	return &structs.Validator{
		Pubkey:                     hexutil.Encode(bytesutil.FromBytes48(val.PublicKey())),
		WithdrawalCredentials:      hexutil.Encode(val.GetWithdrawalCredentials()),
		EffectiveBalance:           fmt.Sprintf("%d", val.EffectiveBalance()),
		Slashed:                    val.Slashed(),
		ActivationEligibilityEpoch: fmt.Sprintf("%d", val.ActivationEligibilityEpoch()),
		ActivationEpoch:            fmt.Sprintf("%d", val.ActivationEpoch()),
		ExitEpoch:                  fmt.Sprintf("%d", val.ExitEpoch()),
		WithdrawableEpoch:          fmt.Sprintf("%d", val.WithdrawableEpoch()),
		PrincipalBalance:           fmt.Sprintf("%d", val.PrincipalBalance()),
	}
}
