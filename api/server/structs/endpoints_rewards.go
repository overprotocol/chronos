package structs

type BlockRewardsResponse struct {
	Data                *BlockRewards `json:"data"`
	ExecutionOptimistic bool          `json:"execution_optimistic"`
	Finalized           bool          `json:"finalized"`
}

type BlockRewards struct {
	ProposerIndex     string `json:"proposer_index"`
	Total             string `json:"total"`
	Attestations      string `json:"attestations"`
	ProposerSlashings string `json:"proposer_slashings"`
	AttesterSlashings string `json:"attester_slashings"`
}

type AttestationRewardsResponse struct {
	Data                AttestationRewards `json:"data"`
	ExecutionOptimistic bool               `json:"execution_optimistic"`
	Finalized           bool               `json:"finalized"`
}

type AttestationRewards struct {
	IdealRewards []IdealAttestationReward `json:"ideal_rewards"`
	TotalRewards []TotalAttestationReward `json:"total_rewards"`
}

type IdealAttestationReward struct {
	EffectiveBalance string `json:"effective_balance"`
	Head             string `json:"head"`
	Target           string `json:"target"`
	Source           string `json:"source"`
}

type TotalAttestationReward struct {
	ValidatorIndex string `json:"validator_index"`
	Head           string `json:"head"`
	Target         string `json:"target"`
	Source         string `json:"source"`
}
