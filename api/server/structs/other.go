package structs

type Validator struct {
	Pubkey                     string `json:"pubkey"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           string `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch string `json:"activation_eligibility_epoch"`
	ActivationEpoch            string `json:"activation_epoch"`
	ExitEpoch                  string `json:"exit_epoch"`
	PrincipalBalance           string `json:"principal_balance"`
}

type PendingAttestation struct {
	AggregationBits string           `json:"aggregation_bits"`
	Data            *AttestationData `json:"data"`
	InclusionDelay  string           `json:"inclusion_delay"`
	ProposerIndex   string           `json:"proposer_index"`
}

type HistoricalSummary struct {
	BlockSummaryRoot string `json:"block_summary_root"`
	StateSummaryRoot string `json:"state_summary_root"`
}

type Attestation struct {
	AggregationBits string           `json:"aggregation_bits"`
	Data            *AttestationData `json:"data"`
	Signature       string           `json:"signature"`
}

type AttestationElectra struct {
	AggregationBits string           `json:"aggregation_bits"`
	Data            *AttestationData `json:"data"`
	Signature       string           `json:"signature"`
	CommitteeBits   string           `json:"committee_bits"`
}

type AttestationData struct {
	Slot            string      `json:"slot"`
	CommitteeIndex  string      `json:"index"`
	BeaconBlockRoot string      `json:"beacon_block_root"`
	Source          *Checkpoint `json:"source"`
	Target          *Checkpoint `json:"target"`
}

type Checkpoint struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root"`
}

type Committee struct {
	Index      string   `json:"index"`
	Slot       string   `json:"slot"`
	Validators []string `json:"validators"`
}

type SignedAggregateAttestationAndProof struct {
	Message   *AggregateAttestationAndProof `json:"message"`
	Signature string                        `json:"signature"`
}

type AggregateAttestationAndProof struct {
	AggregatorIndex string       `json:"aggregator_index"`
	Aggregate       *Attestation `json:"aggregate"`
	SelectionProof  string       `json:"selection_proof"`
}

type SignedAggregateAttestationAndProofElectra struct {
	Message   *AggregateAttestationAndProofElectra `json:"message"`
	Signature string                               `json:"signature"`
}

type AggregateAttestationAndProofElectra struct {
	AggregatorIndex string              `json:"aggregator_index"`
	Aggregate       *AttestationElectra `json:"aggregate"`
	SelectionProof  string              `json:"selection_proof"`
}

type BeaconCommitteeSubscription struct {
	ValidatorIndex   string `json:"validator_index"`
	CommitteeIndex   string `json:"committee_index"`
	CommitteesAtSlot string `json:"committees_at_slot"`
	Slot             string `json:"slot"`
	IsAggregator     bool   `json:"is_aggregator"`
}

type ValidatorRegistration struct {
	FeeRecipient string `json:"fee_recipient"`
	GasLimit     string `json:"gas_limit"`
	Timestamp    string `json:"timestamp"`
	Pubkey       string `json:"pubkey"`
}

type SignedValidatorRegistration struct {
	Message   *ValidatorRegistration `json:"message"`
	Signature string                 `json:"signature"`
}

type FeeRecipient struct {
	ValidatorIndex string `json:"validator_index"`
	FeeRecipient   string `json:"fee_recipient"`
}

type SignedVoluntaryExit struct {
	Message   *VoluntaryExit `json:"message"`
	Signature string         `json:"signature"`
}

type VoluntaryExit struct {
	Epoch          string `json:"epoch"`
	ValidatorIndex string `json:"validator_index"`
}

type Fork struct {
	PreviousVersion string `json:"previous_version"`
	CurrentVersion  string `json:"current_version"`
	Epoch           string `json:"epoch"`
}

// SyncDetails contains information about node sync status.
type SyncDetails struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
	IsOptimistic bool   `json:"is_optimistic"`
	ElOffline    bool   `json:"el_offline"`
}

// SyncDetailsContainer is a wrapper for Data.
type SyncDetailsContainer struct {
	Data *SyncDetails `json:"data"`
}

type Eth1Data struct {
	DepositRoot  string `json:"deposit_root"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash"`
}

type ProposerSlashing struct {
	SignedHeader1 *SignedBeaconBlockHeader `json:"signed_header_1"`
	SignedHeader2 *SignedBeaconBlockHeader `json:"signed_header_2"`
}

type AttesterSlashing struct {
	Attestation1 *IndexedAttestation `json:"attestation_1"`
	Attestation2 *IndexedAttestation `json:"attestation_2"`
}

type AttesterSlashingElectra struct {
	Attestation1 *IndexedAttestationElectra `json:"attestation_1"`
	Attestation2 *IndexedAttestationElectra `json:"attestation_2"`
}

type Deposit struct {
	Proof []string     `json:"proof"`
	Data  *DepositData `json:"data"`
}

type DepositData struct {
	Pubkey                string `json:"pubkey"`
	WithdrawalCredentials string `json:"withdrawal_credentials"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature"`
}

type IndexedAttestation struct {
	AttestingIndices []string         `json:"attesting_indices"`
	Data             *AttestationData `json:"data"`
	Signature        string           `json:"signature"`
}

type IndexedAttestationElectra struct {
	AttestingIndices []string         `json:"attesting_indices"`
	Data             *AttestationData `json:"data"`
	Signature        string           `json:"signature"`
}

type Withdrawal struct {
	WithdrawalIndex  string `json:"index"`
	ValidatorIndex   string `json:"validator_index"`
	ExecutionAddress string `json:"address"`
	Amount           string `json:"amount"`
}

type DepositRequest struct {
	Pubkey                string `json:"pubkey"`
	WithdrawalCredentials string `json:"withdrawal_credentials"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature"`
	Index                 string `json:"index"`
}

type WithdrawalRequest struct {
	SourceAddress   string `json:"source_address"`
	ValidatorPubkey string `json:"validator_pubkey"`
	Amount          string `json:"amount"`
}

type PendingDeposit struct {
	Pubkey                string `json:"pubkey"`
	WithdrawalCredentials string `json:"withdrawal_credentials"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature"`
	Slot                  string `json:"slot"`
}

type PendingPartialWithdrawal struct {
	Index             string `json:"index"`
	Amount            string `json:"amount"`
	WithdrawableEpoch string `json:"withdrawable_epoch"`
}
