// Package operation contains types for block operation-specific events fired during the runtime of a beacon node.
package operation

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

const (
	// UnaggregatedAttReceived is sent after an unaggregated attestation object has been received
	// from the outside world. (eg. in RPC or sync)
	UnaggregatedAttReceived = iota + 1

	// AggregatedAttReceived is sent after an aggregated attestation object has been received
	// from the outside world. (eg. in sync)
	AggregatedAttReceived

	// ExitReceived is sent after an voluntary exit object has been received from the outside world (eg in RPC or sync)
	ExitReceived

	// BlobSidecarReceived is sent after a blob sidecar is received from gossip or rpc.
	BlobSidecarReceived = 6

	// ProposerSlashingReceived is sent after a proposer slashing is received from gossip or rpc
	ProposerSlashingReceived = 7

	// AttesterSlashingReceived is sent after an attester slashing is received from gossip or rpc
	AttesterSlashingReceived = 8
)

// UnAggregatedAttReceivedData is the data sent with UnaggregatedAttReceived events.
type UnAggregatedAttReceivedData struct {
	// Attestation is the unaggregated attestation object.
	Attestation ethpb.Att
}

// AggregatedAttReceivedData is the data sent with AggregatedAttReceived events.
type AggregatedAttReceivedData struct {
	// Attestation is the aggregated attestation object.
	Attestation *ethpb.AggregateAttestationAndProof
}

// ExitReceivedData is the data sent with ExitReceived events.
type ExitReceivedData struct {
	// Exit is the voluntary exit object.
	Exit *ethpb.SignedVoluntaryExit
}

// BlobSidecarReceivedData is the data sent with BlobSidecarReceived events.
type BlobSidecarReceivedData struct {
	Blob *blocks.VerifiedROBlob
}

// ProposerSlashingReceivedData is the data sent with ProposerSlashingReceived events.
type ProposerSlashingReceivedData struct {
	ProposerSlashing *ethpb.ProposerSlashing
}

// AttesterSlashingReceivedData is the data sent with AttesterSlashingReceived events.
type AttesterSlashingReceivedData struct {
	AttesterSlashing ethpb.AttSlashing
}
