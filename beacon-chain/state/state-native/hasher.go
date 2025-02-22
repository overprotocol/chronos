package state_native

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// ComputeFieldRootsWithHasher hashes the provided state and returns its respective field roots.
func ComputeFieldRootsWithHasher(ctx context.Context, state *BeaconState) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "ComputeFieldRootsWithHasher")
	defer span.End()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if state == nil {
		return nil, errors.New("nil state")
	}
	var fieldRoots [][]byte
	switch state.version {
	case version.Phase0:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateFieldCount)
	case version.Altair:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateAltairFieldCount)
	case version.Bellatrix:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateBellatrixFieldCount)
	case version.Capella:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateCapellaFieldCount)
	case version.Deneb:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateDenebFieldCount)
	case version.Alpaca:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateAlpacaFieldCount)
	case version.Badger:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateBadgerFieldCount)
	default:
		return nil, fmt.Errorf("unknown state version %s", version.String(state.version))
	}

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.genesisTime)
	fieldRoots[types.GenesisTime.RealPosition()] = genesisRoot[:]

	// Genesis validators root.
	var r [32]byte
	copy(r[:], state.genesisValidatorsRoot[:])
	fieldRoots[types.GenesisValidatorsRoot.RealPosition()] = r[:]

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.slot))
	fieldRoots[types.Slot.RealPosition()] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[types.Fork.RealPosition()] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := stateutil.BlockHeaderRoot(state.latestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[types.LatestBlockHeader.RealPosition()] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := stateutil.ArraysRoot(state.blockRootsVal().Slice(), fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[types.BlockRoots.RealPosition()] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := stateutil.ArraysRoot(state.stateRootsVal().Slice(), fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[types.StateRoots.RealPosition()] = stateRootsRoot[:]

	// RewardAdjustmentFactor root.
	rewardAdjustmentFactorRoot := ssz.Uint64Root(state.rewardAdjustmentFactor)
	fieldRoots[types.RewardAdjustmentFactor.RealPosition()] = rewardAdjustmentFactorRoot[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := stateutil.Eth1Root(state.eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[types.Eth1Data.RealPosition()] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := stateutil.Eth1DataVotesRoot(state.eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[types.Eth1DataVotes.RealPosition()] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[types.Eth1DepositIndex.RealPosition()] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := stateutil.ValidatorRegistryRoot(state.validatorsVal())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[types.Validators.RealPosition()] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.balancesVal())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[types.Balances.RealPosition()] = balancesRoot[:]

	// Reserves root.
	reservesRoot := ssz.Uint64Root(state.reserves)
	fieldRoots[types.Reserves.RealPosition()] = reservesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := stateutil.ArraysRoot(state.randaoMixesVal().Slice(), fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[types.RandaoMixes.RealPosition()] = randaoRootsRoot[:]

	if state.version == version.Phase0 {
		// PreviousEpochAttestations slice root.
		prevAttsRoot, err := stateutil.EpochAttestationsRoot(state.previousEpochAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
		}
		fieldRoots[types.PreviousEpochAttestations.RealPosition()] = prevAttsRoot[:]

		// CurrentEpochAttestations slice root.
		currAttsRoot, err := stateutil.EpochAttestationsRoot(state.currentEpochAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
		}
		fieldRoots[types.CurrentEpochAttestations.RealPosition()] = currAttsRoot[:]
	}

	if state.version >= version.Altair {
		// PreviousEpochParticipation slice root.
		prevParticipationRoot, err := stateutil.ParticipationBitsRoot(state.previousEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
		}
		fieldRoots[types.PreviousEpochParticipationBits.RealPosition()] = prevParticipationRoot[:]

		// CurrentEpochParticipation slice root.
		currParticipationRoot, err := stateutil.ParticipationBitsRoot(state.currentEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
		}
		fieldRoots[types.CurrentEpochParticipationBits.RealPosition()] = currParticipationRoot[:]
	}

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.justificationBits)
	fieldRoots[types.JustificationBits.RealPosition()] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(state.previousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[types.PreviousJustifiedCheckpoint.RealPosition()] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(state.currentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[types.CurrentJustifiedCheckpoint.RealPosition()] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(state.finalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[types.FinalizedCheckpoint.RealPosition()] = finalRoot[:]

	if state.version >= version.Altair {
		// Inactivity scores root.
		inactivityScoresRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.inactivityScoresVal())
		if err != nil {
			return nil, errors.Wrap(err, "could not compute inactivityScoreRoot")
		}
		fieldRoots[types.InactivityScores.RealPosition()] = inactivityScoresRoot[:]
	}

	if state.version == version.Bellatrix {
		// Execution payload root.
		executionPayloadRoot, err := state.latestExecutionPayloadHeader.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		fieldRoots[types.LatestExecutionPayloadHeader.RealPosition()] = executionPayloadRoot[:]
	}

	if state.version == version.Capella {
		// Execution payload root.
		executionPayloadRoot, err := state.latestExecutionPayloadHeaderCapella.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		fieldRoots[types.LatestExecutionPayloadHeaderCapella.RealPosition()] = executionPayloadRoot[:]
	}

	if state.version >= version.Deneb {
		// Execution payload root.
		executionPayloadRoot, err := state.latestExecutionPayloadHeaderDeneb.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		fieldRoots[types.LatestExecutionPayloadHeaderDeneb.RealPosition()] = executionPayloadRoot[:]
	}

	if state.version >= version.Capella {
		// Next withdrawal index root.
		nextWithdrawalIndexRoot := make([]byte, 32)
		binary.LittleEndian.PutUint64(nextWithdrawalIndexRoot, state.nextWithdrawalIndex)
		fieldRoots[types.NextWithdrawalIndex.RealPosition()] = nextWithdrawalIndexRoot

		// Next partial withdrawal validator index root.
		nextWithdrawalValidatorIndexRoot := make([]byte, 32)
		binary.LittleEndian.PutUint64(nextWithdrawalValidatorIndexRoot, uint64(state.nextWithdrawalValidatorIndex))
		fieldRoots[types.NextWithdrawalValidatorIndex.RealPosition()] = nextWithdrawalValidatorIndexRoot

		// Historical summary root.
		historicalSummaryRoot, err := stateutil.HistoricalSummariesRoot(state.historicalSummaries)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute historical summary merkleization")
		}
		fieldRoots[types.HistoricalSummaries.RealPosition()] = historicalSummaryRoot[:]
	}

	if state.version >= version.Alpaca {
		// DepositRequestsStartIndex root.
		drsiRoot := ssz.Uint64Root(state.depositRequestsStartIndex)
		fieldRoots[types.DepositRequestsStartIndex.RealPosition()] = drsiRoot[:]

		// DepositBalanceToConsume root.
		dbtcRoot := ssz.Uint64Root(uint64(state.depositBalanceToConsume))
		fieldRoots[types.DepositBalanceToConsume.RealPosition()] = dbtcRoot[:]

		// ExitBalanceToConsume root.
		ebtcRoot := ssz.Uint64Root(uint64(state.exitBalanceToConsume))
		fieldRoots[types.ExitBalanceToConsume.RealPosition()] = ebtcRoot[:]

		// EarliestExitEpoch root.
		eeeRoot := ssz.Uint64Root(uint64(state.earliestExitEpoch))
		fieldRoots[types.EarliestExitEpoch.RealPosition()] = eeeRoot[:]

		// PendingDeposits root.
		pbdRoot, err := stateutil.PendingDepositsRoot(state.pendingDeposits)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute pending balance deposits merkleization")
		}
		fieldRoots[types.PendingDeposits.RealPosition()] = pbdRoot[:]

		// PendingPartialWithdrawals root.
		ppwRoot, err := stateutil.PendingPartialWithdrawalsRoot(state.pendingPartialWithdrawals)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute pending partial withdrawals merkleization")
		}
		fieldRoots[types.PendingPartialWithdrawals.RealPosition()] = ppwRoot[:]
	}

	return fieldRoots, nil
}
