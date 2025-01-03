package precompute

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestProcessRewardsAndPenaltiesPrecompute(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(2048)
	base := buildState(e+3, validatorCount)
	atts := make([]*ethpb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &ethpb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				Source: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
			},
			AggregationBits: bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	base.PreviousEpochAttestations = atts

	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)

	vp, bp, err := New(context.Background(), beaconState)
	require.NoError(t, err)
	vp, bp, err = ProcessAttestations(context.Background(), beaconState, vp, bp)
	require.NoError(t, err)

	processedState, err := ProcessRewardsAndPenaltiesPrecompute(beaconState, bp, vp, AttestationsDelta, ProposersDelta)
	require.NoError(t, err)
	require.Equal(t, true, processedState.Version() == version.Phase0)

	// Indices that voted everything except for head, lost a bit money
	wanted := uint64(255910816213)
	assert.Equal(t, wanted, beaconState.Balances()[4], "Unexpected balance")

	// Indices that did not vote, lost more money
	wanted = uint64(255940544141)
	assert.Equal(t, wanted, beaconState.Balances()[0], "Unexpected balance")
}

func TestAttestationDeltas_ZeroEpoch(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(2048)
	base := buildState(e+2, validatorCount)
	atts := make([]*ethpb.PendingAttestation, 3)
	var emptyRoot [32]byte
	for i := 0; i < len(atts); i++ {
		atts[i] = &ethpb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Root: emptyRoot[:],
				},
				Source: &ethpb.Checkpoint{
					Root: emptyRoot[:],
				},
				BeaconBlockRoot: emptyRoot[:],
			},
			AggregationBits: bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	base.PreviousEpochAttestations = atts
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)

	pVals, pBal, err := New(context.Background(), beaconState)
	assert.NoError(t, err)
	pVals, pBal, err = ProcessAttestations(context.Background(), beaconState, pVals, pBal)
	require.NoError(t, err)

	pBal.ActiveCurrentEpoch = 0 // Could cause a divide by zero panic.

	_, _, _, err = AttestationsDelta(beaconState, pBal, pVals)
	require.NoError(t, err)
}

func TestAttestationDeltas_ZeroInclusionDelay(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(2048)
	base := buildState(e+2, validatorCount)
	atts := make([]*ethpb.PendingAttestation, 3)
	var emptyRoot [32]byte
	for i := 0; i < len(atts); i++ {
		atts[i] = &ethpb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Root: emptyRoot[:],
				},
				Source: &ethpb.Checkpoint{
					Root: emptyRoot[:],
				},
				BeaconBlockRoot: emptyRoot[:],
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			// Inclusion delay of 0 is not possible in a valid state and could cause a divide by
			// zero panic.
			InclusionDelay: 0,
		}
	}
	base.PreviousEpochAttestations = atts
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)

	pVals, pBal, err := New(context.Background(), beaconState)
	require.NoError(t, err)
	_, _, err = ProcessAttestations(context.Background(), beaconState, pVals, pBal)
	require.ErrorContains(t, "attestation with inclusion delay of 0", err)
}

func buildState(slot primitives.Slot, validatorCount uint64) *ethpb.BeaconState {
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}
	return &ethpb.BeaconState{
		Slot:                        slot,
		Balances:                    validatorBalances,
		Validators:                  validators,
		RandaoMixes:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:                  make([][]byte, params.BeaconConfig().SlotsPerEpoch*10),
		FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
	}
}

func TestProposerDeltaPrecompute_HappyCase(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(10)
	base := buildState(e, validatorCount)
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)

	proposerIndex := primitives.ValidatorIndex(1)
	b := &Balance{ActiveCurrentEpoch: 1000000000000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex},
	}
	r, _, err := ProposersDelta(beaconState, b, v)
	require.NoError(t, err)

	curEpoch := time.CurrentEpoch(beaconState)
	currentEpochIncrement := b.ActiveCurrentEpoch / params.BeaconConfig().EffectiveBalanceIncrement
	vBalance := v[0].CurrentEpochEffectiveBalance / params.BeaconConfig().EffectiveBalanceIncrement
	baseReward := vBalance * helpers.EpochIssuance(curEpoch) / currentEpochIncrement / params.BeaconConfig().BaseRewardsPerEpoch
	proposerReward := baseReward / params.BeaconConfig().ProposerRewardQuotient

	assert.Equal(t, proposerReward, r[proposerIndex], "Unexpected proposer reward")
}

func TestProposerDeltaPrecompute_ValidatorIndexOutOfRange(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(10)
	base := buildState(e, validatorCount)
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)

	proposerIndex := primitives.ValidatorIndex(validatorCount)
	b := &Balance{ActiveCurrentEpoch: 1000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex},
	}
	_, _, err = ProposersDelta(beaconState, b, v)
	assert.ErrorContains(t, "proposer index out of range", err)
}

func TestProposerDeltaPrecompute_SlashedCase(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(10)
	base := buildState(e, validatorCount)
	beaconState, err := state_native.InitializeFromProtoPhase0(base)
	require.NoError(t, err)

	proposerIndex := primitives.ValidatorIndex(1)
	b := &Balance{ActiveCurrentEpoch: 1000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex, IsSlashed: true},
	}
	r, _, err := ProposersDelta(beaconState, b, v)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), r[proposerIndex], "Unexpected proposer reward for slashed")
}

// BaseReward takes state and validator index and calculate
// individual validator's base reward quotient.
//
// Spec pseudocode definition:
//
//	def get_base_reward(state: BeaconState, index: ValidatorIndex) -> Gwei:
//	  total_balance = get_total_active_balance(state)
//	  effective_balance = state.validators[index].effective_balance
//	  return Gwei(effective_balance * BASE_REWARD_FACTOR // integer_squareroot(total_balance) // BASE_REWARDS_PER_EPOCH)
func baseReward(state state.ReadOnlyBeaconState, index primitives.ValidatorIndex) (uint64, error) {
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return 0, errors.Wrap(err, "could not calculate active balance")
	}
	val, err := state.ValidatorAtIndexReadOnly(index)
	if err != nil {
		return 0, err
	}
	effectiveBalanceInc := val.EffectiveBalance() / params.BeaconConfig().EffectiveBalanceIncrement
	totalBalanceInc := totalBalance / params.BeaconConfig().EffectiveBalanceIncrement
	reward, _ := helpers.TotalRewardWithReserveUsage(state)
	baseReward := effectiveBalanceInc * reward / totalBalanceInc / params.BeaconConfig().BaseRewardsPerEpoch

	return baseReward, nil
}
