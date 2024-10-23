package helpers_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
)

func TestBalanceChurnLimit(t *testing.T) {
	tests := []struct {
		name          string
		activeBalance primitives.Gwei
		expected      primitives.Gwei
	}{
		{
			name:          "less than MIN_PER_EPOCH_CHURN_LIMIT_ALPACA",
			activeBalance: 111,
			expected:      primitives.Gwei(params.BeaconConfig().MinPerEpochChurnLimitAlpaca),
		},
		{
			name:          "modulo EFFECTIVE_BALANCE_INCREMENT",
			activeBalance: primitives.Gwei(111 + params.BeaconConfig().MinPerEpochChurnLimitAlpaca*params.BeaconConfig().ChurnLimitQuotient),
			expected:      primitives.Gwei(params.BeaconConfig().MinPerEpochChurnLimitAlpaca),
		},
		{
			name:          "more than MIN_PER_EPOCH_CHURN_LIMIT_ELECTRA",
			activeBalance: primitives.Gwei(2000 * params.BeaconConfig().EffectiveBalanceIncrement * params.BeaconConfig().ChurnLimitQuotient),
			expected:      primitives.Gwei(2000 * params.BeaconConfig().EffectiveBalanceIncrement),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, helpers.BalanceChurnLimit(tt.activeBalance))
		})
	}
}

func TestActivationBalanceChurnLimit(t *testing.T) {
	tests := []struct {
		name          string
		activeBalance primitives.Gwei
		expected      primitives.Gwei
	}{
		{
			name:          "less than MIN_PER_EPOCH_ACTIVATION_BALANCE_CHURN_LIMIT",
			activeBalance: 1,
			expected:      primitives.Gwei(params.BeaconConfig().MinPerEpochActivationBalanceChurnLimit),
		},
		{
			name:          "more than MIN_PER_EPOCH_ACTIVATION_BALANCE_CHURN_LIMIT",
			activeBalance: primitives.Gwei(2000 * params.BeaconConfig().EffectiveBalanceIncrement * params.BeaconConfig().ChurnLimitQuotient),
			expected:      primitives.Gwei(2000 * params.BeaconConfig().EffectiveBalanceIncrement),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, helpers.ActivationBalanceChurnLimit(tt.activeBalance))
		})
	}
}

func TestExitBalanceChurnLimit(t *testing.T) {
	tests := []struct {
		name          string
		activeBalance primitives.Gwei
		expected      primitives.Gwei
	}{
		{
			name:          "less than MIN_PER_EPOCH_CHURN_LIMIT_ALPACA",
			activeBalance: 111,
			expected:      primitives.Gwei(params.BeaconConfig().MinPerEpochChurnLimitAlpaca),
		},
		{
			name:          "modulo EFFECTIVE_BALANCE_INCREMENT",
			activeBalance: primitives.Gwei(111 + params.BeaconConfig().MinPerEpochChurnLimitAlpaca*params.BeaconConfig().ChurnLimitQuotient),
			expected:      primitives.Gwei(params.BeaconConfig().MinPerEpochChurnLimitAlpaca),
		},
		{
			name:          "more than MIN_PER_EPOCH_CHURN_LIMIT_ELECTRA",
			activeBalance: primitives.Gwei(2000 * params.BeaconConfig().EffectiveBalanceIncrement * params.BeaconConfig().ChurnLimitQuotient),
			expected:      primitives.Gwei(2000 * params.BeaconConfig().EffectiveBalanceIncrement),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, helpers.ExitBalanceChurnLimit(tt.activeBalance))
		})
	}
}
