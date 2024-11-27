package beacon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/prysmaticlabs/prysm/v5/api"
	"github.com/prysmaticlabs/prysm/v5/api/server"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/operation"
	corehelpers "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

const broadcastBLSChangesRateLimit = 128

// ListAttestations retrieves attestations known by the node but
// not necessarily incorporated into any block. Allows filtering by committee index or slot.
func (s *Server) ListAttestations(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "beacon.ListAttestations")
	defer span.End()

	rawSlot, slot, ok := shared.UintFromQuery(w, r, "slot", false)
	if !ok {
		return
	}
	rawCommitteeIndex, committeeIndex, ok := shared.UintFromQuery(w, r, "committee_index", false)
	if !ok {
		return
	}

	attestations := s.AttestationsPool.AggregatedAttestations()
	unaggAtts, err := s.AttestationsPool.UnaggregatedAttestations()
	if err != nil {
		httputil.HandleError(w, "Could not get unaggregated attestations: "+err.Error(), http.StatusInternalServerError)
		return
	}
	attestations = append(attestations, unaggAtts...)
	isEmptyReq := rawSlot == "" && rawCommitteeIndex == ""
	if isEmptyReq {
		allAtts := make([]*structs.Attestation, len(attestations))
		for i, att := range attestations {
			a, ok := att.(*eth.Attestation)
			if ok {
				allAtts[i] = structs.AttFromConsensus(a)
			} else {
				httputil.HandleError(w, fmt.Sprintf("unable to convert attestations of type %T", att), http.StatusInternalServerError)
				return
			}
		}
		httputil.WriteJson(w, &structs.ListAttestationsResponse{Data: allAtts})
		return
	}

	bothDefined := rawSlot != "" && rawCommitteeIndex != ""
	filteredAtts := make([]*structs.Attestation, 0, len(attestations))
	for _, att := range attestations {
		committeeIndexMatch := rawCommitteeIndex != "" && att.GetData().CommitteeIndex == primitives.CommitteeIndex(committeeIndex)
		slotMatch := rawSlot != "" && att.GetData().Slot == primitives.Slot(slot)
		shouldAppend := (bothDefined && committeeIndexMatch && slotMatch) || (!bothDefined && (committeeIndexMatch || slotMatch))
		if shouldAppend {
			a, ok := att.(*eth.Attestation)
			if ok {
				filteredAtts = append(filteredAtts, structs.AttFromConsensus(a))
			} else {
				httputil.HandleError(w, fmt.Sprintf("unable to convert attestations of type %T", att), http.StatusInternalServerError)
				return
			}
		}
	}
	httputil.WriteJson(w, &structs.ListAttestationsResponse{Data: filteredAtts})
}

// SubmitAttestations submits an attestation object to node. If the attestation passes all validation
// constraints, node MUST publish the attestation on an appropriate subnet.
func (s *Server) SubmitAttestations(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.SubmitAttestations")
	defer span.End()

	var req structs.SubmitAttestationsRequest
	err := json.NewDecoder(r.Body).Decode(&req.Data)
	switch {
	case errors.Is(err, io.EOF):
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Data) == 0 {
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	}

	var validAttestations []*eth.Attestation
	var attFailures []*server.IndexedVerificationFailure
	for i, sourceAtt := range req.Data {
		att, err := sourceAtt.ToConsensus()
		if err != nil {
			attFailures = append(attFailures, &server.IndexedVerificationFailure{
				Index:   i,
				Message: "Could not convert request attestation to consensus attestation: " + err.Error(),
			})
			continue
		}
		if _, err = bls.SignatureFromBytes(att.Signature); err != nil {
			attFailures = append(attFailures, &server.IndexedVerificationFailure{
				Index:   i,
				Message: "Incorrect attestation signature: " + err.Error(),
			})
			continue
		}

		// Broadcast the unaggregated attestation on a feed to notify other services in the beacon node
		// of a received unaggregated attestation.
		// Note we can't send for aggregated att because we don't have selection proof.
		if !corehelpers.IsAggregated(att) {
			s.OperationNotifier.OperationFeed().Send(&feed.Event{
				Type: operation.UnaggregatedAttReceived,
				Data: &operation.UnAggregatedAttReceivedData{
					Attestation: att,
				},
			})
		}

		validAttestations = append(validAttestations, att)
	}

	failedBroadcasts := make([]string, 0)
	for i, att := range validAttestations {
		// Determine subnet to broadcast attestation to
		wantedEpoch := slots.ToEpoch(att.Data.Slot)
		vals, err := s.HeadFetcher.HeadValidatorsIndices(ctx, wantedEpoch)
		if err != nil {
			httputil.HandleError(w, "Could not get head validator indices: "+err.Error(), http.StatusInternalServerError)
			return
		}
		subnet := corehelpers.ComputeSubnetFromCommitteeAndSlot(uint64(len(vals)), att.Data.CommitteeIndex, att.Data.Slot)

		if err = s.Broadcaster.BroadcastAttestation(ctx, subnet, att); err != nil {
			log.WithError(err).Errorf("could not broadcast attestation at index %d", i)
		}

		if corehelpers.IsAggregated(att) {
			if err = s.AttestationsPool.SaveAggregatedAttestation(att); err != nil {
				log.WithError(err).Error("could not save aggregated attestation")
			}
		} else {
			if err = s.AttestationsPool.SaveUnaggregatedAttestation(att); err != nil {
				log.WithError(err).Error("could not save unaggregated attestation")
			}
		}
	}
	if len(failedBroadcasts) > 0 {
		httputil.HandleError(
			w,
			fmt.Sprintf("Attestations at index %s could not be broadcasted", strings.Join(failedBroadcasts, ", ")),
			http.StatusInternalServerError,
		)
		return
	}

	if len(attFailures) > 0 {
		failuresErr := &server.IndexedVerificationFailureError{
			Code:     http.StatusBadRequest,
			Message:  "One or more attestations failed validation",
			Failures: attFailures,
		}
		httputil.WriteError(w, failuresErr)
	}
}

// ListVoluntaryExits retrieves voluntary exits known by the node but
// not necessarily incorporated into any block.
func (s *Server) ListVoluntaryExits(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "beacon.ListVoluntaryExits")
	defer span.End()

	sourceExits, err := s.VoluntaryExitsPool.PendingExits()
	if err != nil {
		httputil.HandleError(w, "Could not get exits from the pool: "+err.Error(), http.StatusInternalServerError)
		return
	}
	exits := make([]*structs.SignedVoluntaryExit, len(sourceExits))
	for i, e := range sourceExits {
		exits[i] = structs.SignedExitFromConsensus(e)
	}

	httputil.WriteJson(w, &structs.ListVoluntaryExitsResponse{Data: exits})
}

// SubmitVoluntaryExit submits a SignedVoluntaryExit object to node's pool
// and if passes validation node MUST broadcast it to network.
func (s *Server) SubmitVoluntaryExit(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.SubmitVoluntaryExit")
	defer span.End()

	var req structs.SignedVoluntaryExit
	err := json.NewDecoder(r.Body).Decode(&req)
	switch {
	case errors.Is(err, io.EOF):
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	exit, err := req.ToConsensus()
	if err != nil {
		httputil.HandleError(w, "Could not convert request exit to consensus exit: "+err.Error(), http.StatusBadRequest)
		return
	}

	headState, err := s.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		httputil.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	epochStart, err := slots.EpochStart(exit.Exit.Epoch)
	if err != nil {
		httputil.HandleError(w, "Could not get epoch start: "+err.Error(), http.StatusInternalServerError)
		return
	}
	headState, err = transition.ProcessSlotsIfPossible(ctx, headState, epochStart)
	if err != nil {
		httputil.HandleError(w, "Could not process slots: "+err.Error(), http.StatusInternalServerError)
		return
	}
	val, err := headState.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
	if err != nil {
		if errors.Is(err, consensus_types.ErrOutOfBounds) {
			httputil.HandleError(w, "Could not get validator: "+err.Error(), http.StatusBadRequest)
			return
		}
		httputil.HandleError(w, "Could not get validator: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err = blocks.VerifyExitAndSignature(val, headState, exit); err != nil {
		httputil.HandleError(w, "Invalid exit: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.VoluntaryExitsPool.InsertVoluntaryExit(exit)
	if err = s.Broadcaster.Broadcast(ctx, exit); err != nil {
		httputil.HandleError(w, "Could not broadcast exit: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// broadcastBLSBatch broadcasts the first `broadcastBLSChangesRateLimit` messages from the slice pointed to by ptr.
// It validates the messages again because they could have been invalidated by being included in blocks since the last validation.
// It removes the messages from the slice and modifies it in place.
func (s *Server) broadcastBLSBatch(ctx context.Context, ptr *[]*eth.SignedBLSToExecutionChange) {
	limit := broadcastBLSChangesRateLimit
	if len(*ptr) < broadcastBLSChangesRateLimit {
		limit = len(*ptr)
	}
	st, err := s.ChainInfoFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		log.WithError(err).Error("could not get head state")
		return
	}
	for _, ch := range (*ptr)[:limit] {
		if ch != nil {
			_, err := blocks.ValidateBLSToExecutionChange(st, ch)
			if err != nil {
				log.WithError(err).Error("could not validate BLS to execution change")
				continue
			}
			if err := s.Broadcaster.Broadcast(ctx, ch); err != nil {
				log.WithError(err).Error("could not broadcast BLS to execution changes.")
			}
		}
	}
	*ptr = (*ptr)[limit:]
}

// ListBLSToExecutionChanges retrieves BLS to execution changes known by the node but not necessarily incorporated into any block
func (s *Server) ListBLSToExecutionChanges(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "beacon.ListBLSToExecutionChanges")
	defer span.End()

	sourceChanges, err := s.BLSChangesPool.PendingBLSToExecChanges()
	if err != nil {
		httputil.HandleError(w, fmt.Sprintf("Could not get BLS to execution changes: %v", err), http.StatusInternalServerError)
		return
	}

	httputil.WriteJson(w, &structs.BLSToExecutionChangesPoolResponse{
		Data: structs.SignedBLSChangesFromConsensus(sourceChanges),
	})
}

// GetAttesterSlashings retrieves attester slashings known by the node but
// not necessarily incorporated into any block.
func (s *Server) GetAttesterSlashings(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetAttesterSlashings")
	defer span.End()

	headState, err := s.ChainInfoFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		httputil.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sourceSlashings := s.SlashingsPool.PendingAttesterSlashings(ctx, headState, true /* return unlimited slashings */)
	slashings := make([]*structs.AttesterSlashing, len(sourceSlashings))
	for i, slashing := range sourceSlashings {
		as, ok := slashing.(*eth.AttesterSlashing)
		if !ok {
			httputil.HandleError(w, fmt.Sprintf("Unable to convert slashing of type %T", slashing), http.StatusInternalServerError)
			return
		}
		slashings[i] = structs.AttesterSlashingFromConsensus(as)
	}
	attBytes, err := json.Marshal(slashings)
	if err != nil {
		httputil.HandleError(w, fmt.Sprintf("Failed to marshal slashings: %v", err), http.StatusInternalServerError)
		return
	}
	httputil.WriteJson(w, &structs.GetAttesterSlashingsResponse{Data: attBytes})
}

// GetAttesterSlashingsV2 retrieves attester slashings known by the node but
// not necessarily incorporated into any block, supporting both AttesterSlashing and AttesterSlashingElectra.
func (s *Server) GetAttesterSlashingsV2(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetAttesterSlashingsV2")
	defer span.End()

	headState, err := s.ChainInfoFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		httputil.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var attStructs []interface{}
	sourceSlashings := s.SlashingsPool.PendingAttesterSlashings(ctx, headState, true /* return unlimited slashings */)
	for _, slashing := range sourceSlashings {
		if slashing.Version() >= version.Electra {
			a, ok := slashing.(*eth.AttesterSlashingElectra)
			if !ok {
				httputil.HandleError(w, fmt.Sprintf("Unable to convert electra slashing of type %T to an Electra slashing", slashing), http.StatusInternalServerError)
				return
			}
			attStruct := structs.AttesterSlashingElectraFromConsensus(a)
			attStructs = append(attStructs, attStruct)
		} else {
			a, ok := slashing.(*eth.AttesterSlashing)
			if !ok {
				httputil.HandleError(w, fmt.Sprintf("Unable to convert slashing of type %T to a Phase0 slashing", slashing), http.StatusInternalServerError)
				return
			}
			attStruct := structs.AttesterSlashingFromConsensus(a)
			attStructs = append(attStructs, attStruct)
		}
	}
	attBytes, err := json.Marshal(attStructs)
	if err != nil {
		httputil.HandleError(w, fmt.Sprintf("Failed to marshal slashing: %v", err), http.StatusInternalServerError)
		return
	}
	resp := &structs.GetAttesterSlashingsResponse{
		Version: version.String(sourceSlashings[0].Version()),
		Data:    attBytes,
	}
	httputil.WriteJson(w, resp)
}

// SubmitAttesterSlashings submits an attester slashing object to node's pool and
// if passes validation node MUST broadcast it to network.
func (s *Server) SubmitAttesterSlashings(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.SubmitAttesterSlashings")
	defer span.End()

	var req structs.AttesterSlashing
	err := json.NewDecoder(r.Body).Decode(&req)
	switch {
	case errors.Is(err, io.EOF):
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	slashing, err := req.ToConsensus()
	if err != nil {
		httputil.HandleError(w, "Could not convert request slashing to consensus slashing: "+err.Error(), http.StatusBadRequest)
		return
	}
	s.submitAttesterSlashing(w, ctx, slashing)
}

// SubmitAttesterSlashingsV2 submits an attester slashing object to node's pool and
// if passes validation node MUST broadcast it to network.
func (s *Server) SubmitAttesterSlashingsV2(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.SubmitAttesterSlashingsV2")
	defer span.End()

	versionHeader := r.Header.Get(api.VersionHeader)
	if versionHeader == "" {
		httputil.HandleError(w, api.VersionHeader+" header is required", http.StatusBadRequest)
	}
	v, err := version.FromString(versionHeader)
	if err != nil {
		httputil.HandleError(w, "Invalid version: "+err.Error(), http.StatusBadRequest)
		return
	}

	if v >= version.Electra {
		var req structs.AttesterSlashingElectra
		err := json.NewDecoder(r.Body).Decode(&req)
		switch {
		case errors.Is(err, io.EOF):
			httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
			return
		case err != nil:
			httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		slashing, err := req.ToConsensus()
		if err != nil {
			httputil.HandleError(w, "Could not convert request slashing to consensus slashing: "+err.Error(), http.StatusBadRequest)
			return
		}
		s.submitAttesterSlashing(w, ctx, slashing)
	} else {
		var req structs.AttesterSlashing
		err := json.NewDecoder(r.Body).Decode(&req)
		switch {
		case errors.Is(err, io.EOF):
			httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
			return
		case err != nil:
			httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		slashing, err := req.ToConsensus()
		if err != nil {
			httputil.HandleError(w, "Could not convert request slashing to consensus slashing: "+err.Error(), http.StatusBadRequest)
			return
		}
		s.submitAttesterSlashing(w, ctx, slashing)
	}
}

func (s *Server) submitAttesterSlashing(
	w http.ResponseWriter,
	ctx context.Context,
	slashing eth.AttSlashing,
) {
	headState, err := s.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		httputil.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	headState, err = transition.ProcessSlotsIfPossible(ctx, headState, slashing.FirstAttestation().GetData().Slot)
	if err != nil {
		httputil.HandleError(w, "Could not process slots: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = blocks.VerifyAttesterSlashing(ctx, headState, slashing)
	if err != nil {
		httputil.HandleError(w, "Invalid attester slashing: "+err.Error(), http.StatusBadRequest)
		return
	}
	err = s.SlashingsPool.InsertAttesterSlashing(ctx, headState, slashing)
	if err != nil {
		httputil.HandleError(w, "Could not insert attester slashing into pool: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// notify events
	s.OperationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.AttesterSlashingReceived,
		Data: &operation.AttesterSlashingReceivedData{
			AttesterSlashing: slashing,
		},
	})
	if !features.Get().DisableBroadcastSlashings {
		if err = s.Broadcaster.Broadcast(ctx, slashing); err != nil {
			httputil.HandleError(w, "Could not broadcast slashing object: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GetProposerSlashings retrieves proposer slashings known by the node
// but not necessarily incorporated into any block.
func (s *Server) GetProposerSlashings(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetProposerSlashings")
	defer span.End()

	headState, err := s.ChainInfoFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		httputil.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sourceSlashings := s.SlashingsPool.PendingProposerSlashings(ctx, headState, true /* return unlimited slashings */)
	slashings := structs.ProposerSlashingsFromConsensus(sourceSlashings)

	httputil.WriteJson(w, &structs.GetProposerSlashingsResponse{Data: slashings})
}

// SubmitProposerSlashing submits a proposer slashing object to node's pool and if
// passes validation node MUST broadcast it to network.
func (s *Server) SubmitProposerSlashing(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.SubmitProposerSlashing")
	defer span.End()

	var req structs.ProposerSlashing
	err := json.NewDecoder(r.Body).Decode(&req)
	switch {
	case errors.Is(err, io.EOF):
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	slashing, err := req.ToConsensus()
	if err != nil {
		httputil.HandleError(w, "Could not convert request slashing to consensus slashing: "+err.Error(), http.StatusBadRequest)
		return
	}
	headState, err := s.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		httputil.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	headState, err = transition.ProcessSlotsIfPossible(ctx, headState, slashing.Header_1.Header.Slot)
	if err != nil {
		httputil.HandleError(w, "Could not process slots: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = blocks.VerifyProposerSlashing(headState, slashing)
	if err != nil {
		httputil.HandleError(w, "Invalid proposer slashing: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = s.SlashingsPool.InsertProposerSlashing(ctx, headState, slashing)
	if err != nil {
		httputil.HandleError(w, "Could not insert proposer slashing into pool: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// notify events
	s.OperationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.ProposerSlashingReceived,
		Data: &operation.ProposerSlashingReceivedData{
			ProposerSlashing: slashing,
		},
	})

	if !features.Get().DisableBroadcastSlashings {
		if err = s.Broadcaster.Broadcast(ctx, slashing); err != nil {
			httputil.HandleError(w, "Could not broadcast slashing object: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
