package rewards

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/wealdtech/go-bytesutil"
)

// BlockRewards is an HTTP handler for Beacon API getBlockRewards.
func (s *Server) BlockRewards(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.BlockRewards")
	defer span.End()
	segments := strings.Split(r.URL.Path, "/")
	blockId := segments[len(segments)-1]

	blk, err := s.Blocker.Block(r.Context(), []byte(blockId))
	if !shared.WriteBlockFetchError(w, blk, err) {
		return
	}

	if err := blocks.BeaconBlockIsNil(blk); err != nil {
		httputil.HandleError(w, fmt.Sprintf("block id %s was not found", blockId), http.StatusNotFound)
		return
	}

	if blk.Version() == version.Phase0 {
		httputil.HandleError(w, "Block rewards are not supported for Phase 0 blocks", http.StatusBadRequest)
		return
	}

	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		httputil.HandleError(w, "Could not get optimistic mode info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		httputil.HandleError(w, "Could not get block root: "+err.Error(), http.StatusInternalServerError)
		return
	}
	blockRewards, httpError := s.BlockRewardFetcher.GetBlockRewardsData(ctx, blk.Block())
	if httpError != nil {
		httputil.WriteError(w, httpError)
		return
	}
	response := &structs.BlockRewardsResponse{
		Data:                blockRewards,
		ExecutionOptimistic: optimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(ctx, blkRoot),
	}
	httputil.WriteJson(w, response)
}

// AttestationRewards retrieves attestation reward info for validators specified by array of public keys or validator index.
// If no array is provided, return reward info for every validator.
func (s *Server) AttestationRewards(w http.ResponseWriter, r *http.Request) {
	st, ok := s.attRewardsState(w, r)
	if !ok {
		return
	}
	bal, vals, valIndices, ok := attRewardsBalancesAndVals(w, r, st)
	if !ok {
		return
	}
	totalRewards, ok := totalAttRewards(w, st, bal, vals, valIndices)
	if !ok {
		return
	}
	idealRewards, ok := idealAttRewards(w, st, bal, vals)
	if !ok {
		return
	}

	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(r.Context())
	if err != nil {
		httputil.HandleError(w, "Could not get optimistic mode info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	blkRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		httputil.HandleError(w, "Could not get block root: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := &structs.AttestationRewardsResponse{
		Data: structs.AttestationRewards{
			IdealRewards: idealRewards,
			TotalRewards: totalRewards,
		},
		ExecutionOptimistic: optimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(r.Context(), blkRoot),
	}
	httputil.WriteJson(w, resp)
}

func (s *Server) attRewardsState(w http.ResponseWriter, r *http.Request) (state.BeaconState, bool) {
	segments := strings.Split(r.URL.Path, "/")
	requestedEpoch, err := strconv.ParseUint(segments[len(segments)-1], 10, 64)
	if err != nil {
		httputil.HandleError(w, "Could not decode epoch: "+err.Error(), http.StatusBadRequest)
		return nil, false
	}
	if primitives.Epoch(requestedEpoch) < params.BeaconConfig().AltairForkEpoch {
		httputil.HandleError(w, "Attestation rewards are not supported for Phase 0", http.StatusNotFound)
		return nil, false
	}
	currentEpoch := uint64(slots.ToEpoch(s.TimeFetcher.CurrentSlot()))
	if requestedEpoch+1 >= currentEpoch {
		httputil.HandleError(w,
			"Attestation rewards are available after two epoch transitions to ensure all attestations have a chance of inclusion",
			http.StatusNotFound)
		return nil, false
	}
	nextEpochEnd, err := slots.EpochEnd(primitives.Epoch(requestedEpoch + 1))
	if err != nil {
		httputil.HandleError(w, "Could not get next epoch's ending slot: "+err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	st, err := s.Stater.StateBySlot(r.Context(), nextEpochEnd)
	if err != nil {
		httputil.HandleError(w, "Could not get state for epoch's starting slot: "+err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	return st, true
}

func attRewardsBalancesAndVals(
	w http.ResponseWriter,
	r *http.Request,
	st state.BeaconState,
) (*precompute.Balance, []*precompute.Validator, []primitives.ValidatorIndex, bool) {
	allVals, bal, err := altair.InitializePrecomputeValidators(r.Context(), st)
	if err != nil {
		httputil.HandleError(w, "Could not initialize precompute validators: "+err.Error(), http.StatusBadRequest)
		return nil, nil, nil, false
	}
	allVals, bal, err = altair.ProcessEpochParticipation(r.Context(), st, bal, allVals)
	if err != nil {
		httputil.HandleError(w, "Could not process epoch participation: "+err.Error(), http.StatusBadRequest)
		return nil, nil, nil, false
	}
	valIndices, ok := requestedValIndices(w, r, st, allVals)
	if !ok {
		return nil, nil, nil, false
	}
	if len(valIndices) == len(allVals) {
		return bal, allVals, valIndices, true
	} else {
		filteredVals := make([]*precompute.Validator, len(valIndices))
		for i, valIx := range valIndices {
			filteredVals[i] = allVals[valIx]
		}
		return bal, filteredVals, valIndices, true
	}
}

// idealAttRewards returns rewards for hypothetical, perfectly voting validators
// whose effective balances are over minIdealBalance and match balances in passed in validators.
func idealAttRewards(
	w http.ResponseWriter,
	st state.BeaconState,
	bal *precompute.Balance,
	vals []*precompute.Validator,
) ([]structs.IdealAttestationReward, bool) {
	increment := params.BeaconConfig().EffectiveBalanceIncrement / 1e9
	var maxEffectiveBalance, minIdealBalance uint64
	if st.Version() < version.Electra {
		maxEffectiveBalance = params.BeaconConfig().MinActivationBalance / 1e9
		// Due to bail out and new penalty mechanism, validator will be exited
		// before touching few downstairs.
		minIdealBalance = maxEffectiveBalance - increment
	} else {
		maxEffectiveBalance = params.BeaconConfig().MaxEffectiveBalanceElectra / 1e9
		// Post-Electra, the range of effective balance becomes wider, but the lower bound will be same as pre-Electra.
		minIdealBalance = params.BeaconConfig().MinActivationBalance - increment
	}

	idealValsCount := (maxEffectiveBalance - minIdealBalance) / increment
	maxIdealBalance := maxEffectiveBalance

	idealRewards := make([]structs.IdealAttestationReward, 0, idealValsCount)
	idealVals := make([]*precompute.Validator, 0, idealValsCount)

	for i := minIdealBalance; i <= maxIdealBalance; i += increment {
		effectiveBalance := i * 1e9
		idealVals = append(idealVals, &precompute.Validator{
			IsActivePrevEpoch:            true,
			IsSlashed:                    false,
			CurrentEpochEffectiveBalance: effectiveBalance,
			IsPrevEpochSourceAttester:    true,
			IsPrevEpochTargetAttester:    true,
			IsPrevEpochHeadAttester:      true,
		})
		idealRewards = append(idealRewards, structs.IdealAttestationReward{EffectiveBalance: strconv.FormatUint(effectiveBalance, 10)})
	}

	deltas, _, err := altair.AttestationsDelta(st, bal, idealVals)
	if err != nil {
		httputil.HandleError(w, "Could not get attestations delta: "+err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	for i, d := range deltas {
		idealRewards[i].Head = strconv.FormatUint(d.HeadReward, 10)
		if d.SourcePenalty > 0 {
			idealRewards[i].Source = fmt.Sprintf("-%s", strconv.FormatUint(d.SourcePenalty, 10))
		} else {
			idealRewards[i].Source = strconv.FormatUint(d.SourceReward, 10)
		}
		if d.TargetPenalty > 0 {
			idealRewards[i].Target = fmt.Sprintf("-%s", strconv.FormatUint(d.TargetPenalty, 10))
		} else {
			idealRewards[i].Target = strconv.FormatUint(d.TargetReward, 10)
		}
	}
	return idealRewards, true
}

func totalAttRewards(
	w http.ResponseWriter,
	st state.BeaconState,
	bal *precompute.Balance,
	vals []*precompute.Validator,
	valIndices []primitives.ValidatorIndex,
) ([]structs.TotalAttestationReward, bool) {
	totalRewards := make([]structs.TotalAttestationReward, len(valIndices))
	for i, v := range valIndices {
		totalRewards[i] = structs.TotalAttestationReward{ValidatorIndex: strconv.FormatUint(uint64(v), 10)}
	}
	deltas, _, err := altair.AttestationsDelta(st, bal, vals)
	if err != nil {
		httputil.HandleError(w, "Could not get attestations delta: "+err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	for i, d := range deltas {
		totalRewards[i].Head = strconv.FormatUint(d.HeadReward, 10)
		if d.SourcePenalty > 0 {
			totalRewards[i].Source = fmt.Sprintf("-%s", strconv.FormatUint(d.SourcePenalty, 10))
		} else {
			totalRewards[i].Source = strconv.FormatUint(d.SourceReward, 10)
		}
		if d.TargetPenalty > 0 {
			totalRewards[i].Target = fmt.Sprintf("-%s", strconv.FormatUint(d.TargetPenalty, 10))
		} else {
			totalRewards[i].Target = strconv.FormatUint(d.TargetReward, 10)
		}
	}
	return totalRewards, true
}

func requestedValIndices(w http.ResponseWriter, r *http.Request, st state.BeaconState, allVals []*precompute.Validator) ([]primitives.ValidatorIndex, bool) {
	var rawValIds []string
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&rawValIds); err != nil {
			httputil.HandleError(w, "Could not decode validators: "+err.Error(), http.StatusBadRequest)
			return nil, false
		}
	}
	valIndices := make([]primitives.ValidatorIndex, len(rawValIds))
	for i, v := range rawValIds {
		index, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			pubkey, err := bytesutil.FromHexString(v)
			if err != nil || len(pubkey) != fieldparams.BLSPubkeyLength {
				httputil.HandleError(w, fmt.Sprintf("%s is not a validator index or pubkey", v), http.StatusBadRequest)
				return nil, false
			}
			var ok bool
			valIndices[i], ok = st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
			if !ok {
				httputil.HandleError(w, fmt.Sprintf("No validator index found for pubkey %#x", pubkey), http.StatusBadRequest)
				return nil, false
			}
		} else {
			if index >= uint64(st.NumValidators()) {
				httputil.HandleError(w, fmt.Sprintf("Validator index %d is too large. Maximum allowed index is %d", index, st.NumValidators()-1), http.StatusBadRequest)
				return nil, false
			}
			valIndices[i] = primitives.ValidatorIndex(index)
		}
	}
	if len(valIndices) == 0 {
		valIndices = make([]primitives.ValidatorIndex, len(allVals))
		for i := 0; i < len(allVals); i++ {
			valIndices[i] = primitives.ValidatorIndex(i)
		}
	}

	return valIndices, true
}
