// Package params defines important constants that are essential to Prysm services.
package params

import (
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// BeaconChainConfig contains constant configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	// Constants (non-configurable)
	GenesisSlot              primitives.Slot  `yaml:"GENESIS_SLOT"`                // GenesisSlot represents the first canonical slot number of the beacon chain.
	GenesisEpoch             primitives.Epoch `yaml:"GENESIS_EPOCH"`               // GenesisEpoch represents the first canonical epoch number of the beacon chain.
	FarFutureEpoch           primitives.Epoch `yaml:"FAR_FUTURE_EPOCH"`            // FarFutureEpoch represents a epoch extremely far away in the future used as the default penalization epoch for validators.
	FarFutureSlot            primitives.Slot  `yaml:"FAR_FUTURE_SLOT"`             // FarFutureSlot represents a slot extremely far away in the future.
	BaseRewardsPerEpoch      uint64           `yaml:"BASE_REWARDS_PER_EPOCH"`      // BaseRewardsPerEpoch is used to calculate the per epoch rewards.
	DepositContractTreeDepth uint64           `yaml:"DEPOSIT_CONTRACT_TREE_DEPTH"` // DepositContractTreeDepth depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.
	JustificationBitsLength  uint64           `yaml:"JUSTIFICATION_BITS_LENGTH"`   // JustificationBitsLength defines number of epochs to track when implementing k-finality in Casper FFG.

	// Misc constants.
	PresetBase                      string     `yaml:"PRESET_BASE" spec:"true"`                        // PresetBase represents the underlying spec preset this config is based on.
	ConfigName                      string     `yaml:"CONFIG_NAME" spec:"true"`                        // ConfigName for allowing an easy human-readable way of knowing what chain is being used.
	TargetCommitteeSize             uint64     `yaml:"TARGET_COMMITTEE_SIZE" spec:"true"`              // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	MaxValidatorsPerCommittee       uint64     `yaml:"MAX_VALIDATORS_PER_COMMITTEE" spec:"true"`       // MaxValidatorsPerCommittee defines the upper bound of the size of a committee.
	MaxCommitteesPerSlot            uint64     `yaml:"MAX_COMMITTEES_PER_SLOT" spec:"true"`            // MaxCommitteesPerSlot defines the max amount of committee in a single slot.
	MinPerEpochChurnLimit           uint64     `yaml:"MIN_PER_EPOCH_CHURN_LIMIT" spec:"true"`          // MinPerEpochChurnLimit is the minimum amount of churn allotted for validator rotations.
	ChurnLimitQuotient              uint64     `yaml:"CHURN_LIMIT_QUOTIENT" spec:"true"`               // ChurnLimitQuotient is used to determine the limit of how many validators can rotate per epoch.
	ChurnLimitBias                  uint64     `yaml:"CHURN_LIMIT_BIAS" spec:"true"`                   // ChurnLimitBias is a parameter for dynamic churn limit calculation.
	ShuffleRoundCount               uint64     `yaml:"SHUFFLE_ROUND_COUNT" spec:"true"`                // ShuffleRoundCount is used for retrieving the permuted index.
	MinGenesisActiveValidatorCount  uint64     `yaml:"MIN_GENESIS_ACTIVE_VALIDATOR_COUNT" spec:"true"` // MinGenesisActiveValidatorCount defines how many validator deposits needed to kick off beacon chain.
	MinGenesisTime                  uint64     `yaml:"MIN_GENESIS_TIME" spec:"true"`                   // MinGenesisTime is the time that needed to pass before kicking off beacon chain.
	TargetAggregatorsPerCommittee   uint64     `yaml:"TARGET_AGGREGATORS_PER_COMMITTEE" spec:"true"`   // TargetAggregatorsPerCommittee defines the number of aggregators inside one committee.
	HysteresisQuotient              uint64     `yaml:"HYSTERESIS_QUOTIENT" spec:"true"`                // HysteresisQuotient defines the hysteresis quotient for effective balance calculations.
	HysteresisDownwardMultiplier    uint64     `yaml:"HYSTERESIS_DOWNWARD_MULTIPLIER" spec:"true"`     // HysteresisDownwardMultiplier defines the hysteresis downward multiplier for effective balance calculations.
	HysteresisUpwardMultiplier      uint64     `yaml:"HYSTERESIS_UPWARD_MULTIPLIER" spec:"true"`       // HysteresisUpwardMultiplier defines the hysteresis upward multiplier for effective balance calculations.
	IssuanceRate                    [11]uint64 `yaml:"ISSUANCE_RATE" spec:"true"`                      // IssuanceRate defines the issuance rate for the beacon chain.
	IssuancePrecision               uint64     `yaml:"ISSUANCE_PRECISION" spec:"true"`                 // IssuancePrecision defines the precision of the issuance rate.
	DepositPlanEarlyEnd             uint64     `yaml:"DEPOSIT_PLAN_EARLY_END" spec:"true"`             // DepositPlanEarlyEnd defines the first end of the ~x year deposit plan.
	DepositPlanEarlySlope           uint64     `yaml:"DEPOSIT_PLAN_EARLY_SLOPE" spec:"true"`           // DepositPlanEarlySlope defines the slope of the ~x year deposit plan.
	DepositPlanEarlyOffset          uint64     `yaml:"DEPOSIT_PLAN_EARLY_OFFSET" spec:"true"`          // DepositPlanEarlyOffset defines the bias of the ~x year deposit plan.
	DepositPlanLaterEnd             uint64     `yaml:"DEPOSIT_PLAN_LATER_END" spec:"true"`             // DepositPlanLaterEnd defines the last end of the ~y year deposit plan.
	DepositPlanLaterSlope           uint64     `yaml:"DEPOSIT_PLAN_LATER_SLOPE" spec:"true"`           // DepositPlanLaterSlope defines the slope of the x~y year deposit plan.
	DepositPlanLaterOffset          uint64     `yaml:"DEPOSIT_PLAN_LATER_OFFSET" spec:"true"`          // DepositPlanLaterOffset defines the bias of the x~y year deposit plan.
	DepositPlanFinal                uint64     `yaml:"DEPOSIT_PLAN_FINAL" spec:"true"`                 // DepositPlanFinal defines the final deposit amount after 6 years.
	RewardAdjustmentFactorDelta     uint64     `yaml:"REWARD_ADJUSTMENT_FACTOR_DELTA" spec:"true"`     // RewardAdjustmentFactorDelta defines the delta for reward adjustment factor.
	RewardAdjustmentFactorPrecision uint64     `yaml:"REWARD_ADJUSTMENT_FACTOR_PRECISION" spec:"true"` // RewardAdjustmentFactorPrecision defines the precision of the reward feedback.
	MaxRewardAdjustmentFactors      [11]uint64 `yaml:"MAX_REWARD_ADJUSTMENT_FACTORS" spec:"true"`      // MaxRewardAdjustmentFactors defines the maximum value(1%) for the reward feedback.

	// Gwei value constants.
	MinDepositAmount          uint64 `yaml:"MIN_DEPOSIT_AMOUNT" spec:"true"`          // MinDepositAmount is the minimum amount of Gwei a validator can send to the deposit contract at once (lower amounts will be reverted).
	MinActivationBalance      uint64 `yaml:"MIN_ACTIVATION_BALANCE" spec:"true"`      // MinActivationBalance is the minimum amount of Gwei a validator must have to be activated in the beacon state.
	MaxEffectiveBalance       uint64 `yaml:"MAX_EFFECTIVE_BALANCE" spec:"true"`       // MaxEffectiveBalance is the maximal amount of Gwei that is effective for staking.
	EffectiveBalanceIncrement uint64 `yaml:"EFFECTIVE_BALANCE_INCREMENT" spec:"true"` // EffectiveBalanceIncrement is used for converting the high balance into the low balance for validators.
	MaxTokenSupply            uint64 `yaml:"MAX_TOKEN_SUPPLY" spec:"true"`            // MaxTokenSupply defines the target maximum number of tokens that can be issued in GWei.
	IssuancePerYear           uint64 `yaml:"ISSUANCE_PER_YEAR" spec:"true"`           // IssuancePerYear defines the issuance of tokens per year in GWei.

	// Initial value constants.
	BLSWithdrawalPrefixByte         byte     `yaml:"BLS_WITHDRAWAL_PREFIX" spec:"true"`          // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.
	ETH1AddressWithdrawalPrefixByte byte     `yaml:"ETH1_ADDRESS_WITHDRAWAL_PREFIX" spec:"true"` // ETH1AddressWithdrawalPrefixByte is used for withdrawals and it's the first byte.
	CompoundingWithdrawalPrefixByte byte     `yaml:"COMPOUNDING_WITHDRAWAL_PREFIX" spec:"true"`  // CompoundingWithdrawalPrefixByteByte is used for compounding withdrawals and it's the first byte.
	ZeroHash                        [32]byte // ZeroHash is used to represent a zeroed out 32 byte array.

	// Time parameters constants.
	GenesisDelay                     uint64           `yaml:"GENESIS_DELAY" spec:"true"`                   // GenesisDelay is the minimum number of seconds to delay starting the Ethereum Beacon Chain genesis. Must be at least 1 second.
	MinAttestationInclusionDelay     primitives.Slot  `yaml:"MIN_ATTESTATION_INCLUSION_DELAY" spec:"true"` // MinAttestationInclusionDelay defines how many slots validator has to wait to include attestation for beacon block.
	SecondsPerSlot                   uint64           `yaml:"SECONDS_PER_SLOT" spec:"true"`                // SecondsPerSlot is how many seconds are in a single slot.
	SlotsPerEpoch                    primitives.Slot  `yaml:"SLOTS_PER_EPOCH" spec:"true"`                 // SlotsPerEpoch is the number of slots in an epoch.
	SqrRootSlotsPerEpoch             primitives.Slot  // SqrRootSlotsPerEpoch is a hard coded value where we take the square root of `SlotsPerEpoch` and round down.
	EpochsPerYear                    uint64           `yaml:"EPOCHS_PER_YEAR" spec:"true"`                     // EpochsPerYear defines the number of epochs per year.
	MinSeedLookahead                 primitives.Epoch `yaml:"MIN_SEED_LOOKAHEAD" spec:"true"`                  // MinSeedLookahead is the duration of randao look ahead seed.
	MaxSeedLookahead                 primitives.Epoch `yaml:"MAX_SEED_LOOKAHEAD" spec:"true"`                  // MaxSeedLookahead is the duration a validator has to wait for entry and exit in epoch.
	EpochsPerEth1VotingPeriod        primitives.Epoch `yaml:"EPOCHS_PER_ETH1_VOTING_PERIOD" spec:"true"`       // EpochsPerEth1VotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node on per epoch basis.
	SlotsPerHistoricalRoot           primitives.Slot  `yaml:"SLOTS_PER_HISTORICAL_ROOT" spec:"true"`           // SlotsPerHistoricalRoot defines how often the historical root is saved.
	MinValidatorWithdrawabilityDelay primitives.Epoch `yaml:"MIN_VALIDATOR_WITHDRAWABILITY_DELAY" spec:"true"` // MinValidatorWithdrawabilityDelay is the shortest amount of time a validator has to wait to withdraw.
	ShardCommitteePeriod             primitives.Epoch `yaml:"SHARD_COMMITTEE_PERIOD" spec:"true"`              // ShardCommitteePeriod is the minimum amount of epochs a validator must participate before exiting.
	MinEpochsToInactivityPenalty     primitives.Epoch `yaml:"MIN_EPOCHS_TO_INACTIVITY_PENALTY" spec:"true"`    // MinEpochsToInactivityPenalty defines the minimum amount of epochs since finality to begin penalizing inactivity.
	Eth1FollowDistance               uint64           `yaml:"ETH1_FOLLOW_DISTANCE" spec:"true"`                // Eth1FollowDistance is the number of eth1.0 blocks to wait before considering a new deposit for voting. This only applies after the chain as been started.
	SecondsPerETH1Block              uint64           `yaml:"SECONDS_PER_ETH1_BLOCK" spec:"true"`              // SecondsPerETH1Block is the approximate time for a single eth1 block to be produced.

	// Fork choice algorithm constants.
	ProposerScoreBoost              uint64           `yaml:"PROPOSER_SCORE_BOOST" spec:"true"`                // ProposerScoreBoost defines a value that is a % of the committee weight for fork-choice boosting.
	ReorgWeightThreshold            uint64           `yaml:"REORG_WEIGHT_THRESHOLD" spec:"true"`              // ReorgWeightThreshold defines a value that is a % of the committee weight to consider a block weak and subject to being orphaned.
	ReorgParentWeightThreshold      uint64           `yaml:"REORG_PARENT_WEIGHT_THRESHOLD" spec:"true"`       // ReorgParentWeightThreshold defines a value that is a % of the committee weight to consider a parent block strong and subject its child to being orphaned.
	ReorgMaxEpochsSinceFinalization primitives.Epoch `yaml:"REORG_MAX_EPOCHS_SINCE_FINALIZATION" spec:"true"` // This defines a limit to consider safe to orphan a blockF if the network is finalizing
	IntervalsPerSlot                uint64           `yaml:"INTERVALS_PER_SLOT" spec:"true"`                  // IntervalsPerSlot defines the number of fork choice intervals in a slot defined in the fork choice spec.

	// Ethereum PoW parameters.
	DepositChainID         uint64 `yaml:"DEPOSIT_CHAIN_ID" spec:"true"`         // DepositChainID of the eth1 network. This used for replay protection.
	DepositNetworkID       uint64 `yaml:"DEPOSIT_NETWORK_ID" spec:"true"`       // DepositNetworkID of the eth1 network. This used for replay protection.
	DepositContractAddress string `yaml:"DEPOSIT_CONTRACT_ADDRESS" spec:"true"` // DepositContractAddress is the address of the deposit contract.

	// Validator parameters.
	RandomSubnetsPerValidator         uint64           `yaml:"RANDOM_SUBNETS_PER_VALIDATOR" spec:"true"`          // RandomSubnetsPerValidator specifies the amount of subnets a validator has to be subscribed to at one time.
	EpochsPerRandomSubnetSubscription uint64           `yaml:"EPOCHS_PER_RANDOM_SUBNET_SUBSCRIPTION" spec:"true"` // EpochsPerRandomSubnetSubscription specifies the minimum duration a validator is connected to their subnet.
	MinSlashingWithdrawableDelay      primitives.Epoch `yaml:"MIN_SLASHING_WITHDRAWABLE_DELAY" spec:"true"`       // MinSlashingWithdrawableDelay defines the minimum amount of epochs a validator has to wait to withdraw after being slashed.

	// State list lengths
	EpochsPerHistoricalVector primitives.Epoch `yaml:"EPOCHS_PER_HISTORICAL_VECTOR" spec:"true"` // EpochsPerHistoricalVector defines max length in epoch to store old historical stats in beacon state.
	HistoricalRootsLimit      uint64           `yaml:"HISTORICAL_ROOTS_LIMIT" spec:"true"`       // HistoricalRootsLimit defines max historical roots that can be saved in state before roll over.
	ValidatorRegistryLimit    uint64           `yaml:"VALIDATOR_REGISTRY_LIMIT" spec:"true"`     // ValidatorRegistryLimit defines the upper bound of validators can participate in eth2.

	// Reward and penalty quotients constants.
	WhistleBlowerRewardQuotient          uint64 `yaml:"WHISTLEBLOWER_REWARD_QUOTIENT" spec:"true"`            // WhistleBlowerRewardQuotient is used to calculate whistle blower reward.
	ProposerRewardQuotient               uint64 `yaml:"PROPOSER_REWARD_QUOTIENT" spec:"true"`                 // ProposerRewardQuotient is used to calculate the reward for proposers.
	MinSlashingPenaltyQuotient           uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT" spec:"true"`            // MinSlashingPenaltyQuotient is used to calculate the minimum penalty to prevent DoS attacks.
	InactivityPenaltyRate                uint64 `yaml:"INACTIVITY_PENALTY_RATE" spec:"true"`                  // InactivityPenaltyRate is used to calculate the penalty numerator and bail out.
	InactivityPenaltyRatePrecision       uint64 `yaml:"INACTIVITY_PENALTY_RATE_PRECISION" spec:"true"`        // InactivityPenaltyRatePrecision is used to calculate the penalty numerator and bail out.
	InactivityPenaltyDuration            uint64 `yaml:"INACTIVITY_PENALTY_DURATION" spec:"true"`              // InactivityPenaltyDuration defines the maximum duration a validator can be offline before bail out.
	InactivityScorePenaltyThreshold      uint64 `yaml:"INACTIVITY_SCORE_PENALTY_THRESHOLD" spec:"true"`       // InactivityScorePenaltyThreshold defines the threshold for inactivity accumulated for one day.
	InactivityLeakPenaltyBuffer          uint64 `yaml:"INACTIVITY_LEAK_PENALTY_BUFFER" spec:"true"`           // InactivityLeakPenaltyBuffer is used to calculate the penalty buffer.
	InactivityLeakPenaltyBufferPrecision uint64 `yaml:"INACTIVITY_LEAK_PENALTY_BUFFER_PRECISION" spec:"true"` // InactivityLeakPenaltyBufferPrecision is used to calculate the penalty buffer.
	InactivityLeakBailoutScoreThreshold  uint64 `yaml:"INACTIVITY_LEAK_BAILOUT_SCORE_THRESHOLD" spec:"true"`  // InactivityLeakBailoutScoreThreshold defines the inactivity score accumulated for 15 days.

	// Max operations per block constants.
	MaxProposerSlashings             uint64 `yaml:"MAX_PROPOSER_SLASHINGS" spec:"true"`        // MaxProposerSlashings defines the maximum number of slashings of proposers possible in a block.
	MaxAttesterSlashings             uint64 `yaml:"MAX_ATTESTER_SLASHINGS" spec:"true"`        // MaxAttesterSlashings defines the maximum number of casper FFG slashings possible in a block.
	MaxAttesterSlashingsAlpaca       uint64 `yaml:"MAX_ATTESTER_SLASHINGS_ALPACA" spec:"true"` // MaxAttesterSlashingsAlpaca defines the maximum number of casper FFG slashings possible in a block post Alpaca hard fork.
	MaxAttestations                  uint64 `yaml:"MAX_ATTESTATIONS" spec:"true"`              // MaxAttestations defines the maximum allowed attestations in a beacon block.
	MaxAttestationsAlpaca            uint64 `yaml:"MAX_ATTESTATIONS_ALPACA" spec:"true"`       // MaxAttestationsAlpaca defines the maximum allowed attestations in a beacon block post Alpaca hard fork.
	MaxDeposits                      uint64 `yaml:"MAX_DEPOSITS" spec:"true"`                  // MaxDeposits defines the maximum number of validator deposits in a block.
	MaxDepositsAlpaca                uint64 `yaml:"MAX_DEPOSITS_ALPACA" spec:"true"`           // MaxDepositsAlpaca defines the maximum number of validator deposits in a block.
	MaxVoluntaryExits                uint64 `yaml:"MAX_VOLUNTARY_EXITS" spec:"true"`           // MaxVoluntaryExits defines the maximum number of validator exits in a block.
	MaxWithdrawalsPerPayload         uint64 `yaml:"MAX_WITHDRAWALS_PER_PAYLOAD" spec:"true"`   // MaxWithdrawalsPerPayload defines the maximum number of withdrawals in a block.
	MaxPartialWithdrawalsPerPayload  uint64 `yaml:"MAX_PARTIAL_WITHDRAWALS_PER_PAYLOAD" spec:"true"`
	MaxValidatorsPerWithdrawalsSweep uint64 `yaml:"MAX_VALIDATORS_PER_WITHDRAWALS_SWEEP" spec:"true"` // MaxValidatorsPerWithdrawalsSweep bounds the size of the sweep searching for withdrawals per slot.

	// BLS domain values.
	DomainBeaconProposer     [4]byte `yaml:"DOMAIN_BEACON_PROPOSER" spec:"true"`     // DomainBeaconProposer defines the BLS signature domain for beacon proposal verification.
	DomainRandao             [4]byte `yaml:"DOMAIN_RANDAO" spec:"true"`              // DomainRandao defines the BLS signature domain for randao verification.
	DomainBeaconAttester     [4]byte `yaml:"DOMAIN_BEACON_ATTESTER" spec:"true"`     // DomainBeaconAttester defines the BLS signature domain for attestation verification.
	DomainDeposit            [4]byte `yaml:"DOMAIN_DEPOSIT" spec:"true"`             // DomainDeposit defines the BLS signature domain for deposit verification.
	DomainVoluntaryExit      [4]byte `yaml:"DOMAIN_VOLUNTARY_EXIT" spec:"true"`      // DomainVoluntaryExit defines the BLS signature domain for exit verification.
	DomainSelectionProof     [4]byte `yaml:"DOMAIN_SELECTION_PROOF" spec:"true"`     // DomainSelectionProof defines the BLS signature domain for selection proof.
	DomainAggregateAndProof  [4]byte `yaml:"DOMAIN_AGGREGATE_AND_PROOF" spec:"true"` // DomainAggregateAndProof defines the BLS signature domain for aggregate and proof.
	DomainApplicationMask    [4]byte `yaml:"DOMAIN_APPLICATION_MASK" spec:"true"`    // DomainApplicationMask defines the BLS signature domain for application mask.
	DomainApplicationBuilder [4]byte `yaml:"DOMAIN_APPLICATION_BUILDER" spec:"true"` // DomainApplicationBuilder defines the BLS signature domain for application builder.

	// Prysm constants.
	GenesisValidatorsRoot          [32]byte        // GenesisValidatorsRoot is the root hash of the genesis validators.
	GweiPerEth                     uint64          // GweiPerEth is the amount of gwei corresponding to 1 eth.
	BLSSecretKeyLength             int             // BLSSecretKeyLength defines the expected length of BLS secret keys in bytes.
	BLSPubkeyLength                int             // BLSPubkeyLength defines the expected length of BLS public keys in bytes.
	DefaultBufferSize              int             // DefaultBufferSize for channels across the Prysm repository.
	ValidatorPrivkeyFileName       string          // ValidatorPrivKeyFileName specifies the string name of a validator private key file.
	WithdrawalPrivkeyFileName      string          // WithdrawalPrivKeyFileName specifies the string name of a withdrawal private key file.
	RPCSyncCheck                   time.Duration   // Number of seconds to query the sync service, to find out if the node is synced or not.
	EmptySignature                 [96]byte        // EmptySignature is used to represent a zeroed out BLS Signature.
	DefaultPageSize                int             // DefaultPageSize defines the default page size for RPC server request.
	MaxPeersToSync                 int             // MaxPeersToSync describes the limit for number of peers in round robin sync.
	SlotsPerArchivedPoint          primitives.Slot // SlotsPerArchivedPoint defines the number of slots per one archived point.
	GenesisCountdownInterval       time.Duration   // How often to log the countdown until the genesis time is reached.
	BeaconStateFieldCount          int             // BeaconStateFieldCount defines how many fields are in the Phase0 beacon state.
	BeaconStateAltairFieldCount    int             // BeaconStateAltairFieldCount defines how many fields are in the beacon state post upgrade to Altair.
	BeaconStateBellatrixFieldCount int             // BeaconStateBellatrixFieldCount defines how many fields are in beacon state post upgrade to Bellatrix.
	BeaconStateCapellaFieldCount   int             // BeaconStateCapellaFieldCount defines how many fields are in beacon state post upgrade to Capella.
	BeaconStateDenebFieldCount     int             // BeaconStateDenebFieldCount defines how many fields are in beacon state post upgrade to Deneb.
	BeaconStateAlpacaFieldCount    int             // BeaconStateAlpacaFieldCount defines how many fields are in beacon state post upgrade to Alpaca.
	BeaconStateBadgerFieldCount    int             // BeaconStateBadgerFieldCount defines how many fields are in beacon state post upgrade to Badger.

	// Slasher constants.
	WeakSubjectivityPeriod    primitives.Epoch // WeakSubjectivityPeriod defines the time period expressed in number of epochs were proof of stake network should validate block headers and attestations for slashable events.
	PruneSlasherStoragePeriod primitives.Epoch // PruneSlasherStoragePeriod defines the time period expressed in number of epochs were proof of stake network should prune attestation and block header store.

	// Slashing protection constants.
	SlashingProtectionPruningEpochs primitives.Epoch // SlashingProtectionPruningEpochs defines a period after which all prior epochs are pruned in the validator database.

	// Fork-related values.
	GenesisForkVersion   []byte           `yaml:"GENESIS_FORK_VERSION" spec:"true"`   // GenesisForkVersion is used to track fork version between state transitions.
	AltairForkVersion    []byte           `yaml:"ALTAIR_FORK_VERSION" spec:"true"`    // AltairForkVersion is used to represent the fork version for altair.
	AltairForkEpoch      primitives.Epoch `yaml:"ALTAIR_FORK_EPOCH" spec:"true"`      // AltairForkEpoch is used to represent the assigned fork epoch for altair.
	BellatrixForkVersion []byte           `yaml:"BELLATRIX_FORK_VERSION" spec:"true"` // BellatrixForkVersion is used to represent the fork version for bellatrix.
	BellatrixForkEpoch   primitives.Epoch `yaml:"BELLATRIX_FORK_EPOCH" spec:"true"`   // BellatrixForkEpoch is used to represent the assigned fork epoch for bellatrix.
	CapellaForkVersion   []byte           `yaml:"CAPELLA_FORK_VERSION" spec:"true"`   // CapellaForkVersion is used to represent the fork version for capella.
	CapellaForkEpoch     primitives.Epoch `yaml:"CAPELLA_FORK_EPOCH" spec:"true"`     // CapellaForkEpoch is used to represent the assigned fork epoch for capella.
	DenebForkVersion     []byte           `yaml:"DENEB_FORK_VERSION" spec:"true"`     // DenebForkVersion is used to represent the fork version for deneb.
	DenebForkEpoch       primitives.Epoch `yaml:"DENEB_FORK_EPOCH" spec:"true"`       // DenebForkEpoch is used to represent the assigned fork epoch for deneb.
	AlpacaForkVersion    []byte           `yaml:"ALPACA_FORK_VERSION" spec:"true"`    // AlpacaForkVersion is used to represent the fork version for alpaca.
	AlpacaForkEpoch      primitives.Epoch `yaml:"ALPACA_FORK_EPOCH" spec:"true"`      // AlpacaForkEpoch is used to represent the assigned fork epoch for alpaca.
	BadgerForkVersion    []byte           `yaml:"BADGER_FORK_VERSION" spec:"true"`    // BadgerForkVersion is used to represent the fork version for badger.
	BadgerForkEpoch      primitives.Epoch `yaml:"BADGER_FORK_EPOCH" spec:"true"`      // BadgerForkEpoch is used to represent the assigned fork epoch for badger.

	ForkVersionSchedule map[[fieldparams.VersionLength]byte]primitives.Epoch // Schedule of fork epochs by version.
	ForkVersionNames    map[[fieldparams.VersionLength]byte]string           // Human-readable names of fork versions.

	// Weak subjectivity values.
	SafetyDecay uint64 // SafetyDecay is defined as the loss in the 1/3 consensus safety margin of the casper FFG mechanism.

	// New values introduced in Altair hard fork 1.
	// Participation flag indices.
	TimelySourceFlagIndex uint8 `yaml:"TIMELY_SOURCE_FLAG_INDEX" spec:"true"` // TimelySourceFlagIndex is the source flag position of the participation bits.
	TimelyTargetFlagIndex uint8 `yaml:"TIMELY_TARGET_FLAG_INDEX" spec:"true"` // TimelyTargetFlagIndex is the target flag position of the participation bits.
	TimelyHeadFlagIndex   uint8 `yaml:"TIMELY_HEAD_FLAG_INDEX" spec:"true"`   // TimelyHeadFlagIndex is the head flag position of the participation bits.

	// Incentivization weights.
	TimelySourceWeight uint64 `yaml:"TIMELY_SOURCE_WEIGHT" spec:"true"` // TimelySourceWeight is the factor of how much source rewards receives.
	TimelyTargetWeight uint64 `yaml:"TIMELY_TARGET_WEIGHT" spec:"true"` // TimelyTargetWeight is the factor of how much target rewards receives.
	TimelyHeadWeight   uint64 `yaml:"TIMELY_HEAD_WEIGHT" spec:"true"`   // TimelyHeadWeight is the factor of how much head rewards receives.
	ProposerWeight     uint64 `yaml:"PROPOSER_WEIGHT" spec:"true"`      // ProposerWeight is the factor of how much proposer rewards receives.
	LightLayerWeight   uint64 `yaml:"LIGHT_LAYER_WEIGHT" spec:"true"`   // LightLayerWeight is the factor of how much light layer rewards receives.
	WeightDenominator  uint64 `yaml:"WEIGHT_DENOMINATOR" spec:"true"`   // WeightDenominator accounts for total rewards denomination.

	// Misc.
	InactivityScoreBias         uint64 `yaml:"INACTIVITY_SCORE_BIAS" spec:"true"`          // InactivityScoreBias for calculating score bias penalties during inactivity
	InactivityScoreRecoveryRate uint64 `yaml:"INACTIVITY_SCORE_RECOVERY_RATE" spec:"true"` // InactivityScoreRecoveryRate for recovering score bias penalties during inactivity.

	// Updated penalty values. This moves penalty parameters toward their final, maximum security values.
	// Note: We do not override previous configuration values but instead creates new values and replaces usage throughout.
	MinSlashingPenaltyQuotientAltair    uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT_ALTAIR" spec:"true"`    // MinSlashingPenaltyQuotientAltair for slashing penalties post Altair hard fork.
	MinSlashingPenaltyQuotientBellatrix uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT_BELLATRIX" spec:"true"` // MinSlashingPenaltyQuotientBellatrix for slashing penalties post Bellatrix hard fork.

	// Bellatrix
	TerminalBlockHash                common.Hash      `yaml:"TERMINAL_BLOCK_HASH" spec:"true"`                  // TerminalBlockHash of beacon chain.
	TerminalBlockHashActivationEpoch primitives.Epoch `yaml:"TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH" spec:"true"` // TerminalBlockHashActivationEpoch of beacon chain.
	TerminalTotalDifficulty          string           `yaml:"TERMINAL_TOTAL_DIFFICULTY" spec:"true"`            // TerminalTotalDifficulty is part of the experimental Bellatrix spec. This value is type is currently TBD.
	DefaultFeeRecipient              common.Address   // DefaultFeeRecipient where the transaction fee goes to.
	EthBurnAddressHex                string           // EthBurnAddressHex is the constant eth address written in hex format to burn fees in that network. the default is 0x0
	DefaultBuilderGasLimit           uint64           // DefaultBuilderGasLimit is the default used to set the gaslimit for the Builder APIs, typically at around 30M wei.

	// Mev-boost circuit breaker
	MaxBuilderConsecutiveMissedSlots primitives.Slot // MaxBuilderConsecutiveMissedSlots defines the number of consecutive skip slot to fallback from using relay/builder to local execution engine for block construction.
	MaxBuilderEpochMissedSlots       primitives.Slot // MaxBuilderEpochMissedSlots is defining the number of total skip slot (per epoch rolling windows) to fallback from using relay/builder to local execution engine for block construction.
	LocalBlockValueBoost             uint64          // LocalBlockValueBoost is the value boost for local block construction. This is used to prioritize local block construction over relay/builder block construction.
	MinBuilderBid                    uint64          // MinBuilderBid is the minimum value that the builder's block can have to be considered by this node.
	MinBuilderDiff                   uint64          // MinBuilderDiff is the minimum value above the local block value that the builder has to bid to be considered by this node
	// Execution engine timeout value
	ExecutionEngineTimeoutValue uint64 // ExecutionEngineTimeoutValue defines the seconds to wait before timing out engine endpoints with execution payload execution semantics (newPayload, forkchoiceUpdated).

	// Subnet value
	BlobsidecarSubnetCount uint64 `yaml:"BLOB_SIDECAR_SUBNET_COUNT"` // BlobsidecarSubnetCount is the number of blobsidecar subnets used in the gossipsub protocol.

	// Values introduced in Deneb hard fork
	MinEpochsForBlobsSidecarsRequest primitives.Epoch `yaml:"MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS" spec:"true"` // MinEpochsForBlobsSidecarsRequest is the minimum number of epochs the node will keep the blobs for.
	MaxRequestBlobSidecars           uint64           `yaml:"MAX_REQUEST_BLOB_SIDECARS" spec:"true"`             // MaxRequestBlobSidecars is the maximum number of blobs to request in a single request.
	MaxRequestBlocksDeneb            uint64           `yaml:"MAX_REQUEST_BLOCKS_DENEB" spec:"true"`              // MaxRequestBlocksDeneb is the maximum number of blocks in a single request after the deneb epoch.

	// Values introduce in Alpaca upgrade
	DataColumnSidecarSubnetCount           uint64 `yaml:"DATA_COLUMN_SIDECAR_SUBNET_COUNT" spec:"true"`             // DataColumnSidecarSubnetCount is the number of data column sidecar subnets used in the gossipsub protocol
	MinPerEpochActivationBalanceChurnLimit uint64 `yaml:"MIN_PER_EPOCH_ACTIVATION_BALANCE_CHURN_LIMIT" spec:"true"` // MinPerEpochActivationBalanceChurnLimit represents the minimum balance of activation churn.
	MinPerEpochChurnLimitAlpaca            uint64 `yaml:"MIN_PER_EPOCH_CHURN_LIMIT_ALPACA" spec:"true"`             // MinPerEpochChurnLimitAlpaca is the minimum amount of churn allotted for validator rotations for alpaca.
	MaxRequestDataColumnSidecars           uint64 `yaml:"MAX_REQUEST_DATA_COLUMN_SIDECARS" spec:"true"`             // MaxRequestDataColumnSidecars is the maximum number of data column sidecars in a single request
	MaxEffectiveBalanceAlpaca              uint64 `yaml:"MAX_EFFECTIVE_BALANCE_ALPACA" spec:"true"`                 // MaxEffectiveBalanceAlpaca is the maximal amount of Gwei that is effective for staking, increased in alpaca.
	PendingDepositLimit                    uint64 `yaml:"PENDING_DEPOSITS_LIMIT" spec:"true"`                       // PendingDepositLimit is the maximum number of pending balance deposits allowed in the beacon state.
	PendingPartialWithdrawalsLimit         uint64 `yaml:"PENDING_PARTIAL_WITHDRAWALS_LIMIT" spec:"true"`            // PendingPartialWithdrawalsLimit is the maximum number of pending partial withdrawals allowed in the beacon state.
	MaxPendingPartialsPerWithdrawalsSweep  uint64 `yaml:"MAX_PENDING_PARTIALS_PER_WITHDRAWALS_SWEEP" spec:"true"`   // MaxPendingPartialsPerWithdrawalsSweep is the maximum number of pending partial withdrawals to process per payload.
	MaxPendingDepositsPerEpoch             uint64 `yaml:"MAX_PENDING_DEPOSITS_PER_EPOCH" spec:"true"`               // MaxPendingDepositsPerEpoch is the maximum number of pending deposits per epoch processing.
	FullExitRequestAmount                  uint64 `yaml:"FULL_EXIT_REQUEST_AMOUNT" spec:"true"`                     // FullExitRequestAmount is the amount of Gwei required to request a full exit.
	MaxWithdrawalRequestsPerPayload        uint64 `yaml:"MAX_WITHDRAWAL_REQUESTS_PER_PAYLOAD" spec:"true"`          // MaxWithdrawalRequestsPerPayload is the maximum number of execution layer withdrawal requests in each payload.
	MaxDepositRequestsPerPayload           uint64 `yaml:"MAX_DEPOSIT_REQUESTS_PER_PAYLOAD" spec:"true"`             // MaxDepositRequestsPerPayload is the maximum number of execution layer deposits in each payload
	UnsetDepositRequestsStartIndex         uint64 `yaml:"UNSET_DEPOSIT_REQUESTS_START_INDEX" spec:"true"`           // UnsetDepositRequestsStartIndex is used to check the start index for eip6110

	// Values introduce in Alpaca upgrade
	MinSlashingPenaltyQuotientAlpaca  uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT_ALPACA" spec:"true"` // MinSlashingPenaltyQuotientAlpaca is used to calculate the minimum penalty to prevent DoS attacks, modified for alpaca.
	WhistleBlowerRewardQuotientAlpaca uint64 `yaml:"WHISTLEBLOWER_REWARD_QUOTIENT_ALPACA" spec:"true"` // WhistleBlowerRewardQuotientAlpaca is used to calculate whistle blower reward, modified in alpaca.

	// Networking Specific Parameters
	GossipMaxSize                   uint64          `yaml:"GOSSIP_MAX_SIZE" spec:"true"`                    // GossipMaxSize is the maximum allowed size of uncompressed gossip messages.
	MaxChunkSize                    uint64          `yaml:"MAX_CHUNK_SIZE" spec:"true"`                     // MaxChunkSize is the maximum allowed size of uncompressed req/resp chunked responses.
	AttestationSubnetCount          uint64          `yaml:"ATTESTATION_SUBNET_COUNT" spec:"true"`           // AttestationSubnetCount is the number of attestation subnets used in the gossipsub protocol.
	AttestationPropagationSlotRange primitives.Slot `yaml:"ATTESTATION_PROPAGATION_SLOT_RANGE" spec:"true"` // AttestationPropagationSlotRange is the maximum number of slots during which an attestation can be propagated.
	MaxRequestBlocks                uint64          `yaml:"MAX_REQUEST_BLOCKS" spec:"true"`                 // MaxRequestBlocks is the maximum number of blocks in a single request.
	TtfbTimeout                     uint64          `yaml:"TTFB_TIMEOUT" spec:"true"`                       // TtfbTimeout is the maximum time to wait for first byte of request response (time-to-first-byte).
	RespTimeout                     uint64          `yaml:"RESP_TIMEOUT" spec:"true"`                       // RespTimeout is the maximum time for complete response transfer.
	MaximumGossipClockDisparity     uint64          `yaml:"MAXIMUM_GOSSIP_CLOCK_DISPARITY" spec:"true"`     // MaximumGossipClockDisparity is the maximum milliseconds of clock disparity assumed between honest nodes.
	MessageDomainInvalidSnappy      [4]byte         `yaml:"MESSAGE_DOMAIN_INVALID_SNAPPY" spec:"true"`      // MessageDomainInvalidSnappy is the 4-byte domain for gossip message-id isolation of invalid snappy messages.
	MessageDomainValidSnappy        [4]byte         `yaml:"MESSAGE_DOMAIN_VALID_SNAPPY" spec:"true"`        // MessageDomainValidSnappy is the 4-byte domain for gossip message-id isolation of valid snappy messages.
	MinEpochsForBlockRequests       uint64          `yaml:"MIN_EPOCHS_FOR_BLOCK_REQUESTS" spec:"true"`      // MinEpochsForBlockRequests represents the minimum number of epochs for which we can serve block requests.
	EpochsPerSubnetSubscription     uint64          `yaml:"EPOCHS_PER_SUBNET_SUBSCRIPTION" spec:"true"`     // EpochsPerSubnetSubscription specifies the minimum duration a validator is connected to their subnet.
	AttestationSubnetExtraBits      uint64          `yaml:"ATTESTATION_SUBNET_EXTRA_BITS" spec:"true"`      // AttestationSubnetExtraBits is the number of extra bits of a NodeId to use when mapping to a subscribed subnet.
	AttestationSubnetPrefixBits     uint64          `yaml:"ATTESTATION_SUBNET_PREFIX_BITS" spec:"true"`     // AttestationSubnetPrefixBits is defined as (ceillog2(ATTESTATION_SUBNET_COUNT) + ATTESTATION_SUBNET_EXTRA_BITS).
	SubnetsPerNode                  uint64          `yaml:"SUBNETS_PER_NODE" spec:"true"`                   // SubnetsPerNode is the number of long-lived subnets a beacon node should be subscribed to.
	NodeIdBits                      uint64          `yaml:"NODE_ID_BITS" spec:"true"`                       // NodeIdBits defines the bit length of a node id.

	// PeerDAS
	NumberOfColumns          uint64 `yaml:"NUMBER_OF_COLUMNS" spec:"true"`            // NumberOfColumns in the extended data matrix.
	MaxCellsInExtendedMatrix uint64 `yaml:"MAX_CELLS_IN_EXTENDED_MATRIX" spec:"true"` // MaxCellsInExtendedMatrix is the full data of one-dimensional erasure coding extended blobs (in row major format).
}

// InitializeForkSchedule initializes the schedules forks baked into the config.
func (b *BeaconChainConfig) InitializeForkSchedule() {
	// Reset Fork Version Schedule.
	b.ForkVersionSchedule = configForkSchedule(b)
	b.ForkVersionNames = configForkNames(b)
}

func configForkSchedule(b *BeaconChainConfig) map[[fieldparams.VersionLength]byte]primitives.Epoch {
	fvs := map[[fieldparams.VersionLength]byte]primitives.Epoch{}
	fvs[bytesutil.ToBytes4(b.GenesisForkVersion)] = b.GenesisEpoch
	fvs[bytesutil.ToBytes4(b.AltairForkVersion)] = b.AltairForkEpoch
	fvs[bytesutil.ToBytes4(b.BellatrixForkVersion)] = b.BellatrixForkEpoch
	fvs[bytesutil.ToBytes4(b.CapellaForkVersion)] = b.CapellaForkEpoch
	fvs[bytesutil.ToBytes4(b.DenebForkVersion)] = b.DenebForkEpoch
	fvs[bytesutil.ToBytes4(b.AlpacaForkVersion)] = b.AlpacaForkEpoch
	fvs[bytesutil.ToBytes4(b.BadgerForkVersion)] = b.BadgerForkEpoch
	return fvs
}

func configForkNames(b *BeaconChainConfig) map[[fieldparams.VersionLength]byte]string {
	cfv := ConfigForkVersions(b)
	fvn := map[[fieldparams.VersionLength]byte]string{}
	for k, v := range cfv {
		fvn[k] = version.String(v)
	}
	return fvn
}

// ConfigForkVersions returns a mapping between a fork version param and the version identifier
// from the runtime/version package.
func ConfigForkVersions(b *BeaconChainConfig) map[[fieldparams.VersionLength]byte]int {
	return map[[fieldparams.VersionLength]byte]int{
		bytesutil.ToBytes4(b.GenesisForkVersion):   version.Phase0,
		bytesutil.ToBytes4(b.AltairForkVersion):    version.Altair,
		bytesutil.ToBytes4(b.BellatrixForkVersion): version.Bellatrix,
		bytesutil.ToBytes4(b.CapellaForkVersion):   version.Capella,
		bytesutil.ToBytes4(b.DenebForkVersion):     version.Deneb,
		bytesutil.ToBytes4(b.AlpacaForkVersion):    version.Alpaca,
		bytesutil.ToBytes4(b.BadgerForkVersion):    version.Badger,
	}
}

// Eth1DataVotesLength returns the maximum length of the votes on the Eth1 data,
// computed from the parameters in BeaconChainConfig.
func (b *BeaconChainConfig) Eth1DataVotesLength() uint64 {
	return uint64(b.EpochsPerEth1VotingPeriod.Mul(uint64(b.SlotsPerEpoch)))
}

// PreviousEpochAttestationsLength returns the maximum length of the pending
// attestation list for the previous epoch, computed from the parameters in
// BeaconChainConfig.
func (b *BeaconChainConfig) PreviousEpochAttestationsLength() uint64 {
	return uint64(b.SlotsPerEpoch.Mul(b.MaxAttestations))
}

// CurrentEpochAttestationsLength returns the maximum length of the pending
// attestation list for the current epoch, computed from the parameters in
// BeaconChainConfig.
func (b *BeaconChainConfig) CurrentEpochAttestationsLength() uint64 {
	return uint64(b.SlotsPerEpoch.Mul(b.MaxAttestations))
}

// InitializeDepositPlan initializes the deposit plan of consensus.
func (b *BeaconChainConfig) InitializeDepositPlan() {
	b.DepositPlanEarlySlope = 160000000 * 1e9 / (b.EpochsPerYear * b.DepositPlanEarlyEnd)
	b.DepositPlanEarlyOffset = 40000000 * 1e9
	b.DepositPlanLaterSlope = 100000000 * 1e9 / (b.EpochsPerYear * (b.DepositPlanLaterEnd - b.DepositPlanEarlyEnd))
	b.DepositPlanLaterOffset = 133333334 * 1e9
	b.DepositPlanFinal = 300000000 * 1e9
}

// InitializeDolphinDepositPlan initializes the deposit plan of Dolphin testnet consensus.
func (b *BeaconChainConfig) InitializeDolphinDepositPlan() {
	b.DepositPlanEarlySlope = 160000000 * 1e9 / (b.EpochsPerYear * b.DepositPlanEarlyEnd)
	b.DepositPlanEarlyOffset = 40000000 * 1e9
	b.DepositPlanLaterSlope = 100000000 * 1e9 / (b.EpochsPerYear * (b.DepositPlanLaterEnd - b.DepositPlanEarlyEnd))
	b.DepositPlanLaterOffset = 133333334 * 1e9
	b.DepositPlanFinal = 300000000 * 1e9
}

// InitializeInactivityValues initializes INACTIVITY_SCORE_PENALTY_THRESHOLD and INACTIVITY_LEAK_BAILOUT_SCORE_THRESHOLD.
func (b *BeaconChainConfig) InitializeInactivityValues() {
	b.InactivityScorePenaltyThreshold = 225 * b.InactivityScoreBias                                                                   // 900
	b.InactivityLeakBailoutScoreThreshold = b.InactivityScorePenaltyThreshold + (b.InactivityPenaltyDuration * b.InactivityScoreBias) // 900 + (1575 * 4) = 900 + 6300 = 7200
}

// TtfbTimeoutDuration returns the time duration of the timeout.
func (b *BeaconChainConfig) TtfbTimeoutDuration() time.Duration {
	return time.Duration(b.TtfbTimeout) * time.Second
}

// RespTimeoutDuration returns the time duration of the timeout.
func (b *BeaconChainConfig) RespTimeoutDuration() time.Duration {
	return time.Duration(b.RespTimeout) * time.Second
}

// MaximumGossipClockDisparityDuration returns the time duration of the clock disparity.
func (b *BeaconChainConfig) MaximumGossipClockDisparityDuration() time.Duration {
	return time.Duration(b.MaximumGossipClockDisparity) * time.Millisecond
}

// DenebEnabled centralizes the check to determine if code paths
// that are specific to deneb should be allowed to execute. This will make it easier to find call sites that do this
// kind of check and remove them post-deneb.
func DenebEnabled() bool {
	return BeaconConfig().DenebForkEpoch < math.MaxUint64
}

// WithinDAPeriod checks if the block epoch is within MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS of the given current epoch.
func WithinDAPeriod(block, current primitives.Epoch) bool {
	return block+BeaconConfig().MinEpochsForBlobsSidecarsRequest >= current
}
