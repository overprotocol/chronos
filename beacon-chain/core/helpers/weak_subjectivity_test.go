package helpers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestWeakSubjectivity_ComputeWeakSubjectivityPeriod(t *testing.T) {
	tests := []struct {
		valCount   uint64
		avgBalance uint64
		want       primitives.Epoch
	}{
		// Asserting that we get the same numbers as defined in the reference table:
		// https://github.com/ethereum/consensus-specs/blob/master/specs/phase0/weak-subjectivity.md#calculating-the-weak-subjectivity-period
		{valCount: 32768, avgBalance: 256, want: 256},
		{valCount: 65536, avgBalance: 256, want: 256},
		{valCount: 131072, avgBalance: 256, want: 256},
		{valCount: 262144, avgBalance: 256, want: 256},
		{valCount: 524288, avgBalance: 256, want: 256},
		{valCount: 1048576, avgBalance: 256, want: 256},
		{valCount: 32768, avgBalance: 8192, want: 256},
		{valCount: 65536, avgBalance: 8192, want: 256},
		{valCount: 131072, avgBalance: 8192, want: 257},
		{valCount: 262144, avgBalance: 8192, want: 258},
		{valCount: 524288, avgBalance: 8192, want: 260},
		{valCount: 1048576, avgBalance: 8192, want: 265},
		{valCount: 32768, avgBalance: 16384, want: 307},
		{valCount: 65536, avgBalance: 16384, want: 307},
		{valCount: 131072, avgBalance: 16384, want: 307},
		{valCount: 262144, avgBalance: 16384, want: 307},
		{valCount: 524288, avgBalance: 16384, want: 307},
		{valCount: 1048576, avgBalance: 16384, want: 307},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("valCount: %d, avgBalance: %d", tt.valCount, tt.avgBalance), func(t *testing.T) {
			// Reset committee cache - as we need to recalculate active validator set for each test.
			helpers.ClearCache()

			got, err := helpers.ComputeWeakSubjectivityPeriod(context.Background(), genState(t, tt.valCount, tt.avgBalance), params.BeaconConfig())
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "valCount: %v, avgBalance: %v", tt.valCount, tt.avgBalance)
		})
	}
}

type mockWsCheckpoint func() (stateRoot [32]byte, blockRoot [32]byte, e primitives.Epoch)

func TestWeakSubjectivity_IsWithinWeakSubjectivityPeriod(t *testing.T) {
	tests := []struct {
		name            string
		epoch           primitives.Epoch
		genWsState      func() state.ReadOnlyBeaconState
		genWsCheckpoint mockWsCheckpoint
		want            bool
		wantedErr       string
	}{
		{
			name: "nil weak subjectivity state",
			genWsState: func() state.ReadOnlyBeaconState {
				return nil
			},
			genWsCheckpoint: func() ([32]byte, [32]byte, primitives.Epoch) {
				return [32]byte{}, [32]byte{}, 42
			},
			wantedErr: "invalid weak subjectivity state or checkpoint",
		},
		{
			name: "state and checkpoint roots do not match",
			genWsState: func() state.ReadOnlyBeaconState {
				beaconState := genState(t, 128, 256)
				require.NoError(t, beaconState.SetSlot(42*params.BeaconConfig().SlotsPerEpoch))
				err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
					Slot:      42 * params.BeaconConfig().SlotsPerEpoch,
					StateRoot: bytesutil.PadTo([]byte("stateroot1"), 32),
				})
				require.NoError(t, err)
				return beaconState
			},
			genWsCheckpoint: func() ([32]byte, [32]byte, primitives.Epoch) {
				var sr [32]byte
				copy(sr[:], bytesutil.PadTo([]byte("stateroot2"), 32))
				return sr, [32]byte{}, 42
			},
			wantedErr: fmt.Sprintf("state (%#x) and checkpoint (%#x) roots do not match",
				bytesutil.PadTo([]byte("stateroot1"), 32), bytesutil.PadTo([]byte("stateroot2"), 32)),
		},
		{
			name: "state and checkpoint epochs do not match",
			genWsState: func() state.ReadOnlyBeaconState {
				beaconState := genState(t, 128, 256)
				require.NoError(t, beaconState.SetSlot(42*params.BeaconConfig().SlotsPerEpoch))
				err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
					Slot:      42 * params.BeaconConfig().SlotsPerEpoch,
					StateRoot: bytesutil.PadTo([]byte("stateroot"), 32),
				})
				require.NoError(t, err)
				return beaconState
			},
			genWsCheckpoint: func() ([32]byte, [32]byte, primitives.Epoch) {
				var sr [32]byte
				copy(sr[:], bytesutil.PadTo([]byte("stateroot"), 32))
				return sr, [32]byte{}, 43
			},
			wantedErr: "state (42) and checkpoint (43) epochs do not match",
		},
		{
			name: "no active validators",
			genWsState: func() state.ReadOnlyBeaconState {
				beaconState := genState(t, 0, 256)
				require.NoError(t, beaconState.SetSlot(42*params.BeaconConfig().SlotsPerEpoch))
				err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
					Slot:      42 * params.BeaconConfig().SlotsPerEpoch,
					StateRoot: bytesutil.PadTo([]byte("stateroot"), 32),
				})
				require.NoError(t, err)
				return beaconState
			},
			genWsCheckpoint: func() ([32]byte, [32]byte, primitives.Epoch) {
				var sr [32]byte
				copy(sr[:], bytesutil.PadTo([]byte("stateroot"), 32))
				return sr, [32]byte{}, 42
			},
			wantedErr: "cannot compute weak subjectivity period: no active validators found",
		},
		{
			name:  "outside weak subjectivity period",
			epoch: 300,
			genWsState: func() state.ReadOnlyBeaconState {
				beaconState := genState(t, 128, 256)
				require.NoError(t, beaconState.SetSlot(42*params.BeaconConfig().SlotsPerEpoch))
				err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
					Slot:      42 * params.BeaconConfig().SlotsPerEpoch,
					StateRoot: bytesutil.PadTo([]byte("stateroot"), 32),
				})
				require.NoError(t, err)
				return beaconState
			},
			genWsCheckpoint: func() ([32]byte, [32]byte, primitives.Epoch) {
				var sr [32]byte
				copy(sr[:], bytesutil.PadTo([]byte("stateroot"), 32))
				return sr, [32]byte{}, 42
			},
			want: false,
		},
		{
			name:  "within weak subjectivity period",
			epoch: 298,
			genWsState: func() state.ReadOnlyBeaconState {
				beaconState := genState(t, 128, 256)
				require.NoError(t, beaconState.SetSlot(42*params.BeaconConfig().SlotsPerEpoch))
				err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
					Slot:      42 * params.BeaconConfig().SlotsPerEpoch,
					StateRoot: bytesutil.PadTo([]byte("stateroot"), 32),
				})
				require.NoError(t, err)
				return beaconState
			},
			genWsCheckpoint: func() ([32]byte, [32]byte, primitives.Epoch) {
				var sr [32]byte
				copy(sr[:], bytesutil.PadTo([]byte("stateroot"), 32))
				return sr, [32]byte{}, 42
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()

			sr, _, e := tt.genWsCheckpoint()
			got, err := helpers.IsWithinWeakSubjectivityPeriod(context.Background(), tt.epoch, tt.genWsState(), sr, e, params.BeaconConfig())
			if tt.wantedErr != "" {
				assert.Equal(t, false, got)
				assert.ErrorContains(t, tt.wantedErr, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWeakSubjectivity_ParseWeakSubjectivityInputString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkpt   *ethpb.Checkpoint
		wantedErr string
	}{
		{
			name:      "No column in string",
			input:     "0x111111;123",
			wantedErr: "did not contain column",
		},
		{
			name:      "Too many columns in string",
			input:     "0x010203:123:456",
			wantedErr: "weak subjectivity checkpoint input should be in `block_root:epoch_number` format",
		},
		{
			name:      "Incorrect block root length",
			input:     "0x010203:987",
			wantedErr: "block root is not length of 32",
		},
		{
			name:  "Correct input",
			input: "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF:123456789",
			checkpt: &ethpb.Checkpoint{
				Root:  []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				Epoch: primitives.Epoch(123456789),
			},
		},
		{
			name:  "Correct input without 0x",
			input: "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF:123456789",
			checkpt: &ethpb.Checkpoint{
				Root:  []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				Epoch: primitives.Epoch(123456789),
			},
		},
		{
			name:  "Correct input",
			input: "0xF0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF:123456789",
			checkpt: &ethpb.Checkpoint{
				Root:  []byte{0xf0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				Epoch: primitives.Epoch(123456789),
			},
		},
		{
			name:  "Correct input without 0x",
			input: "F0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF:123456789",
			checkpt: &ethpb.Checkpoint{
				Root:  []byte{0xf0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				Epoch: primitives.Epoch(123456789),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()

			wsCheckpt, err := helpers.ParseWeakSubjectivityInputString(tt.input)
			if tt.wantedErr != "" {
				require.ErrorContains(t, tt.wantedErr, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, wsCheckpt)
			require.DeepEqual(t, tt.checkpt.Root, wsCheckpt.Root, "Roots do not match")
			require.Equal(t, tt.checkpt.Epoch, wsCheckpt.Epoch, "Epochs do not match")
		})
	}
}

func genState(t *testing.T, valCount, avgBalance uint64) state.BeaconState {
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)

	validators := make([]*ethpb.Validator, valCount)
	balances := make([]uint64, len(validators))
	for i := uint64(0); i < valCount; i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:             make([]byte, params.BeaconConfig().BLSPubkeyLength),
			WithdrawalCredentials: make([]byte, 32),
			EffectiveBalance:      avgBalance * 1e9,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
		balances[i] = validators[i].EffectiveBalance
	}

	require.NoError(t, beaconState.SetValidators(validators))
	require.NoError(t, beaconState.SetBalances(balances))

	return beaconState
}

func TestMinEpochsForBlockRequests(t *testing.T) {
	helpers.ClearCache()

	params.SetActiveTestCleanup(t, params.MainnetConfig())
	var expected primitives.Epoch = 33024
	// expected value of 33024 via spec commentary:
	// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/p2p-interface.md#why-are-blocksbyrange-requests-only-required-to-be-served-for-the-latest-min_epochs_for_block_requests-epochs
	//	MIN_EPOCHS_FOR_BLOCK_REQUESTS is calculated using the arithmetic from compute_weak_subjectivity_period found in the weak subjectivity guide. Specifically to find this max epoch range, we use the worst case event of a very large validator size (>= MIN_PER_EPOCH_CHURN_LIMIT * CHURN_LIMIT_QUOTIENT).
	//
	//	MIN_EPOCHS_FOR_BLOCK_REQUESTS = (
	//	    MIN_VALIDATOR_WITHDRAWABILITY_DELAY
	//	    + MAX_SAFETY_DECAY * CHURN_LIMIT_QUOTIENT // (2 * 100)
	//	)
	//
	//	Where MAX_SAFETY_DECAY = 100 and thus MIN_EPOCHS_FOR_BLOCK_REQUESTS = 33024 (~5 months).
	require.Equal(t, expected, helpers.MinEpochsForBlockRequests())
}
