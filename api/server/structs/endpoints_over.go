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

type GetDepositPreEstimationResponse struct {
	Data                *DepositPreEstimationContainer `json:"data"`
	ExecutionOptimistic bool                           `json:"execution_optimistic"`
	Finalized           bool                           `json:"finalized"`
}

type DepositPreEstimationContainer struct {
	ExpectedEpoch           uint64 `json:"expected_epoch"`
	ExpectedActivationEpoch uint64 `json:"expected_activation_epoch"` // For initial deposits
}

type GetDepositEstimationResponse struct {
	Data                *DepositEstimationContainer `json:"data"`
	ExecutionOptimistic bool                        `json:"execution_optimistic"`
	Finalized           bool                        `json:"finalized"`
}

type DepositEstimationContainer struct {
	Pubkey string `json:"pubkey"`

	// if the validator is found on registry, include the validator in the response.
	Validator       *Validator                           `json:"validator,omitempty"`
	PendingDeposits []*PendingDepositEstimationContainer `json:"pending_deposits"`

	// if the validator is found on registry, but its activation epoch is not yet decided,
	// include estimation for activation.
	ExpectedActivationEpoch uint64 `json:"expected_activation_epoch,omitempty"`
}

type PendingDepositEstimationContainer struct {
	Type string                    `json:"type"` // "initial" or "top-up"
	Data *PendingDepositEstimation `json:"data"`
}

type PendingDepositEstimation struct {
	Amount        uint64 `json:"amount"`
	Slot          uint64 `json:"slot"`
	ExpectedEpoch uint64 `json:"expected_epoch"`

	// if it is "initial" deposit, include estimation for activation.
	ExpectedActivationEpoch uint64 `json:"expected_activation_epoch,omitempty"`
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

type GetExitQueueEpochResponse struct {
	Data                *ExitQueueEpochContainer `json:"data"`
	ExecutionOptimistic bool                     `json:"execution_optimistic"`
	Finalized           bool                     `json:"finalized"`
}

type ExitQueueEpochContainer struct {
	ExitQueueEpoch uint64 `json:"exit_queue_epoch"`
}
