package iface

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/client/event"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

type BeaconCommitteeSelection struct {
	SelectionProof []byte
	Slot           primitives.Slot
	ValidatorIndex primitives.ValidatorIndex
}

type beaconCommitteeSelectionJson struct {
	SelectionProof string `json:"selection_proof"`
	Slot           string `json:"slot"`
	ValidatorIndex string `json:"validator_index"`
}

func (b *BeaconCommitteeSelection) MarshalJSON() ([]byte, error) {
	return json.Marshal(beaconCommitteeSelectionJson{
		SelectionProof: hexutil.Encode(b.SelectionProof),
		Slot:           strconv.FormatUint(uint64(b.Slot), 10),
		ValidatorIndex: strconv.FormatUint(uint64(b.ValidatorIndex), 10),
	})
}

func (b *BeaconCommitteeSelection) UnmarshalJSON(input []byte) error {
	var bjson beaconCommitteeSelectionJson
	err := json.Unmarshal(input, &bjson)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal beacon committee selection")
	}

	slot, err := strconv.ParseUint(bjson.Slot, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse slot")
	}

	vIdx, err := strconv.ParseUint(bjson.ValidatorIndex, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse validator index")
	}

	selectionProof, err := hexutil.Decode(bjson.SelectionProof)
	if err != nil {
		return errors.Wrap(err, "failed to parse selection proof")
	}

	b.Slot = primitives.Slot(slot)
	b.SelectionProof = selectionProof
	b.ValidatorIndex = primitives.ValidatorIndex(vIdx)

	return nil
}

type ValidatorClient interface {
	Duties(ctx context.Context, in *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error)
	DomainData(ctx context.Context, in *ethpb.DomainRequest) (*ethpb.DomainResponse, error)
	WaitForChainStart(ctx context.Context, in *empty.Empty) (*ethpb.ChainStartResponse, error)
	ValidatorIndex(ctx context.Context, in *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error)
	ValidatorStatus(ctx context.Context, in *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error)
	MultipleValidatorStatus(ctx context.Context, in *ethpb.MultipleValidatorStatusRequest) (*ethpb.MultipleValidatorStatusResponse, error)
	BeaconBlock(ctx context.Context, in *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error)
	ProposeBeaconBlock(ctx context.Context, in *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error)
	PrepareBeaconProposer(ctx context.Context, in *ethpb.PrepareBeaconProposerRequest) (*empty.Empty, error)
	FeeRecipientByPubKey(ctx context.Context, in *ethpb.FeeRecipientByPubKeyRequest) (*ethpb.FeeRecipientByPubKeyResponse, error)
	AttestationData(ctx context.Context, in *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error)
	ProposeAttestation(ctx context.Context, in *ethpb.Attestation) (*ethpb.AttestResponse, error)
	ProposeAttestationElectra(ctx context.Context, in *ethpb.AttestationElectra) (*ethpb.AttestResponse, error)
	SubmitAggregateSelectionProof(ctx context.Context, in *ethpb.AggregateSelectionRequest, index primitives.ValidatorIndex, committeeLength uint64) (*ethpb.AggregateSelectionResponse, error)
	SubmitAggregateSelectionProofElectra(ctx context.Context, in *ethpb.AggregateSelectionRequest, _ primitives.ValidatorIndex, _ uint64) (*ethpb.AggregateSelectionElectraResponse, error)
	SubmitSignedAggregateSelectionProof(ctx context.Context, in *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error)
	SubmitSignedAggregateSelectionProofElectra(ctx context.Context, in *ethpb.SignedAggregateSubmitElectraRequest) (*ethpb.SignedAggregateSubmitResponse, error)
	ProposeExit(ctx context.Context, in *ethpb.SignedVoluntaryExit) (*ethpb.ProposeExitResponse, error)
	SubscribeCommitteeSubnets(ctx context.Context, in *ethpb.CommitteeSubnetsSubscribeRequest, duties []*ethpb.DutiesResponse_Duty) (*empty.Empty, error)
	CheckDoppelGanger(ctx context.Context, in *ethpb.DoppelGangerRequest) (*ethpb.DoppelGangerResponse, error)
	SubmitValidatorRegistrations(ctx context.Context, in *ethpb.SignedValidatorRegistrationsV1) (*empty.Empty, error)
	StartEventStream(ctx context.Context, topics []string, eventsChannel chan<- *event.Event)
	EventStreamIsRunning() bool
	AggregatedSelections(ctx context.Context, selections []BeaconCommitteeSelection) ([]BeaconCommitteeSelection, error)
	Host() string
	SetHost(host string)
}
