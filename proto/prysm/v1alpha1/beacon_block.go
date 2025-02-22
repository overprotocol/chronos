package eth

import (
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
)

// Copy --
func (sigBlock *SignedBeaconBlock) Copy() *SignedBeaconBlock {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlock{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BeaconBlock) Copy() *BeaconBlock {
	if block == nil {
		return nil
	}
	return &BeaconBlock{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BeaconBlockBody) Copy() *BeaconBlockBody {
	if body == nil {
		return nil
	}
	return &BeaconBlockBody{
		RandaoReveal:      bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:          body.Eth1Data.Copy(),
		Graffiti:          bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings: CopySlice(body.ProposerSlashings),
		AttesterSlashings: CopySlice(body.AttesterSlashings),
		Attestations:      CopySlice(body.Attestations),
		Deposits:          CopySlice(body.Deposits),
		VoluntaryExits:    CopySlice(body.VoluntaryExits),
	}
}

// Copy --
func (sigBlock *SignedBeaconBlockAltair) Copy() *SignedBeaconBlockAltair {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockAltair{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BeaconBlockAltair) Copy() *BeaconBlockAltair {
	if block == nil {
		return nil
	}
	return &BeaconBlockAltair{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BeaconBlockBodyAltair) Copy() *BeaconBlockBodyAltair {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyAltair{
		RandaoReveal:      bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:          body.Eth1Data.Copy(),
		Graffiti:          bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings: CopySlice(body.ProposerSlashings),
		AttesterSlashings: CopySlice(body.AttesterSlashings),
		Attestations:      CopySlice(body.Attestations),
		Deposits:          CopySlice(body.Deposits),
		VoluntaryExits:    CopySlice(body.VoluntaryExits),
	}
}

// Copy --
func (sigBlock *SignedBeaconBlockBellatrix) Copy() *SignedBeaconBlockBellatrix {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockBellatrix{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BeaconBlockBellatrix) Copy() *BeaconBlockBellatrix {
	if block == nil {
		return nil
	}
	return &BeaconBlockBellatrix{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BeaconBlockBodyBellatrix) Copy() *BeaconBlockBodyBellatrix {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyBellatrix{
		RandaoReveal:      bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:          body.Eth1Data.Copy(),
		Graffiti:          bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings: CopySlice(body.ProposerSlashings),
		AttesterSlashings: CopySlice(body.AttesterSlashings),
		Attestations:      CopySlice(body.Attestations),
		Deposits:          CopySlice(body.Deposits),
		VoluntaryExits:    CopySlice(body.VoluntaryExits),
		ExecutionPayload:  body.ExecutionPayload.Copy(),
	}
}

// Copy --
func (sigBlock *SignedBeaconBlockCapella) Copy() *SignedBeaconBlockCapella {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockCapella{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BeaconBlockCapella) Copy() *BeaconBlockCapella {
	if block == nil {
		return nil
	}
	return &BeaconBlockCapella{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BeaconBlockBodyCapella) Copy() *BeaconBlockBodyCapella {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyCapella{
		RandaoReveal:      bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:          body.Eth1Data.Copy(),
		Graffiti:          bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings: CopySlice(body.ProposerSlashings),
		AttesterSlashings: CopySlice(body.AttesterSlashings),
		Attestations:      CopySlice(body.Attestations),
		Deposits:          CopySlice(body.Deposits),
		VoluntaryExits:    CopySlice(body.VoluntaryExits),
		ExecutionPayload:  body.ExecutionPayload.Copy(),
	}
}

// Copy --
func (sigBlock *SignedBlindedBeaconBlockCapella) Copy() *SignedBlindedBeaconBlockCapella {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockCapella{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BlindedBeaconBlockCapella) Copy() *BlindedBeaconBlockCapella {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockCapella{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BlindedBeaconBlockBodyCapella) Copy() *BlindedBeaconBlockBodyCapella {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyCapella{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
	}
}

// Copy --
func (sigBlock *SignedBlindedBeaconBlockDeneb) Copy() *SignedBlindedBeaconBlockDeneb {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockDeneb{
		Message:   sigBlock.Message.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BlindedBeaconBlockDeneb) Copy() *BlindedBeaconBlockDeneb {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockDeneb{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BlindedBeaconBlockBodyDeneb) Copy() *BlindedBeaconBlockBodyDeneb {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyDeneb{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
		BlobKzgCommitments:     CopyBlobKZGs(body.BlobKzgCommitments),
	}
}

// Copy --
func (sigBlock *SignedBlindedBeaconBlockElectra) Copy() *SignedBlindedBeaconBlockElectra {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockElectra{
		Message:   sigBlock.Message.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BlindedBeaconBlockElectra) Copy() *BlindedBeaconBlockElectra {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockElectra{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BlindedBeaconBlockBodyElectra) Copy() *BlindedBeaconBlockBodyElectra {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyElectra{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
		BlobKzgCommitments:     CopyBlobKZGs(body.BlobKzgCommitments),
		ExecutionRequests:      CopyExecutionRequests(body.ExecutionRequests),
	}
}

// Copy --
func (sigBlock *SignedBlindedBeaconBlockBadger) Copy() *SignedBlindedBeaconBlockBadger {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockBadger{
		Message:   sigBlock.Message.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BlindedBeaconBlockBadger) Copy() *BlindedBeaconBlockBadger {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockBadger{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BlindedBeaconBlockBodyBadger) Copy() *BlindedBeaconBlockBodyBadger {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyBadger{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
		BlobKzgCommitments:     CopyBlobKZGs(body.BlobKzgCommitments),
		ExecutionRequests:      CopyExecutionRequests(body.ExecutionRequests),
	}
}

// Copy --
func (sigBlock *SignedBlindedBeaconBlockBellatrix) Copy() *SignedBlindedBeaconBlockBellatrix {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockBellatrix{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BlindedBeaconBlockBellatrix) Copy() *BlindedBeaconBlockBellatrix {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockBellatrix{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BlindedBeaconBlockBodyBellatrix) Copy() *BlindedBeaconBlockBodyBellatrix {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyBellatrix{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
	}
}

// CopyBlobKZGs copies the provided blob kzgs object.
func CopyBlobKZGs(b [][]byte) [][]byte {
	return bytesutil.SafeCopy2dBytes(b)
}

// CopyExecutionRequests copies the provided execution requests.
func CopyExecutionRequests(e *enginev1.ExecutionRequests) *enginev1.ExecutionRequests {
	if e == nil {
		return nil
	}
	dr := make([]*enginev1.DepositRequest, len(e.Deposits))
	for i, d := range e.Deposits {
		dr[i] = d.Copy()
	}
	wr := make([]*enginev1.WithdrawalRequest, len(e.Withdrawals))
	for i, w := range e.Withdrawals {
		wr[i] = w.Copy()
	}

	return &enginev1.ExecutionRequests{
		Deposits:    dr,
		Withdrawals: wr,
	}
}

// Copy --
func (sigBlock *SignedBeaconBlockDeneb) Copy() *SignedBeaconBlockDeneb {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockDeneb{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BeaconBlockDeneb) Copy() *BeaconBlockDeneb {
	if block == nil {
		return nil
	}
	return &BeaconBlockDeneb{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BeaconBlockBodyDeneb) Copy() *BeaconBlockBodyDeneb {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyDeneb{
		RandaoReveal:       bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:           body.Eth1Data.Copy(),
		Graffiti:           bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:  CopySlice(body.ProposerSlashings),
		AttesterSlashings:  CopySlice(body.AttesterSlashings),
		Attestations:       CopySlice(body.Attestations),
		Deposits:           CopySlice(body.Deposits),
		VoluntaryExits:     CopySlice(body.VoluntaryExits),
		ExecutionPayload:   body.ExecutionPayload.Copy(),
		BlobKzgCommitments: CopyBlobKZGs(body.BlobKzgCommitments),
	}
}

// Copy --
func (sigBlock *SignedBeaconBlockElectra) Copy() *SignedBeaconBlockElectra {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockElectra{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BeaconBlockElectra) Copy() *BeaconBlockElectra {
	if block == nil {
		return nil
	}
	return &BeaconBlockElectra{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BeaconBlockBodyElectra) Copy() *BeaconBlockBodyElectra {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyElectra{
		RandaoReveal:       bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:           body.Eth1Data.Copy(),
		Graffiti:           bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:  CopySlice(body.ProposerSlashings),
		AttesterSlashings:  CopySlice(body.AttesterSlashings),
		Attestations:       CopySlice(body.Attestations),
		Deposits:           CopySlice(body.Deposits),
		VoluntaryExits:     CopySlice(body.VoluntaryExits),
		ExecutionPayload:   body.ExecutionPayload.Copy(),
		BlobKzgCommitments: CopyBlobKZGs(body.BlobKzgCommitments),
		ExecutionRequests:  CopyExecutionRequests(body.ExecutionRequests),
	}
}

// Copy --
func (sigBlock *SignedBeaconBlockBadger) Copy() *SignedBeaconBlockBadger {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockBadger{
		Block:     sigBlock.Block.Copy(),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// Copy --
func (block *BeaconBlockBadger) Copy() *BeaconBlockBadger {
	if block == nil {
		return nil
	}
	return &BeaconBlockBadger{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          block.Body.Copy(),
	}
}

// Copy --
func (body *BeaconBlockBodyBadger) Copy() *BeaconBlockBodyBadger {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyBadger{
		RandaoReveal:       bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:           body.Eth1Data.Copy(),
		Graffiti:           bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:  CopySlice(body.ProposerSlashings),
		AttesterSlashings:  CopySlice(body.AttesterSlashings),
		Attestations:       CopySlice(body.Attestations),
		Deposits:           CopySlice(body.Deposits),
		VoluntaryExits:     CopySlice(body.VoluntaryExits),
		ExecutionPayload:   body.ExecutionPayload.Copy(),
		BlobKzgCommitments: CopyBlobKZGs(body.BlobKzgCommitments),
		ExecutionRequests:  CopyExecutionRequests(body.ExecutionRequests),
	}
}

// Copy --
func (data *Eth1Data) Copy() *Eth1Data {
	if data == nil {
		return nil
	}
	return &Eth1Data{
		DepositRoot:  bytesutil.SafeCopyBytes(data.DepositRoot),
		DepositCount: data.DepositCount,
		BlockHash:    bytesutil.SafeCopyBytes(data.BlockHash),
	}
}

// Copy --
func (slashing *ProposerSlashing) Copy() *ProposerSlashing {
	if slashing == nil {
		return nil
	}
	return &ProposerSlashing{
		Header_1: slashing.Header_1.Copy(),
		Header_2: slashing.Header_2.Copy(),
	}
}

// Copy --
func (header *SignedBeaconBlockHeader) Copy() *SignedBeaconBlockHeader {
	if header == nil {
		return nil
	}
	return &SignedBeaconBlockHeader{
		Header:    header.Header.Copy(),
		Signature: bytesutil.SafeCopyBytes(header.Signature),
	}
}

// Copy --
func (header *BeaconBlockHeader) Copy() *BeaconBlockHeader {
	if header == nil {
		return nil
	}
	parentRoot := bytesutil.SafeCopyBytes(header.ParentRoot)
	stateRoot := bytesutil.SafeCopyBytes(header.StateRoot)
	bodyRoot := bytesutil.SafeCopyBytes(header.BodyRoot)
	return &BeaconBlockHeader{
		Slot:          header.Slot,
		ProposerIndex: header.ProposerIndex,
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		BodyRoot:      bodyRoot,
	}
}

// Copy --
func (deposit *Deposit) Copy() *Deposit {
	if deposit == nil {
		return nil
	}
	return &Deposit{
		Proof: bytesutil.SafeCopy2dBytes(deposit.Proof),
		Data:  deposit.Data.Copy(),
	}
}

// Copy --
func (depData *Deposit_Data) Copy() *Deposit_Data {
	if depData == nil {
		return nil
	}
	return &Deposit_Data{
		PublicKey:             bytesutil.SafeCopyBytes(depData.PublicKey),
		WithdrawalCredentials: bytesutil.SafeCopyBytes(depData.WithdrawalCredentials),
		Amount:                depData.Amount,
		Signature:             bytesutil.SafeCopyBytes(depData.Signature),
	}
}

// Copy --
func (exit *SignedVoluntaryExit) Copy() *SignedVoluntaryExit {
	if exit == nil {
		return nil
	}
	return &SignedVoluntaryExit{
		Exit:      exit.Exit.Copy(),
		Signature: bytesutil.SafeCopyBytes(exit.Signature),
	}
}

// Copy --
func (exit *VoluntaryExit) Copy() *VoluntaryExit {
	if exit == nil {
		return nil
	}
	return &VoluntaryExit{
		Epoch:          exit.Epoch,
		ValidatorIndex: exit.ValidatorIndex,
	}
}

// Copy --
func (summary *HistoricalSummary) Copy() *HistoricalSummary {
	if summary == nil {
		return nil
	}
	return &HistoricalSummary{
		BlockSummaryRoot: bytesutil.SafeCopyBytes(summary.BlockSummaryRoot),
		StateSummaryRoot: bytesutil.SafeCopyBytes(summary.StateSummaryRoot),
	}
}
