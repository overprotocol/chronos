package altair_test

import (
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func Test_BaseReward(t *testing.T) {
	helpers.ClearCache()
	genState := func(valCount uint64) state.ReadOnlyBeaconState {
		s, _ := util.DeterministicGenesisStateAltair(t, valCount)
		return s
	}
	tests := []struct {
		name      string
		valIdx    primitives.ValidatorIndex
		st        state.ReadOnlyBeaconState
		want      uint64
		errString string
	}{
		{
			name:      "unknown validator",
			valIdx:    2,
			st:        genState(1),
			want:      0,
			errString: "validator index 2 does not exist",
		},
		{
			name:      "active balance is 256eth",
			valIdx:    0,
			st:        genState(1),
			want:      243531202304,
			errString: "",
		},
		{
			name:      "active balance is 256eth * target committee size",
			valIdx:    0,
			st:        genState(params.BeaconConfig().TargetCommitteeSize),
			want:      1902587392,
			errString: "",
		},
		{
			name:      "active balance is 256eth * max validator per  committee size",
			valIdx:    0,
			st:        genState(params.BeaconConfig().MaxValidatorsPerCommittee),
			want:      118911488,
			errString: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := altair.BaseReward(tt.st, tt.valIdx)
			if (err != nil) && (tt.errString != "") {
				require.ErrorContains(t, tt.errString, err)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_BaseRewardWithTotalBalance(t *testing.T) {
	helpers.ClearCache()
	s, _ := util.DeterministicGenesisStateAltair(t, 1)
	tests := []struct {
		name          string
		valIdx        primitives.ValidatorIndex
		activeBalance uint64
		want          uint64
		wantReserve   uint64
		errString     string
	}{
		{
			name:          "active balance is 0",
			valIdx:        0,
			activeBalance: 0,
			want:          0,
			wantReserve:   0,
			errString:     "active balance can't be 0",
		},
		{
			name:          "unknown validator",
			valIdx:        2,
			activeBalance: 1,
			want:          0,
			wantReserve:   0,
			errString:     "validator index 2 does not exist",
		},
		{
			name:          "active balance is 1",
			valIdx:        0,
			activeBalance: 1,
			want:          0,
			wantReserve:   0,
			errString:     "active balance can't be lower than effective balance increment",
		},
		{
			name:          "active balance is 1 over",
			valIdx:        0,
			activeBalance: params.BeaconConfig().EffectiveBalanceIncrement,
			want:          62343987823360,
			wantReserve:   0,
			errString:     "",
		},
		{
			name:          "active balance is 256 over",
			valIdx:        0,
			activeBalance: params.BeaconConfig().MaxEffectiveBalance,
			want:          243531202304,
			wantReserve:   0,
			errString:     "",
		},
		{
			name:          "active balance is 256 over * 1m validators",
			valIdx:        0,
			activeBalance: params.BeaconConfig().MaxEffectiveBalance * 1e6,
			want:          243456,
			wantReserve:   0,
			errString:     "",
		},
		{
			name:          "active balance is max uint64",
			valIdx:        0,
			activeBalance: math.MaxUint64,
			want:          3328,
			wantReserve:   0,
			errString:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reserve, err := altair.BaseRewardWithTotalBalance(s, tt.valIdx, tt.activeBalance)
			if (err != nil) && (tt.errString != "") {
				require.ErrorContains(t, tt.errString, err)
				return
			}
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantReserve, reserve)
		})
	}
}

func Test_BaseRewardPerIncrement(t *testing.T) {
	helpers.ClearCache()
	genState := func(valCount uint64) state.ReadOnlyBeaconState {
		s, _ := util.DeterministicGenesisStateAltair(t, valCount)
		return s
	}
	tests := []struct {
		name          string
		epoch         primitives.Epoch
		activeBalance uint64
		want          uint64
		wantReserve   uint64
		errString     string
	}{
		{
			name:          "active balance is 0",
			epoch:         primitives.Epoch(0),
			activeBalance: 0,
			want:          0,
			wantReserve:   0,
			errString:     "active balance can't be 0",
		},
		{
			name:          "active balance is 1",
			epoch:         primitives.Epoch(0),
			activeBalance: 1,
			want:          0,
			wantReserve:   0,
			errString:     "active balance can't be lower than effective balance increment",
		},
		{
			name:          "active balance is 8 over",
			epoch:         primitives.Epoch(0),
			activeBalance: params.BeaconConfig().EffectiveBalanceIncrement,
			want:          243531202435,
			wantReserve:   0,
			errString:     "",
		},
		{
			name:          "active balance is 256 over",
			epoch:         primitives.Epoch(0),
			activeBalance: params.BeaconConfig().MaxEffectiveBalance,
			want:          951293759,
			wantReserve:   0,
			errString:     "",
		},
		{
			name:          "active balance is 256eth * 1m validators",
			epoch:         primitives.Epoch(0),
			activeBalance: params.BeaconConfig().MaxEffectiveBalance * 1e9,
			want:          15,
			wantReserve:   0,
			errString:     "",
		},
		{
			name:          "active balance is max uint64",
			epoch:         primitives.Epoch(0),
			activeBalance: math.MaxUint64,
			want:          13,
			wantReserve:   0,
			errString:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reserve, err := altair.BaseRewardPerIncrement(genState(20000), tt.activeBalance)
			if (err != nil) && (tt.errString != "") {
				require.ErrorContains(t, tt.errString, err)
				return
			}
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantReserve, reserve)
		})
	}
}
