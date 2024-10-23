package types

import (
	"fmt"

	"github.com/pkg/errors"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
)

// DataType signifies the data type of the field.
type DataType int

// List of current data types the state supports.
const (
	// BasicArray represents a simple array type for a field.
	BasicArray DataType = iota
	// CompositeArray represents a variable length array with
	// a non primitive type.
	CompositeArray
	// CompressedArray represents a variable length array which
	// can pack multiple elements into a leaf of the underlying
	// trie.
	CompressedArray
)

// FieldIndex represents the relevant field position in the
// state struct for a field.
type FieldIndex int

// String returns the name of the field index.
func (f FieldIndex) String() string {
	switch f {
	case GenesisTime:
		return "genesisTime"
	case GenesisValidatorsRoot:
		return "genesisValidatorsRoot"
	case Slot:
		return "slot"
	case Fork:
		return "fork"
	case LatestBlockHeader:
		return "latestBlockHeader"
	case BlockRoots:
		return "blockRoots"
	case StateRoots:
		return "stateRoots"
	case HistoricalRoots:
		return "historicalRoots"
	case RewardAdjustmentFactor:
		return "rewardAdjustmentFactor"
	case Eth1Data:
		return "eth1Data"
	case Eth1DataVotes:
		return "eth1DataVotes"
	case Eth1DepositIndex:
		return "eth1DepositIndex"
	case Validators:
		return "validators"
	case Balances:
		return "balances"
	case Reserves:
		return "reserves"
	case RandaoMixes:
		return "randaoMixes"
	case PreviousEpochAttestations:
		return "previousEpochAttestations"
	case CurrentEpochAttestations:
		return "currentEpochAttestations"
	case PreviousEpochParticipationBits:
		return "previousEpochParticipationBits"
	case CurrentEpochParticipationBits:
		return "currentEpochParticipationBits"
	case JustificationBits:
		return "justificationBits"
	case PreviousJustifiedCheckpoint:
		return "previousJustifiedCheckpoint"
	case CurrentJustifiedCheckpoint:
		return "currentJustifiedCheckpoint"
	case FinalizedCheckpoint:
		return "finalizedCheckpoint"
	case InactivityScores:
		return "inactivityScores"
	case LatestExecutionPayloadHeader:
		return "latestExecutionPayloadHeader"
	case LatestExecutionPayloadHeaderCapella:
		return "latestExecutionPayloadHeaderCapella"
	case LatestExecutionPayloadHeaderDeneb:
		return "latestExecutionPayloadHeaderDeneb"
	case NextWithdrawalIndex:
		return "nextWithdrawalIndex"
	case NextWithdrawalValidatorIndex:
		return "nextWithdrawalValidatorIndex"
	case HistoricalSummaries:
		return "historicalSummaries"
	case DepositRequestsStartIndex:
		return "depositRequestsStartIndex"
	case DepositBalanceToConsume:
		return "depositBalanceToConsume"
	case ExitBalanceToConsume:
		return "exitBalanceToConsume"
	case EarliestExitEpoch:
		return "earliestExitEpoch"
	case PendingDeposits:
		return "pendingDeposits"
	case PendingPartialWithdrawals:
		return "pendingPartialWithdrawals"
	default:
		return fmt.Sprintf("unknown field index number: %d", f)
	}
}

// RealPosition denotes the position of the field in the beacon state.
// The value might differ for different state versions.
func (f FieldIndex) RealPosition() int {
	switch f {
	case GenesisTime:
		return 0
	case GenesisValidatorsRoot:
		return 1
	case Slot:
		return 2
	case Fork:
		return 3
	case LatestBlockHeader:
		return 4
	case BlockRoots:
		return 5
	case StateRoots:
		return 6
	case HistoricalRoots:
		return 7
	case RewardAdjustmentFactor:
		return 8
	case Eth1Data:
		return 9
	case Eth1DataVotes:
		return 10
	case Eth1DepositIndex:
		return 11
	case Validators:
		return 12
	case Balances:
		return 13
	case Reserves:
		return 14
	case RandaoMixes:
		return 15
	case PreviousEpochAttestations, PreviousEpochParticipationBits:
		return 16
	case CurrentEpochAttestations, CurrentEpochParticipationBits:
		return 17
	case JustificationBits:
		return 18
	case PreviousJustifiedCheckpoint:
		return 19
	case CurrentJustifiedCheckpoint:
		return 20
	case FinalizedCheckpoint:
		return 21
	case InactivityScores:
		return 23
	case LatestExecutionPayloadHeader, LatestExecutionPayloadHeaderCapella, LatestExecutionPayloadHeaderDeneb:
		return 24
	case NextWithdrawalIndex:
		return 25
	case NextWithdrawalValidatorIndex:
		return 26
	case HistoricalSummaries:
		return 27
	case DepositRequestsStartIndex:
		return 28
	case DepositBalanceToConsume:
		return 29
	case ExitBalanceToConsume:
		return 30
	case EarliestExitEpoch:
		return 31
	case PendingDeposits:
		return 32
	case PendingPartialWithdrawals:
		return 33
	default:
		return -1
	}
}

// ElemsInChunk returns the number of elements in the chunk (number of
// elements that are able to be packed).
func (f FieldIndex) ElemsInChunk() (uint64, error) {
	switch f {
	case Balances:
		return 4, nil
	default:
		return 0, errors.Errorf("field %d doesn't support element compression", f)
	}
}

// Below we define a set of useful enum values for the field
// indices of the beacon state. For example, genesisTime is the
// 0th field of the beacon state. This is helpful when we are
// updating the Merkle branches up the trie representation
// of the beacon state. The below field indexes correspond
// to the state.
const (
	GenesisTime FieldIndex = iota
	GenesisValidatorsRoot
	Slot
	Fork
	LatestBlockHeader
	BlockRoots
	StateRoots
	HistoricalRoots
	RewardAdjustmentFactor
	Eth1Data
	Eth1DataVotes
	Eth1DepositIndex
	Validators
	Balances
	Reserves
	RandaoMixes
	PreviousEpochAttestations
	CurrentEpochAttestations
	PreviousEpochParticipationBits
	CurrentEpochParticipationBits
	JustificationBits
	PreviousJustifiedCheckpoint
	CurrentJustifiedCheckpoint
	FinalizedCheckpoint
	InactivityScores
	LatestExecutionPayloadHeader
	LatestExecutionPayloadHeaderCapella
	LatestExecutionPayloadHeaderDeneb
	NextWithdrawalIndex
	NextWithdrawalValidatorIndex
	HistoricalSummaries
	DepositRequestsStartIndex // Electra: EIP-6110
	DepositBalanceToConsume   // Electra: EIP-7251
	ExitBalanceToConsume      // Electra: EIP-7251
	EarliestExitEpoch         // Electra: EIP-7251
	PendingDeposits           // Electra: EIP-7251
	PendingPartialWithdrawals // Electra: EIP-7251
)

// Enumerator keeps track of the number of states created since the node's start.
var Enumerator = &consensus_types.ThreadSafeEnumerator{}
