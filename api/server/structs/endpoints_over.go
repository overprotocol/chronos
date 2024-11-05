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

type GetWithdrawalEstimationResponse struct {
	Data                *WithdrawalEstimationContainer `json:"data"`
	ExecutionOptimistic bool                           `json:"execution_optimistic"`
	Finalized           bool                           `json:"finalized"`
}

type WithdrawalEstimationContainer struct {
	Pubkey                    string                               `json:"pubkey"`
	PendingPartialWithdrawals []*PendingPartialWithdrawalContainer `json:"pending_partial_withdrawals"`
}

type PendingPartialWithdrawalContainer struct {
	Amount        uint64 `json:"amount"`
	ExpectedEpoch uint64 `json:"expected_epoch"`
}
