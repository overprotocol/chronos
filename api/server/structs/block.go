package structs

import "encoding/json"

// MessageJsoner describes a signed consensus type wrapper that can return the `.Message` field in a json envelope
// encoded as a []byte, for use as a json.RawMessage value when encoding the outer envelope.
type MessageJsoner interface {
	MessageRawJson() ([]byte, error)
}

// SignedMessageJsoner embeds MessageJsoner and adds a method to also retrieve the Signature field as a string.
type SignedMessageJsoner interface {
	MessageJsoner
	SigString() string
}

type SignedBeaconBlock struct {
	Message   *BeaconBlock `json:"message"`
	Signature string       `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlock{}

func (s *SignedBeaconBlock) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlock) SigString() string {
	return s.Signature
}

type BeaconBlock struct {
	Slot          string           `json:"slot"`
	ProposerIndex string           `json:"proposer_index"`
	ParentRoot    string           `json:"parent_root"`
	StateRoot     string           `json:"state_root"`
	Body          *BeaconBlockBody `json:"body"`
}

type BeaconBlockBody struct {
	RandaoReveal      string                 `json:"randao_reveal"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          string                 `json:"graffiti"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings"`
	Attestations      []*Attestation         `json:"attestations"`
	Deposits          []*Deposit             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits"`
}

type SignedBeaconBlockAltair struct {
	Message   *BeaconBlockAltair `json:"message"`
	Signature string             `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlockAltair{}

func (s *SignedBeaconBlockAltair) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlockAltair) SigString() string {
	return s.Signature
}

type BeaconBlockAltair struct {
	Slot          string                 `json:"slot"`
	ProposerIndex string                 `json:"proposer_index"`
	ParentRoot    string                 `json:"parent_root"`
	StateRoot     string                 `json:"state_root"`
	Body          *BeaconBlockBodyAltair `json:"body"`
}

type BeaconBlockBodyAltair struct {
	RandaoReveal      string                 `json:"randao_reveal"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          string                 `json:"graffiti"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings"`
	Attestations      []*Attestation         `json:"attestations"`
	Deposits          []*Deposit             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits"`
}

type SignedBeaconBlockBellatrix struct {
	Message   *BeaconBlockBellatrix `json:"message"`
	Signature string                `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlockBellatrix{}

func (s *SignedBeaconBlockBellatrix) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlockBellatrix) SigString() string {
	return s.Signature
}

type BeaconBlockBellatrix struct {
	Slot          string                    `json:"slot"`
	ProposerIndex string                    `json:"proposer_index"`
	ParentRoot    string                    `json:"parent_root"`
	StateRoot     string                    `json:"state_root"`
	Body          *BeaconBlockBodyBellatrix `json:"body"`
}

type BeaconBlockBodyBellatrix struct {
	RandaoReveal      string                 `json:"randao_reveal"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          string                 `json:"graffiti"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings"`
	Attestations      []*Attestation         `json:"attestations"`
	Deposits          []*Deposit             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits"`
	ExecutionPayload  *ExecutionPayload      `json:"execution_payload"`
}

type SignedBlindedBeaconBlockBellatrix struct {
	Message   *BlindedBeaconBlockBellatrix `json:"message"`
	Signature string                       `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBlindedBeaconBlockBellatrix{}

func (s *SignedBlindedBeaconBlockBellatrix) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBlindedBeaconBlockBellatrix) SigString() string {
	return s.Signature
}

type BlindedBeaconBlockBellatrix struct {
	Slot          string                           `json:"slot"`
	ProposerIndex string                           `json:"proposer_index"`
	ParentRoot    string                           `json:"parent_root"`
	StateRoot     string                           `json:"state_root"`
	Body          *BlindedBeaconBlockBodyBellatrix `json:"body"`
}

type BlindedBeaconBlockBodyBellatrix struct {
	RandaoReveal           string                  `json:"randao_reveal"`
	Eth1Data               *Eth1Data               `json:"eth1_data"`
	Graffiti               string                  `json:"graffiti"`
	ProposerSlashings      []*ProposerSlashing     `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashing     `json:"attester_slashings"`
	Attestations           []*Attestation          `json:"attestations"`
	Deposits               []*Deposit              `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExit  `json:"voluntary_exits"`
	ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header"`
}

type SignedBeaconBlockCapella struct {
	Message   *BeaconBlockCapella `json:"message"`
	Signature string              `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlockCapella{}

func (s *SignedBeaconBlockCapella) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlockCapella) SigString() string {
	return s.Signature
}

type BeaconBlockCapella struct {
	Slot          string                  `json:"slot"`
	ProposerIndex string                  `json:"proposer_index"`
	ParentRoot    string                  `json:"parent_root"`
	StateRoot     string                  `json:"state_root"`
	Body          *BeaconBlockBodyCapella `json:"body"`
}

type BeaconBlockBodyCapella struct {
	RandaoReveal      string                   `json:"randao_reveal"`
	Eth1Data          *Eth1Data                `json:"eth1_data"`
	Graffiti          string                   `json:"graffiti"`
	ProposerSlashings []*ProposerSlashing      `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing      `json:"attester_slashings"`
	Attestations      []*Attestation           `json:"attestations"`
	Deposits          []*Deposit               `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit   `json:"voluntary_exits"`
	ExecutionPayload  *ExecutionPayloadCapella `json:"execution_payload"`
}

type SignedBlindedBeaconBlockCapella struct {
	Message   *BlindedBeaconBlockCapella `json:"message"`
	Signature string                     `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBlindedBeaconBlockCapella{}

func (s *SignedBlindedBeaconBlockCapella) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBlindedBeaconBlockCapella) SigString() string {
	return s.Signature
}

type BlindedBeaconBlockCapella struct {
	Slot          string                         `json:"slot"`
	ProposerIndex string                         `json:"proposer_index"`
	ParentRoot    string                         `json:"parent_root"`
	StateRoot     string                         `json:"state_root"`
	Body          *BlindedBeaconBlockBodyCapella `json:"body"`
}

type BlindedBeaconBlockBodyCapella struct {
	RandaoReveal           string                         `json:"randao_reveal"`
	Eth1Data               *Eth1Data                      `json:"eth1_data"`
	Graffiti               string                         `json:"graffiti"`
	ProposerSlashings      []*ProposerSlashing            `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashing            `json:"attester_slashings"`
	Attestations           []*Attestation                 `json:"attestations"`
	Deposits               []*Deposit                     `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExit         `json:"voluntary_exits"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderCapella `json:"execution_payload_header"`
}

type SignedBeaconBlockContentsDeneb struct {
	SignedBlock *SignedBeaconBlockDeneb `json:"signed_block"`
	KzgProofs   []string                `json:"kzg_proofs"`
	Blobs       []string                `json:"blobs"`
}

type BeaconBlockContentsDeneb struct {
	Block     *BeaconBlockDeneb `json:"block"`
	KzgProofs []string          `json:"kzg_proofs"`
	Blobs     []string          `json:"blobs"`
}

type SignedBeaconBlockDeneb struct {
	Message   *BeaconBlockDeneb `json:"message"`
	Signature string            `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlockDeneb{}

func (s *SignedBeaconBlockDeneb) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlockDeneb) SigString() string {
	return s.Signature
}

type BeaconBlockDeneb struct {
	Slot          string                `json:"slot"`
	ProposerIndex string                `json:"proposer_index"`
	ParentRoot    string                `json:"parent_root"`
	StateRoot     string                `json:"state_root"`
	Body          *BeaconBlockBodyDeneb `json:"body"`
}

type BeaconBlockBodyDeneb struct {
	RandaoReveal       string                 `json:"randao_reveal"`
	Eth1Data           *Eth1Data              `json:"eth1_data"`
	Graffiti           string                 `json:"graffiti"`
	ProposerSlashings  []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings  []*AttesterSlashing    `json:"attester_slashings"`
	Attestations       []*Attestation         `json:"attestations"`
	Deposits           []*Deposit             `json:"deposits"`
	VoluntaryExits     []*SignedVoluntaryExit `json:"voluntary_exits"`
	ExecutionPayload   *ExecutionPayloadDeneb `json:"execution_payload"`
	BlobKzgCommitments []string               `json:"blob_kzg_commitments"`
}

type BlindedBeaconBlockDeneb struct {
	Slot          string                       `json:"slot"`
	ProposerIndex string                       `json:"proposer_index"`
	ParentRoot    string                       `json:"parent_root"`
	StateRoot     string                       `json:"state_root"`
	Body          *BlindedBeaconBlockBodyDeneb `json:"body"`
}

type SignedBlindedBeaconBlockDeneb struct {
	Message   *BlindedBeaconBlockDeneb `json:"message"`
	Signature string                   `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBlindedBeaconBlockDeneb{}

func (s *SignedBlindedBeaconBlockDeneb) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBlindedBeaconBlockDeneb) SigString() string {
	return s.Signature
}

type BlindedBeaconBlockBodyDeneb struct {
	RandaoReveal           string                       `json:"randao_reveal"`
	Eth1Data               *Eth1Data                    `json:"eth1_data"`
	Graffiti               string                       `json:"graffiti"`
	ProposerSlashings      []*ProposerSlashing          `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashing          `json:"attester_slashings"`
	Attestations           []*Attestation               `json:"attestations"`
	Deposits               []*Deposit                   `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExit       `json:"voluntary_exits"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderDeneb `json:"execution_payload_header"`
	BlobKzgCommitments     []string                     `json:"blob_kzg_commitments"`
}

type SignedBeaconBlockContentsElectra struct {
	SignedBlock *SignedBeaconBlockElectra `json:"signed_block"`
	KzgProofs   []string                  `json:"kzg_proofs"`
	Blobs       []string                  `json:"blobs"`
}

type BeaconBlockContentsElectra struct {
	Block     *BeaconBlockElectra `json:"block"`
	KzgProofs []string            `json:"kzg_proofs"`
	Blobs     []string            `json:"blobs"`
}

type SignedBeaconBlockElectra struct {
	Message   *BeaconBlockElectra `json:"message"`
	Signature string              `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlockElectra{}

func (s *SignedBeaconBlockElectra) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlockElectra) SigString() string {
	return s.Signature
}

type BeaconBlockElectra struct {
	Slot          string                  `json:"slot"`
	ProposerIndex string                  `json:"proposer_index"`
	ParentRoot    string                  `json:"parent_root"`
	StateRoot     string                  `json:"state_root"`
	Body          *BeaconBlockBodyElectra `json:"body"`
}

type BeaconBlockBodyElectra struct {
	RandaoReveal       string                     `json:"randao_reveal"`
	Eth1Data           *Eth1Data                  `json:"eth1_data"`
	Graffiti           string                     `json:"graffiti"`
	ProposerSlashings  []*ProposerSlashing        `json:"proposer_slashings"`
	AttesterSlashings  []*AttesterSlashingElectra `json:"attester_slashings"`
	Attestations       []*AttestationElectra      `json:"attestations"`
	Deposits           []*Deposit                 `json:"deposits"`
	VoluntaryExits     []*SignedVoluntaryExit     `json:"voluntary_exits"`
	ExecutionPayload   *ExecutionPayloadDeneb     `json:"execution_payload"`
	BlobKzgCommitments []string                   `json:"blob_kzg_commitments"`
	ExecutionRequests  *ExecutionRequests         `json:"execution_requests"`
}

type BlindedBeaconBlockElectra struct {
	Slot          string                         `json:"slot"`
	ProposerIndex string                         `json:"proposer_index"`
	ParentRoot    string                         `json:"parent_root"`
	StateRoot     string                         `json:"state_root"`
	Body          *BlindedBeaconBlockBodyElectra `json:"body"`
}

type SignedBlindedBeaconBlockElectra struct {
	Message   *BlindedBeaconBlockElectra `json:"message"`
	Signature string                     `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBlindedBeaconBlockElectra{}

func (s *SignedBlindedBeaconBlockElectra) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBlindedBeaconBlockElectra) SigString() string {
	return s.Signature
}

type BlindedBeaconBlockBodyElectra struct {
	RandaoReveal           string                       `json:"randao_reveal"`
	Eth1Data               *Eth1Data                    `json:"eth1_data"`
	Graffiti               string                       `json:"graffiti"`
	ProposerSlashings      []*ProposerSlashing          `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashingElectra   `json:"attester_slashings"`
	Attestations           []*AttestationElectra        `json:"attestations"`
	Deposits               []*Deposit                   `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExit       `json:"voluntary_exits"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderDeneb `json:"execution_payload_header"`
	BlobKzgCommitments     []string                     `json:"blob_kzg_commitments"`
	ExecutionRequests      *ExecutionRequests           `json:"execution_requests"`
}

type SignedBeaconBlockContentsBadger struct {
	SignedBlock *SignedBeaconBlockBadger `json:"signed_block"`
	KzgProofs   []string                 `json:"kzg_proofs"`
	Blobs       []string                 `json:"blobs"`
}

type BeaconBlockContentsBadger struct {
	Block     *BeaconBlockBadger `json:"block"`
	KzgProofs []string           `json:"kzg_proofs"`
	Blobs     []string           `json:"blobs"`
}

type SignedBeaconBlockBadger struct {
	Message   *BeaconBlockBadger `json:"message"`
	Signature string             `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlockBadger{}

func (s *SignedBeaconBlockBadger) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlockBadger) SigString() string {
	return s.Signature
}

type BeaconBlockBadger struct {
	Slot          string                 `json:"slot"`
	ProposerIndex string                 `json:"proposer_index"`
	ParentRoot    string                 `json:"parent_root"`
	StateRoot     string                 `json:"state_root"`
	Body          *BeaconBlockBodyBadger `json:"body"`
}

type BeaconBlockBodyBadger struct {
	RandaoReveal       string                     `json:"randao_reveal"`
	Eth1Data           *Eth1Data                  `json:"eth1_data"`
	Graffiti           string                     `json:"graffiti"`
	ProposerSlashings  []*ProposerSlashing        `json:"proposer_slashings"`
	AttesterSlashings  []*AttesterSlashingElectra `json:"attester_slashings"`
	Attestations       []*AttestationElectra      `json:"attestations"`
	Deposits           []*Deposit                 `json:"deposits"`
	VoluntaryExits     []*SignedVoluntaryExit     `json:"voluntary_exits"`
	ExecutionPayload   *ExecutionPayloadDeneb     `json:"execution_payload"`
	BlobKzgCommitments []string                   `json:"blob_kzg_commitments"`
	ExecutionRequests  *ExecutionRequests         `json:"execution_requests"`
}

type BlindedBeaconBlockBadger struct {
	Slot          string                        `json:"slot"`
	ProposerIndex string                        `json:"proposer_index"`
	ParentRoot    string                        `json:"parent_root"`
	StateRoot     string                        `json:"state_root"`
	Body          *BlindedBeaconBlockBodyBadger `json:"body"`
}

type SignedBlindedBeaconBlockBadger struct {
	Message   *BlindedBeaconBlockBadger `json:"message"`
	Signature string                    `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBlindedBeaconBlockBadger{}

func (s *SignedBlindedBeaconBlockBadger) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBlindedBeaconBlockBadger) SigString() string {
	return s.Signature
}

type BlindedBeaconBlockBodyBadger struct {
	RandaoReveal           string                       `json:"randao_reveal"`
	Eth1Data               *Eth1Data                    `json:"eth1_data"`
	Graffiti               string                       `json:"graffiti"`
	ProposerSlashings      []*ProposerSlashing          `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashingElectra   `json:"attester_slashings"`
	Attestations           []*AttestationElectra        `json:"attestations"`
	Deposits               []*Deposit                   `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExit       `json:"voluntary_exits"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderDeneb `json:"execution_payload_header"`
	BlobKzgCommitments     []string                     `json:"blob_kzg_commitments"`
	ExecutionRequests      *ExecutionRequests           `json:"execution_requests"`
}

type SignedBeaconBlockHeaderContainer struct {
	Header    *SignedBeaconBlockHeader `json:"header"`
	Root      string                   `json:"root"`
	Canonical bool                     `json:"canonical"`
}

type SignedBeaconBlockHeader struct {
	Message   *BeaconBlockHeader `json:"message"`
	Signature string             `json:"signature"`
}

type BeaconBlockHeader struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

type ExecutionPayload struct {
	ParentHash    string   `json:"parent_hash"`
	FeeRecipient  string   `json:"fee_recipient"`
	StateRoot     string   `json:"state_root"`
	ReceiptsRoot  string   `json:"receipts_root"`
	LogsBloom     string   `json:"logs_bloom"`
	PrevRandao    string   `json:"prev_randao"`
	BlockNumber   string   `json:"block_number"`
	GasLimit      string   `json:"gas_limit"`
	GasUsed       string   `json:"gas_used"`
	Timestamp     string   `json:"timestamp"`
	ExtraData     string   `json:"extra_data"`
	BaseFeePerGas string   `json:"base_fee_per_gas"`
	BlockHash     string   `json:"block_hash"`
	Transactions  []string `json:"transactions"`
}

type ExecutionPayloadHeader struct {
	ParentHash       string `json:"parent_hash"`
	FeeRecipient     string `json:"fee_recipient"`
	StateRoot        string `json:"state_root"`
	ReceiptsRoot     string `json:"receipts_root"`
	LogsBloom        string `json:"logs_bloom"`
	PrevRandao       string `json:"prev_randao"`
	BlockNumber      string `json:"block_number"`
	GasLimit         string `json:"gas_limit"`
	GasUsed          string `json:"gas_used"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extra_data"`
	BaseFeePerGas    string `json:"base_fee_per_gas"`
	BlockHash        string `json:"block_hash"`
	TransactionsRoot string `json:"transactions_root"`
}

type ExecutionPayloadCapella struct {
	ParentHash    string        `json:"parent_hash"`
	FeeRecipient  string        `json:"fee_recipient"`
	StateRoot     string        `json:"state_root"`
	ReceiptsRoot  string        `json:"receipts_root"`
	LogsBloom     string        `json:"logs_bloom"`
	PrevRandao    string        `json:"prev_randao"`
	BlockNumber   string        `json:"block_number"`
	GasLimit      string        `json:"gas_limit"`
	GasUsed       string        `json:"gas_used"`
	Timestamp     string        `json:"timestamp"`
	ExtraData     string        `json:"extra_data"`
	BaseFeePerGas string        `json:"base_fee_per_gas"`
	BlockHash     string        `json:"block_hash"`
	Transactions  []string      `json:"transactions"`
	Withdrawals   []*Withdrawal `json:"withdrawals"`
}

type ExecutionPayloadHeaderCapella struct {
	ParentHash       string `json:"parent_hash"`
	FeeRecipient     string `json:"fee_recipient"`
	StateRoot        string `json:"state_root"`
	ReceiptsRoot     string `json:"receipts_root"`
	LogsBloom        string `json:"logs_bloom"`
	PrevRandao       string `json:"prev_randao"`
	BlockNumber      string `json:"block_number"`
	GasLimit         string `json:"gas_limit"`
	GasUsed          string `json:"gas_used"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extra_data"`
	BaseFeePerGas    string `json:"base_fee_per_gas"`
	BlockHash        string `json:"block_hash"`
	TransactionsRoot string `json:"transactions_root"`
	WithdrawalsRoot  string `json:"withdrawals_root"`
}

type ExecutionPayloadDeneb struct {
	ParentHash    string        `json:"parent_hash"`
	FeeRecipient  string        `json:"fee_recipient"`
	StateRoot     string        `json:"state_root"`
	ReceiptsRoot  string        `json:"receipts_root"`
	LogsBloom     string        `json:"logs_bloom"`
	PrevRandao    string        `json:"prev_randao"`
	BlockNumber   string        `json:"block_number"`
	GasLimit      string        `json:"gas_limit"`
	GasUsed       string        `json:"gas_used"`
	Timestamp     string        `json:"timestamp"`
	ExtraData     string        `json:"extra_data"`
	BaseFeePerGas string        `json:"base_fee_per_gas"`
	BlockHash     string        `json:"block_hash"`
	Transactions  []string      `json:"transactions"`
	Withdrawals   []*Withdrawal `json:"withdrawals"`
	BlobGasUsed   string        `json:"blob_gas_used"`
	ExcessBlobGas string        `json:"excess_blob_gas"`
}

type ExecutionPayloadHeaderDeneb struct {
	ParentHash       string `json:"parent_hash"`
	FeeRecipient     string `json:"fee_recipient"`
	StateRoot        string `json:"state_root"`
	ReceiptsRoot     string `json:"receipts_root"`
	LogsBloom        string `json:"logs_bloom"`
	PrevRandao       string `json:"prev_randao"`
	BlockNumber      string `json:"block_number"`
	GasLimit         string `json:"gas_limit"`
	GasUsed          string `json:"gas_used"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extra_data"`
	BaseFeePerGas    string `json:"base_fee_per_gas"`
	BlockHash        string `json:"block_hash"`
	TransactionsRoot string `json:"transactions_root"`
	WithdrawalsRoot  string `json:"withdrawals_root"`
	BlobGasUsed      string `json:"blob_gas_used"`
	ExcessBlobGas    string `json:"excess_blob_gas"`
}

type ExecutionRequests struct {
	Deposits    []*DepositRequest    `json:"deposits"`
	Withdrawals []*WithdrawalRequest `json:"withdrawals"`
}
