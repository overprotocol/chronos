package helpers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v5/beacon-chain/forkchoice/types"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/hash"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestIsActiveValidator_OK(t *testing.T) {
	tests := []struct {
		a primitives.Epoch
		b bool
	}{
		{a: 0, b: false},
		{a: 10, b: true},
		{a: 100, b: false},
		{a: 1000, b: false},
		{a: 64, b: true},
	}
	for _, test := range tests {
		validator := &ethpb.Validator{ActivationEpoch: 10, ExitEpoch: 100}
		assert.Equal(t, test.b, helpers.IsActiveValidator(validator, test.a), "IsActiveValidator(%d)", test.a)
	}
}

func TestIsActiveValidatorUsingTrie_OK(t *testing.T) {
	tests := []struct {
		a primitives.Epoch
		b bool
	}{
		{a: 0, b: false},
		{a: 10, b: true},
		{a: 100, b: false},
		{a: 1000, b: false},
		{a: 64, b: true},
	}
	val := &ethpb.Validator{ActivationEpoch: 10, ExitEpoch: 100}
	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: []*ethpb.Validator{val}})
	require.NoError(t, err)
	for _, test := range tests {
		readOnlyVal, err := beaconState.ValidatorAtIndexReadOnly(0)
		require.NoError(t, err)
		assert.Equal(t, test.b, helpers.IsActiveValidatorUsingTrie(readOnlyVal, test.a), "IsActiveValidatorUsingTrie(%d)", test.a)
	}
}

func TestIsActiveNonSlashedValidatorUsingTrie_OK(t *testing.T) {
	tests := []struct {
		a primitives.Epoch
		s bool
		b bool
	}{
		{a: 0, s: false, b: false},
		{a: 10, s: false, b: true},
		{a: 100, s: false, b: false},
		{a: 1000, s: false, b: false},
		{a: 64, s: false, b: true},
		{a: 0, s: true, b: false},
		{a: 10, s: true, b: false},
		{a: 100, s: true, b: false},
		{a: 1000, s: true, b: false},
		{a: 64, s: true, b: false},
	}
	for _, test := range tests {
		val := &ethpb.Validator{ActivationEpoch: 10, ExitEpoch: 100}
		val.Slashed = test.s
		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: []*ethpb.Validator{val}})
		require.NoError(t, err)
		readOnlyVal, err := beaconState.ValidatorAtIndexReadOnly(0)
		require.NoError(t, err)
		assert.Equal(t, test.b, helpers.IsActiveNonSlashedValidatorUsingTrie(readOnlyVal, test.a), "IsActiveNonSlashedValidatorUsingTrie(%d)", test.a)
	}
}

func TestIsSlashableValidator_OK(t *testing.T) {
	tests := []struct {
		name      string
		validator *ethpb.Validator
		epoch     primitives.Epoch
		slashable bool
	}{
		{
			name:      "Unset withdrawable, slashable",
			validator: &ethpb.Validator{},
			epoch:     0,
			slashable: true,
		},
		{
			name: "before withdrawable, slashable",
			validator: &ethpb.Validator{
				ExitEpoch: 5,
			},
			epoch:     6,
			slashable: true,
		},
		{
			name: "inactive, not slashable",
			validator: &ethpb.Validator{
				ActivationEpoch: 5,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			},
			epoch:     2,
			slashable: false,
		},
		{
			name: "after withdrawable, not slashable",
			validator: &ethpb.Validator{
				ExitEpoch: 0,
			},
			epoch:     257,
			slashable: false,
		},
		{
			name: "slashed and withdrawable, not slashable",
			validator: &ethpb.Validator{
				Slashed:   true,
				ExitEpoch: 0,
			},
			epoch:     257,
			slashable: false,
		},
		{
			name: "slashed, not slashable",
			validator: &ethpb.Validator{
				Slashed:   true,
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			},
			epoch:     2,
			slashable: false,
		},
		{
			name: "inactive and slashed, not slashable",
			validator: &ethpb.Validator{
				Slashed:         true,
				ActivationEpoch: 4,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			},
			epoch:     2,
			slashable: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Run("without trie", func(t *testing.T) {
				withdrawableEpoch := helpers.GetWithdrawableEpoch(test.validator.ExitEpoch, test.validator.Slashed)
				slashableValidator := helpers.IsSlashableValidator(test.validator.ActivationEpoch,
					withdrawableEpoch, test.validator.Slashed, test.epoch)
				assert.Equal(t, test.slashable, slashableValidator, "Expected active validator slashable to be %t", test.slashable)
			})
			t.Run("with trie", func(t *testing.T) {
				beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: []*ethpb.Validator{test.validator}})
				require.NoError(t, err)
				readOnlyVal, err := beaconState.ValidatorAtIndexReadOnly(0)
				require.NoError(t, err)
				slashableValidator := helpers.IsSlashableValidatorUsingTrie(readOnlyVal, test.epoch)
				assert.Equal(t, test.slashable, slashableValidator, "Expected active validator slashable to be %t", test.slashable)
			})
		})
	}
}

func TestBeaconProposerIndex_OK(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig()
	c.MinGenesisActiveValidatorCount = 16384
	params.OverrideBeaconConfig(c)
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Validators:  validators,
		Slot:        0,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)

	tests := []struct {
		slot  primitives.Slot
		index primitives.ValidatorIndex
	}{
		{
			slot:  1,
			index: 2039,
		},
		{
			slot:  5,
			index: 1895,
		},
		{
			slot:  19,
			index: 1947,
		},
		{
			slot:  30,
			index: 369,
		},
		{
			slot:  43,
			index: 464,
		},
	}

	for _, tt := range tests {
		helpers.ClearCache()

		require.NoError(t, state.SetSlot(tt.slot))
		result, err := helpers.BeaconProposerIndex(context.Background(), state)
		require.NoError(t, err, "Failed to get shard and committees at slot")
		assert.Equal(t, tt.index, result, "Result index was an unexpected value")
	}
}

func TestBeaconProposerIndex_BadState(t *testing.T) {
	helpers.ClearCache()

	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig()
	c.MinGenesisActiveValidatorCount = 16384
	params.OverrideBeaconConfig(c)
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	roots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := uint64(0); i < uint64(params.BeaconConfig().SlotsPerHistoricalRoot); i++ {
		roots[i] = make([]byte, fieldparams.RootLength)
	}

	state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Validators:  validators,
		Slot:        0,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:  roots,
		StateRoots:  roots,
	})
	require.NoError(t, err)
	// Set a very high slot, so that retrieved block root will be
	// non existent for the proposer cache.
	require.NoError(t, state.SetSlot(100))
	_, err = helpers.BeaconProposerIndex(context.Background(), state)
	require.NoError(t, err)
}

func TestComputeProposerIndex_Compatibility(t *testing.T) {
	helpers.ClearCache()

	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)

	indices, err := helpers.ActiveValidatorIndices(context.Background(), state, 0)
	require.NoError(t, err)

	var proposerIndices []primitives.ValidatorIndex
	seed, err := helpers.Seed(state, 0, params.BeaconConfig().DomainBeaconProposer)
	require.NoError(t, err)
	for i := uint64(0); i < uint64(params.BeaconConfig().SlotsPerEpoch); i++ {
		seedWithSlot := append(seed[:], bytesutil.Bytes8(i)...)
		seedWithSlotHash := hash.Hash(seedWithSlot)
		index, err := helpers.ComputeProposerIndex(state, indices, seedWithSlotHash)
		require.NoError(t, err)
		proposerIndices = append(proposerIndices, index)
	}

	var wantedProposerIndices []primitives.ValidatorIndex
	seed, err = helpers.Seed(state, 0, params.BeaconConfig().DomainBeaconProposer)
	require.NoError(t, err)
	for i := uint64(0); i < uint64(params.BeaconConfig().SlotsPerEpoch); i++ {
		seedWithSlot := append(seed[:], bytesutil.Bytes8(i)...)
		seedWithSlotHash := hash.Hash(seedWithSlot)
		index, err := computeProposerIndexWithValidators(state.Validators(), indices, seedWithSlotHash)
		require.NoError(t, err)
		wantedProposerIndices = append(wantedProposerIndices, index)
	}
	assert.DeepEqual(t, wantedProposerIndices, proposerIndices, "Wanted proposer indices from ComputeProposerIndexWithValidators does not match")
}

func TestDelayedActivationExitEpoch_OK(t *testing.T) {
	helpers.ClearCache()

	epoch := primitives.Epoch(9999)
	wanted := epoch + 1 + params.BeaconConfig().MaxSeedLookahead
	assert.Equal(t, wanted, helpers.ActivationExitEpoch(epoch))
}

func TestActiveValidatorCount_Genesis(t *testing.T) {
	helpers.ClearCache()

	c := 1000
	validators := make([]*ethpb.Validator, c)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Slot:        0,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)

	// Preset cache to a bad count.
	seed, err := helpers.Seed(beaconState, 0, params.BeaconConfig().DomainBeaconAttester)
	require.NoError(t, err)
	require.NoError(t, helpers.CommitteeCache().AddCommitteeShuffledList(context.Background(), &cache.Committees{Seed: seed, ShuffledIndices: []primitives.ValidatorIndex{1, 2, 3}}))
	validatorCount, err := helpers.ActiveValidatorCount(context.Background(), beaconState, time.CurrentEpoch(beaconState))
	require.NoError(t, err)
	assert.Equal(t, uint64(c), validatorCount, "Did not get the correct validator count")
}

func TestChurnLimit_OK(t *testing.T) {
	tests := []struct {
		validatorCount int
		epoch          primitives.Epoch
		wantedChurn    uint64
	}{
		{validatorCount: 1000, wantedChurn: 4},
		{validatorCount: 100000, wantedChurn: 4},
		{validatorCount: 327680, wantedChurn: 5},
		{validatorCount: 1000000, wantedChurn: 15 /* validatorCount/churnLimitQuotient */},
		{validatorCount: 2000000, wantedChurn: 30 /* validatorCount/churnLimitQuotient */},
	}
	for _, test := range tests {
		helpers.ClearCache()

		validators := make([]*ethpb.Validator, test.validatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			}
		}

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot:        1,
			Validators:  validators,
			RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		})
		require.NoError(t, err)
		validatorCount, err := helpers.ActiveValidatorCount(context.Background(), beaconState, time.CurrentEpoch(beaconState))
		require.NoError(t, err)
		resultChurn := helpers.ValidatorActivationChurnLimit(validatorCount)
		assert.Equal(t, test.wantedChurn, resultChurn, "ValidatorActivationChurnLimit(%d)", test.validatorCount)
	}
}

func TestActivationBalanceChurnLimit_OK(t *testing.T) {
	tests := []struct {
		validatorCount int
		epoch          primitives.Epoch
		wantedChurn    uint64
	}{
		{validatorCount: 1000, wantedChurn: 4096000000000},
		{validatorCount: 1048831, wantedChurn: 4096000000000},
		{validatorCount: 1048832, wantedChurn: 4097000000000},
		{validatorCount: 2000000, wantedChurn: 7812000000000},
		{validatorCount: 4000000, wantedChurn: 15625000000000},
	}
	for _, test := range tests {
		helpers.ClearCache()

		validators := make([]*ethpb.Validator, test.validatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			}
		}

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot:        1,
			Validators:  validators,
			RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		})
		require.NoError(t, err)
		totalBalance, err := helpers.TotalActiveBalance(beaconState)
		require.NoError(t, err)
		resultChurn := helpers.ActivationBalanceChurnLimit(primitives.Gwei(totalBalance))
		assert.Equal(t, test.wantedChurn, uint64(resultChurn), "ActivationBalanceChurnLimit(%d)", test.validatorCount)
	}
}

func TestExitBalanceChurnLimit_OK(t *testing.T) {
	tests := []struct {
		validatorCount int
		epoch          primitives.Epoch
		wantedChurn    uint64
	}{
		{validatorCount: 1000, wantedChurn: 1024000000000},
		{validatorCount: 262144, wantedChurn: 1024000000000},
		{validatorCount: 262400, wantedChurn: 1025000000000},
		{validatorCount: 2000000, wantedChurn: 7812000000000},
		{validatorCount: 4000000, wantedChurn: 15625000000000},
	}
	for _, test := range tests {
		helpers.ClearCache()

		validators := make([]*ethpb.Validator, test.validatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			}
		}

		beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Slot:        1,
			Validators:  validators,
			RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		})
		require.NoError(t, err)
		totalBalance, err := helpers.TotalActiveBalance(beaconState)
		require.NoError(t, err)
		resultChurn := helpers.ExitBalanceChurnLimit(primitives.Gwei(totalBalance))
		assert.Equal(t, test.wantedChurn, uint64(resultChurn), "ExitBalanceChurnLimit(%d)", test.validatorCount)
	}
}

// Test basic functionality of ActiveValidatorIndices without caching. This test will need to be
// rewritten when releasing some cache flag.
func TestActiveValidatorIndices(t *testing.T) {
	//farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	type args struct {
		state *ethpb.BeaconState
		epoch primitives.Epoch
	}
	tests := []struct {
		name      string
		args      args
		want      []primitives.ValidatorIndex
		wantedErr string
	}{
		/*{
			name: "all_active_epoch_10",
			args: args{
				state: &ethpb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
					},
				},
				epoch: 10,
			},
			want: []primitives.ValidatorIndex{0, 1, 2},
		},
		{
			name: "some_active_epoch_10",
			args: args{
				state: &ethpb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       1,
						},
					},
				},
				epoch: 10,
			},
			want: []primitives.ValidatorIndex{0, 1},
		},
		{
			name: "some_active_with_recent_new_epoch_10",
			args: args{
				state: &ethpb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       1,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
					},
				},
				epoch: 10,
			},
			want: []primitives.ValidatorIndex{0, 1, 3},
		},
		{
			name: "some_active_with_recent_new_epoch_10",
			args: args{
				state: &ethpb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       1,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
					},
				},
				epoch: 10,
			},
			want: []primitives.ValidatorIndex{0, 1, 3},
		},
		{
			name: "some_active_with_recent_new_epoch_10",
			args: args{
				state: &ethpb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       1,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
					},
				},
				epoch: 10,
			},
			want: []primitives.ValidatorIndex{0, 2, 3},
		},*/
		{
			name: "impossible_zero_validators", // Regression test for issue #13051
			args: args{
				state: &ethpb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators:  make([]*ethpb.Validator, 0),
				},
				epoch: 10,
			},
			wantedErr: "state has nil validator slice",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()

			s, err := state_native.InitializeFromProtoPhase0(tt.args.state)
			require.NoError(t, err)
			got, err := helpers.ActiveValidatorIndices(context.Background(), s, tt.args.epoch)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
				return
			}
			assert.DeepEqual(t, tt.want, got, "ActiveValidatorIndices()")
		})
	}
}

func TestComputeProposerIndex(t *testing.T) {
	seed := bytesutil.ToBytes32([]byte("seed"))
	type args struct {
		validators []*ethpb.Validator
		indices    []primitives.ValidatorIndex
		seed       [32]byte
	}
	tests := []struct {
		name             string
		isElectraOrAbove bool
		args             args
		want             primitives.ValidatorIndex
		wantedErr        string
	}{
		{
			name: "all_active_indices",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []primitives.ValidatorIndex{0, 1, 2, 3, 4},
				seed:    seed,
			},
			want: 2,
		},
		{ // Regression test for https://github.com/prysmaticlabs/prysm/issues/4259.
			name: "1_active_index",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []primitives.ValidatorIndex{3},
				seed:    seed,
			},
			want: 3,
		},
		{
			name: "empty_active_indices",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []primitives.ValidatorIndex{},
				seed:    seed,
			},
			wantedErr: "empty active indices list",
		},
		{
			name: "active_indices_out_of_range",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []primitives.ValidatorIndex{100},
				seed:    seed,
			},
			wantedErr: "active index out of range",
		},
		{
			name: "second_half_active",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []primitives.ValidatorIndex{5, 6, 7, 8, 9},
				seed:    seed,
			},
			want: 7,
		},
		{
			name:             "electra_probability_changes",
			isElectraOrAbove: true,
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalanceAlpaca},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []primitives.ValidatorIndex{3},
				seed:    seed,
			},
			want: 3,
		},
		{
			name:             "electra_probability_changes_all_active",
			isElectraOrAbove: true,
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalanceAlpaca},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalanceAlpaca},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance}, // skip this one
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalanceAlpaca},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalanceAlpaca},
				},
				indices: []primitives.ValidatorIndex{0, 1, 2, 3, 4},
				seed:    seed,
			},
			want: 4,
		},
		{
			name:             "electra_probability_returns_first_validator_with_criteria",
			isElectraOrAbove: true,
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: 1},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalanceAlpaca},
					{EffectiveBalance: 1},
					{EffectiveBalance: 1},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalanceAlpaca},
				},
				indices: []primitives.ValidatorIndex{1},
				seed:    seed,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()

			bState := &ethpb.BeaconState{Validators: tt.args.validators}
			stTrie, err := state_native.InitializeFromProtoUnsafePhase0(bState)
			require.NoError(t, err)
			if tt.isElectraOrAbove {
				stTrie, err = state_native.InitializeFromProtoUnsafeElectra(&ethpb.BeaconStateElectra{Validators: tt.args.validators})
				require.NoError(t, err)
			}

			got, err := helpers.ComputeProposerIndex(stTrie, tt.args.indices, tt.args.seed)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
				return
			}
			assert.NoError(t, err, "received unexpected error")
			assert.Equal(t, tt.want, got, "ComputeProposerIndex()")
		})
	}
}

func TestIsEligibleForActivationQueue(t *testing.T) {
	params.SetupForkEpochConfigForTest()
	tests := []struct {
		name         string
		validator    *ethpb.Validator
		currentEpoch primitives.Epoch
		want         bool
	}{
		{
			name:         "Eligible",
			validator:    &ethpb.Validator{ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
			currentEpoch: primitives.Epoch(params.BeaconConfig().AlpacaForkEpoch - 1),
			want:         true,
		},
		{
			name:         "Incorrect activation eligibility epoch",
			validator:    &ethpb.Validator{ActivationEligibilityEpoch: 1, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
			currentEpoch: primitives.Epoch(params.BeaconConfig().AlpacaForkEpoch - 1),
			want:         false,
		},
		{
			name:         "Not enough balance",
			validator:    &ethpb.Validator{ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: 1},
			currentEpoch: primitives.Epoch(params.BeaconConfig().AlpacaForkEpoch - 1),
			want:         false,
		},
		{
			name:         "More than max effective balance before alpaca",
			validator:    &ethpb.Validator{ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance + 1},
			currentEpoch: primitives.Epoch(params.BeaconConfig().AlpacaForkEpoch - 1),
			want:         false,
		},
		{
			name:         "More than min activation balance after alpaca",
			validator:    &ethpb.Validator{ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MinActivationBalance + 1},
			currentEpoch: primitives.Epoch(params.BeaconConfig().AlpacaForkEpoch),
			want:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()

			v, err := state_native.NewValidator(tt.validator)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, helpers.IsEligibleForActivationQueue(v, tt.currentEpoch), "IsEligibleForActivationQueue()")
		})
	}
}

func TestIsIsEligibleForActivation(t *testing.T) {
	tests := []struct {
		name      string
		validator *ethpb.Validator
		state     *ethpb.BeaconState
		want      bool
	}{
		{"Eligible",
			&ethpb.Validator{ActivationEligibilityEpoch: 1, ActivationEpoch: params.BeaconConfig().FarFutureEpoch},
			&ethpb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 2}},
			true},
		{"Not yet finalized",
			&ethpb.Validator{ActivationEligibilityEpoch: 1, ActivationEpoch: params.BeaconConfig().FarFutureEpoch},
			&ethpb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)}},
			false},
		{"Incorrect activation epoch",
			&ethpb.Validator{ActivationEligibilityEpoch: 1},
			&ethpb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 2}},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()

			s, err := state_native.InitializeFromProtoPhase0(tt.state)
			require.NoError(t, err)
			assert.Equal(t, tt.want, helpers.IsEligibleForActivation(s, tt.validator), "IsEligibleForActivation()")
		})
	}
}

func TestIsEligibleForBailOut(t *testing.T) {
	bailoutBuffer := params.BeaconConfig().MaxEffectiveBalance * params.BeaconConfig().InactivityPenaltyRate / params.BeaconConfig().InactivityPenaltyRatePrecision

	tests := []struct {
		name      string
		validator *ethpb.Validator
		state     *ethpb.BeaconStateAltair
		leak      bool
		want      bool
	}{
		{"Eligible",
			&ethpb.Validator{
				ActivationEpoch:  0,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance,
			},
			&ethpb.BeaconStateAltair{
				Balances: []uint64{params.BeaconConfig().MaxEffectiveBalance - bailoutBuffer - 1},
			},
			false,
			true,
		},
		{"Eligible (leak)",
			&ethpb.Validator{
				ActivationEpoch:  0,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance,
			},
			&ethpb.BeaconStateAltair{
				Balances:         []uint64{params.BeaconConfig().MaxEffectiveBalance},
				InactivityScores: []uint64{params.BeaconConfig().InactivityLeakBailoutScoreThreshold + 1},
			},
			true,
			true,
		},
		{"Not eligible",
			&ethpb.Validator{
				ActivationEpoch:  0,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance,
			},
			&ethpb.BeaconStateAltair{
				Balances: []uint64{params.BeaconConfig().MaxEffectiveBalance},
			},
			false,
			false,
		},
		{"Not eligible (leak)",
			&ethpb.Validator{
				ActivationEpoch:  0,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				PrincipalBalance: params.BeaconConfig().MaxEffectiveBalance,
			},
			&ethpb.BeaconStateAltair{
				Balances:         []uint64{params.BeaconConfig().MaxEffectiveBalance},
				InactivityScores: []uint64{params.BeaconConfig().InactivityLeakBailoutScoreThreshold},
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()

			s, err := state_native.InitializeFromProtoAltair(tt.state)
			require.NoError(t, err)
			roVal, err := state_native.NewValidator(tt.validator)
			require.NoError(t, err)
			eligible, err := helpers.IsEligibleForBailOut(s, roVal, 0 /* idx */, tt.leak)
			require.NoError(t, err)
			assert.Equal(t, tt.want, eligible, "IsEligibleForBailOut()")
		})
	}
}

func computeProposerIndexWithValidators(validators []*ethpb.Validator, activeIndices []primitives.ValidatorIndex, seed [32]byte) (primitives.ValidatorIndex, error) {
	length := uint64(len(activeIndices))
	if length == 0 {
		return 0, errors.New("empty active indices list")
	}
	maxRandomByte := uint64(1<<8 - 1)
	hashFunc := hash.CustomSHA256Hasher()

	for i := uint64(0); ; i++ {
		candidateIndex, err := helpers.ComputeShuffledIndex(primitives.ValidatorIndex(i%length), length, seed, true /* shuffle */)
		if err != nil {
			return 0, err
		}
		candidateIndex = activeIndices[candidateIndex]
		if uint64(candidateIndex) >= uint64(len(validators)) {
			return 0, errors.New("active index out of range")
		}
		b := append(seed[:], bytesutil.Bytes8(i/32)...)
		randomByte := hashFunc(b)[i%32]
		v := validators[candidateIndex]
		var effectiveBal uint64
		if v != nil {
			effectiveBal = v.EffectiveBalance
		}
		if effectiveBal*maxRandomByte >= params.BeaconConfig().MaxEffectiveBalance*uint64(randomByte) {
			return candidateIndex, nil
		}
	}
}

func TestLastActivatedValidatorIndex_OK(t *testing.T) {
	helpers.ClearCache()

	beaconState, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{})
	require.NoError(t, err)

	validators := make([]*ethpb.Validator, 4)
	balances := make([]uint64, len(validators))
	for i := uint64(0); i < 4; i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:             make([]byte, params.BeaconConfig().BLSPubkeyLength),
			WithdrawalCredentials: make([]byte, 32),
			EffectiveBalance:      32 * 1e9,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
		balances[i] = validators[i].EffectiveBalance
	}
	require.NoError(t, beaconState.SetValidators(validators))
	require.NoError(t, beaconState.SetBalances(balances))

	index, err := helpers.LastActivatedValidatorIndex(context.Background(), beaconState)
	require.NoError(t, err)
	require.Equal(t, index, primitives.ValidatorIndex(3))
}

func TestProposerIndexFromCheckpoint(t *testing.T) {
	helpers.ClearCache()

	e := primitives.Epoch(2)
	r := [32]byte{'a'}
	root := [32]byte{'b'}
	ids := [32]primitives.ValidatorIndex{}
	slot := primitives.Slot(69) // slot 5 in the Epoch
	ids[5] = primitives.ValidatorIndex(19)
	helpers.ProposerIndicesCache().Set(e, r, ids)
	c := &forkchoicetypes.Checkpoint{Root: root, Epoch: e - 1}
	helpers.ProposerIndicesCache().SetCheckpoint(*c, r)
	id, err := helpers.ProposerIndexAtSlotFromCheckpoint(c, slot)
	require.NoError(t, err)
	require.Equal(t, ids[5], id)
}

func TestIsFullyWithdrawableValidator(t *testing.T) {
	tests := []struct {
		name      string
		validator *ethpb.Validator
		balance   uint64
		epoch     primitives.Epoch
		fork      int
		want      bool
	}{
		{
			name: "No ETH1 prefix",
			validator: &ethpb.Validator{
				WithdrawalCredentials: []byte{0xFA, 0xCC},
				ExitEpoch:             2,
			},
			balance: params.BeaconConfig().MaxEffectiveBalance,
			epoch:   3,
			want:    false,
		},
		{
			name: "Wrong withdrawable epoch",
			validator: &ethpb.Validator{
				WithdrawalCredentials: []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC},
				ExitEpoch:             2,
			},
			balance: params.BeaconConfig().MaxEffectiveBalance,
			epoch:   2 + params.BeaconConfig().MinValidatorWithdrawabilityDelay - 1,
			want:    false,
		},
		{
			name: "No balance",
			validator: &ethpb.Validator{
				WithdrawalCredentials: []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC},
				ExitEpoch:             2,
			},
			balance: 0,
			epoch:   2 + params.BeaconConfig().MinValidatorWithdrawabilityDelay,
			want:    false,
		},
		{
			name: "Fully withdrawable",
			validator: &ethpb.Validator{
				WithdrawalCredentials: []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC},
				ExitEpoch:             2,
			},
			balance: params.BeaconConfig().MaxEffectiveBalance,
			epoch:   2 + params.BeaconConfig().MinValidatorWithdrawabilityDelay,
			want:    true,
		},
		{
			name: "Fully withdrawable compounding validator electra",
			validator: &ethpb.Validator{
				WithdrawalCredentials: []byte{params.BeaconConfig().CompoundingWithdrawalPrefixByte, 0xCC},
				ExitEpoch:             2,
			},
			balance: params.BeaconConfig().MaxEffectiveBalance,
			epoch:   params.BeaconConfig().MinValidatorWithdrawabilityDelay + 2,
			fork:    version.Alpaca,
			want:    true,
		},
		{
			name: "Slashed ETH1 validator before withdrawable epoch electra",
			validator: &ethpb.Validator{
				WithdrawalCredentials: []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC},
				ExitEpoch:             2,
				Slashed:               true,
			},
			balance: params.BeaconConfig().MaxEffectiveBalance,
			epoch:   2 + params.BeaconConfig().MinSlashingWithdrawableDelay - 1,
			fork:    version.Alpaca,
			want:    false,
		},
		{
			name: "Slashed ETH1 validator after withdrawable epoch electra",
			validator: &ethpb.Validator{
				WithdrawalCredentials: []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC},
				ExitEpoch:             2,
				Slashed:               true,
			},
			balance: params.BeaconConfig().MaxEffectiveBalance,
			epoch:   2 + params.BeaconConfig().MinSlashingWithdrawableDelay,
			fork:    version.Alpaca,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := state_native.NewValidator(tt.validator)
			require.NoError(t, err)
			assert.Equal(t, tt.want, helpers.IsFullyWithdrawableValidator(v, tt.balance, tt.epoch, tt.fork))
		})
	}
}

func TestIsPartiallyWithdrawableValidator(t *testing.T) {
	tests := []struct {
		name      string
		validator *ethpb.Validator
		balance   uint64
		epoch     primitives.Epoch
		fork      int
		want      bool
	}{
		{
			name: "No ETH1 prefix", // TODO this test is not valid in next spec (nathan)
			validator: &ethpb.Validator{
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawalCredentials: []byte{0xFA, 0xCC},
				PrincipalBalance:      params.BeaconConfig().MaxEffectiveBalance,
			},
			balance: params.BeaconConfig().MaxEffectiveBalance,
			epoch:   3,
			want:    false,
		},
		{
			name: "No balance",
			validator: &ethpb.Validator{
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawalCredentials: []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC},
				PrincipalBalance:      params.BeaconConfig().MaxEffectiveBalance,
			},
			balance: 0,
			epoch:   3,
			want:    false,
		},
		{
			name: "Partially withdrawable",
			validator: &ethpb.Validator{
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				WithdrawalCredentials: []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC},
				PrincipalBalance:      params.BeaconConfig().MaxEffectiveBalance,
			},
			balance: params.BeaconConfig().MaxEffectiveBalance * 2,
			epoch:   3,
			want:    true,
		},
		{
			name: "Fully withdrawable vanilla validator alpaca",
			validator: &ethpb.Validator{
				EffectiveBalance:      params.BeaconConfig().MinActivationBalance,
				WithdrawalCredentials: []byte{params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, 0xCC},
				PrincipalBalance:      params.BeaconConfig().MaxEffectiveBalance,
			},
			balance: params.BeaconConfig().MinActivationBalance * 2,
			epoch:   params.BeaconConfig().AlpacaForkEpoch,
			fork:    version.Alpaca,
			want:    true,
		},
		{
			name: "Fully withdrawable compounding validator alpaca",
			validator: &ethpb.Validator{
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalanceAlpaca,
				WithdrawalCredentials: []byte{params.BeaconConfig().CompoundingWithdrawalPrefixByte, 0xCC},
				PrincipalBalance:      params.BeaconConfig().MaxEffectiveBalance,
			},
			balance: params.BeaconConfig().MaxEffectiveBalanceAlpaca * 2,
			epoch:   params.BeaconConfig().AlpacaForkEpoch,
			fork:    version.Alpaca,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := state_native.NewValidator(tt.validator)
			require.NoError(t, err)
			assert.Equal(t, tt.want, helpers.IsPartiallyWithdrawableValidator(v, tt.balance, tt.epoch, tt.fork))
		})
	}
}

func TestIsSameWithdrawalCredentials(t *testing.T) {
	makeWithdrawalCredentials := func(address []byte) []byte {
		b := make([]byte, 12)
		return append(b, address...)
	}

	tests := []struct {
		name string
		a    *ethpb.Validator
		b    *ethpb.Validator
		want bool
	}{
		{
			"Same credentials",
			&ethpb.Validator{WithdrawalCredentials: makeWithdrawalCredentials([]byte("same"))},
			&ethpb.Validator{WithdrawalCredentials: makeWithdrawalCredentials([]byte("same"))},
			true,
		},
		{
			"Different credentials",
			&ethpb.Validator{WithdrawalCredentials: makeWithdrawalCredentials([]byte("foo"))},
			&ethpb.Validator{WithdrawalCredentials: makeWithdrawalCredentials([]byte("bar"))},
			false,
		},
		{"Handles nil case", nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, helpers.IsSameWithdrawalCredentials(tt.a, tt.b))
		})
	}
}

func TestGetWithdrawableEpoch(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()

	tests := []struct {
		name      string
		exitEpoch primitives.Epoch
		slashed   bool
		want      primitives.Epoch
	}{
		{
			name:      "Returns FAR_FUTURE_EPOCH",
			exitEpoch: config.FarFutureEpoch,
			slashed:   false,
			want:      config.FarFutureEpoch,
		},
		{
			name:      "Slashed validator",
			exitEpoch: 100,
			slashed:   true,
			want:      100 + config.MinSlashingWithdrawableDelay,
		},
		{
			name:      "Regular validator",
			exitEpoch: 100,
			slashed:   false,
			want:      100 + config.MinValidatorWithdrawabilityDelay,
		},
		{
			name:      "Validator with 0 exit epoch",
			exitEpoch: 0,
			slashed:   false,
			want:      0 + config.MinValidatorWithdrawabilityDelay,
		},
		{
			name:      "Slashed validator with 0 exit epoch",
			exitEpoch: 0,
			slashed:   true,
			want:      0 + config.MinSlashingWithdrawableDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helpers.GetWithdrawableEpoch(tt.exitEpoch, tt.slashed)
			if got != tt.want {
				t.Errorf("GetWithdrawableEpoch() = %v, want %v", got, tt.want)
			}
		})
	}
}
