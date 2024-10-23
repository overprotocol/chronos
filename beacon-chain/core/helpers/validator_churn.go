package helpers

import (
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

// BalanceChurnLimit for the current active balance, in gwei.
// New in Electra EIP-7251: https://eips.ethereum.org/EIPS/eip-7251
//
// Spec definition:
//
//	def get_balance_churn_limit(state: BeaconState) -> Gwei:
//	    """
//	    Return the churn limit for the current epoch.
//	    """
//	    churn = max(
//	        MIN_PER_EPOCH_CHURN_LIMIT_ELECTRA,
//	        get_total_active_balance(state) // CHURN_LIMIT_QUOTIENT
//	    )
//	    return churn - churn % EFFECTIVE_BALANCE_INCREMENT
func BalanceChurnLimit(activeBalance primitives.Gwei) primitives.Gwei {
	cfg := params.BeaconConfig()
	churn := max(
		cfg.MinPerEpochChurnLimitAlpaca,
		uint64(activeBalance)/cfg.ChurnLimitQuotient,
	)
	return primitives.Gwei(churn - churn%cfg.EffectiveBalanceIncrement)
}

// ActivationBalanceChurnLimit for the current active balance, in gwei.
//
// Spec definition:
//
//	def get_activation_balance_churn_limit(state: BeaconState) -> Gwei:
//	   return max(MIN_PER_EPOCH_ACTIVATION_BALANCE_CHURN_LIMIT, get_balance_churn_limit(state))
func ActivationBalanceChurnLimit(activeBalance primitives.Gwei) primitives.Gwei {
	return max(primitives.Gwei(params.BeaconConfig().MinPerEpochActivationBalanceChurnLimit), BalanceChurnLimit(activeBalance))
}

// ExitBalanceChurnLimit for the current active balance, in gwei.
//
// Spec definition:
//
//		def get_exit_balance_churn_limit(state: BeaconState) -> Gwei:
//	   return get_balance_churn_limit(state)
func ExitBalanceChurnLimit(activeBalance primitives.Gwei) primitives.Gwei {
	return BalanceChurnLimit(activeBalance)
}
