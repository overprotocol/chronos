package structs

type EstimatedActivationResponse struct {
	WaitingEpoch  uint64 `json:"waiting_epoch"`
	EligibleEpoch uint64 `json:"eligible_epoch"`
	Status        uint64 `json:"status"`
}

type EpochReward struct {
	Reward string `json:"reward"`
}

type GetReservesResponse struct {
	Data                *Reserves `json:"data"`
	ExecutionOptimistic bool      `json:"execution_optimistic"`
	Finalized           bool      `json:"finalized"`
}

type Reserves struct {
	RewardAdjustmentFactor string `json:"reward_adjustment_factor"`
	Reserves               string `json:"reserves"`
}
