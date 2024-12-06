package params

import (
	"math"
	"time"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
)

// MainnetConfig returns the configuration to be used in the main network.
func MainnetConfig() *BeaconChainConfig {
	if mainnetBeaconConfig.ForkVersionSchedule == nil {
		mainnetBeaconConfig.InitializeForkSchedule()
	}
	mainnetBeaconConfig.InitializeDepositPlan()
	mainnetBeaconConfig.InitializeInactivityValues()
	return mainnetBeaconConfig
}

const (
	// Genesis Fork Epoch for the mainnet config.
	genesisForkEpoch = 0
	// Altair Fork Epoch for mainnet config.
	mainnetAltairForkEpoch = 0 // epoch 0
	// Bellatrix Fork Epoch for mainnet config.
	mainnetBellatrixForkEpoch = 0 // epoch 0
	// Capella Fork Epoch for mainnet config.
	mainnetCapellaForkEpoch = 10 // epoch 10
	// Deneb Fork Epoch for mainnet config.
	mainnetDenebForkEpoch = math.MaxUint64 // not activated
	// Alpaca Fork Epoch for mainnet config
	mainnetAlpacaForkEpoch = math.MaxUint64 // Far future / to be defined
)

var mainnetNetworkConfig = &NetworkConfig{
	ETH2Key:                    "over",
	AttSubnetKey:               "attnets",
	MinimumPeersInSubnetSearch: 20,
	ContractDeploymentBlock:    0, // Note: contract was deployed in genesis block.
	BootstrapNodes: []string{
		// Over Mainnet Bootnodes
		"enr:-LG4QPp8Y6HmKPsO_XN1tQr_vGZb9ffClN82Q6iVW6yXzYkFVl2F_9wmYdKOTv-SUx-75xvPt6Ox52peebZklnvu7U-GAZOagV46h2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEnfU0eoRvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQP2qzv7Pu9M-cq4sXfhZwTiAdW3poF9suGlydcXpq4i1YN1ZHCCyyA", // Bootnode1
		"enr:-LG4QGE-EG3sZAHLqqhiOeUfbrIqlAnfSQCKSL2dd0JB-GnDfcTbIvafmjRWNdlbNTJ6UtNAiYxur5Rlp0IQbAL33BKGAZOagV1zh2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEnfUwuYRvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQPM9H2udXWq869IwctRfNSpzscXw8XGIrrnUHn7lE88aYN1ZHCCyyA", // Bootnode2
	},
}

var mainnetBeaconConfig = &BeaconChainConfig{
	// Constants (Non-configurable)
	FarFutureEpoch:           math.MaxUint64,
	FarFutureSlot:            math.MaxUint64,
	BaseRewardsPerEpoch:      4,
	DepositContractTreeDepth: 32,
	GenesisDelay:             30, // 30 seconds

	// Misc constant.
	TargetCommitteeSize:             128,
	MaxValidatorsPerCommittee:       2048,
	MaxCommitteesPerSlot:            64,
	MinPerEpochChurnLimit:           4,
	ChurnLimitQuotient:              1 << 16,
	ChurnLimitBias:                  1,
	ShuffleRoundCount:               90,
	MinGenesisActiveValidatorCount:  8192,
	MinGenesisTime:                  1718690400, // Jun 19, 2024, 00 AM UTC+9.
	TargetAggregatorsPerCommittee:   16,
	HysteresisQuotient:              4,
	HysteresisDownwardMultiplier:    1,
	HysteresisUpwardMultiplier:      5,
	IssuanceRate:                    [11]uint64{20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 0},
	IssuancePrecision:               1000,
	DepositPlanEarlyEnd:             4,
	DepositPlanLaterEnd:             10,
	RewardAdjustmentFactorDelta:     150,
	RewardAdjustmentFactorPrecision: 100000000,
	MaxRewardAdjustmentFactors:      [11]uint64{1000000, 1000000, 1000000, 1000000, 1000000, 1000000, 1000000, 1000000, 1000000, 1000000, 1000000},

	// Gwei value constants.
	MinDepositAmount:          32 * 1e9,
	MaxEffectiveBalance:       256 * 1e9,
	EffectiveBalanceIncrement: 1 * 1e9,
	MaxTokenSupply:            1000000000 * 1e9,
	IssuancePerYear:           20000000 * 1e9,

	// Initial value constants.
	BLSWithdrawalPrefixByte:         byte(0),
	ETH1AddressWithdrawalPrefixByte: byte(1),
	CompoundingWithdrawalPrefixByte: byte(2),
	ZeroHash:                        [32]byte{},

	// Time parameter constants.
	MinAttestationInclusionDelay:     1,
	SecondsPerSlot:                   12,
	SlotsPerEpoch:                    32,
	SqrRootSlotsPerEpoch:             5,
	EpochsPerYear:                    82125, // 365(Days)*24(Hours)*60(minutes)*60(Seconds)/12(Seconds per slot)/32(Slots per epoch)
	MinSeedLookahead:                 1,
	MaxSeedLookahead:                 4,
	EpochsPerEth1VotingPeriod:        64,
	SlotsPerHistoricalRoot:           8192,
	MinValidatorWithdrawabilityDelay: 256,
	ShardCommitteePeriod:             256,
	MinEpochsToInactivityPenalty:     4,
	Eth1FollowDistance:               1024,

	// Fork choice algorithm constants.
	ProposerScoreBoost:              40,
	ReorgWeightThreshold:            20,
	ReorgParentWeightThreshold:      160,
	ReorgMaxEpochsSinceFinalization: 2,
	IntervalsPerSlot:                3,

	// Ethereum PoW parameters.
	DepositChainID:         54176, // Chain ID of over mainnet.
	DepositNetworkID:       54176, // Network ID of over mainnet.
	DepositContractAddress: "000000000000000000000000000000000beac017",

	// Validator params.
	RandomSubnetsPerValidator:         1 << 0,
	EpochsPerRandomSubnetSubscription: 1 << 8,
	MinSlashingWithdrawableDelay:      8192,

	// While eth1 mainnet block times are closer to 13s, we must conform with other clients in
	// order to vote on the correct eth1 blocks.
	//
	// Additional context: https://github.com/ethereum/consensus-specs/issues/2132
	// Bug prompting this change: https://github.com/prysmaticlabs/prysm/issues/7856
	// Future optimization: https://github.com/prysmaticlabs/prysm/issues/7739
	SecondsPerETH1Block: 12,

	// State list length constants.
	EpochsPerHistoricalVector: 65536,
	HistoricalRootsLimit:      16777216,
	ValidatorRegistryLimit:    1099511627776,

	// Reward and penalty quotients constants.
	WhistleBlowerRewardQuotient: 512,
	ProposerRewardQuotient:      8,
	MinSlashingPenaltyQuotient:  128,

	// Max operations per block constants.
	MaxProposerSlashings:             16,
	MaxAttesterSlashings:             2,
	MaxAttesterSlashingsAlpaca:       1,
	MaxAttestations:                  128,
	MaxAttestationsAlpaca:            8,
	MaxDeposits:                      16,
	MaxDepositsAlpaca:                512,
	MaxVoluntaryExits:                16,
	MaxWithdrawalsPerPayload:         16,
	MaxValidatorsPerWithdrawalsSweep: 16384,

	// BLS domain values.
	DomainBeaconProposer:     bytesutil.Uint32ToBytes4(0x00000000),
	DomainBeaconAttester:     bytesutil.Uint32ToBytes4(0x01000000),
	DomainRandao:             bytesutil.Uint32ToBytes4(0x02000000),
	DomainDeposit:            bytesutil.Uint32ToBytes4(0x03000000),
	DomainVoluntaryExit:      bytesutil.Uint32ToBytes4(0x04000000),
	DomainSelectionProof:     bytesutil.Uint32ToBytes4(0x05000000),
	DomainAggregateAndProof:  bytesutil.Uint32ToBytes4(0x06000000),
	DomainApplicationMask:    bytesutil.Uint32ToBytes4(0x00000001),
	DomainApplicationBuilder: bytesutil.Uint32ToBytes4(0x00000001),

	// Prysm constants.
	GenesisValidatorsRoot:          [32]byte{4, 120, 213, 155, 32, 199, 36, 195, 69, 244, 33, 54, 197, 26, 109, 158, 179, 40, 217, 229, 64, 245, 178, 140, 249, 235, 92, 227, 176, 186, 205, 138},
	GweiPerEth:                     1000000000,
	BLSSecretKeyLength:             32,
	BLSPubkeyLength:                48,
	DefaultBufferSize:              10000,
	WithdrawalPrivkeyFileName:      "/shardwithdrawalkey",
	ValidatorPrivkeyFileName:       "/validatorprivatekey",
	RPCSyncCheck:                   1,
	EmptySignature:                 [96]byte{},
	DefaultPageSize:                250,
	MaxPeersToSync:                 15,
	SlotsPerArchivedPoint:          2048,
	GenesisCountdownInterval:       time.Minute,
	ConfigName:                     MainnetName,
	PresetBase:                     "mainnet",
	BeaconStateFieldCount:          21,
	BeaconStateAltairFieldCount:    22,
	BeaconStateBellatrixFieldCount: 23,
	BeaconStateCapellaFieldCount:   26,
	BeaconStateDenebFieldCount:     26,
	BeaconStateAlpacaFieldCount:    32,

	// Slasher related values.
	WeakSubjectivityPeriod:          54000,
	PruneSlasherStoragePeriod:       10,
	SlashingProtectionPruningEpochs: 512,

	// Weak subjectivity values.
	SafetyDecay: 10,

	// Fork related values.
	GenesisEpoch:         genesisForkEpoch,
	GenesisForkVersion:   []byte{0x00, 0x00, 0x00, 0x18},
	AltairForkVersion:    []byte{0x01, 0x00, 0x00, 0x18},
	AltairForkEpoch:      mainnetAltairForkEpoch,
	BellatrixForkVersion: []byte{0x02, 0x00, 0x00, 0x18},
	BellatrixForkEpoch:   mainnetBellatrixForkEpoch,
	CapellaForkVersion:   []byte{0x03, 0x00, 0x00, 0x18},
	CapellaForkEpoch:     mainnetCapellaForkEpoch,
	DenebForkVersion:     []byte{0x04, 0x00, 0x00, 0x18},
	DenebForkEpoch:       mainnetDenebForkEpoch,
	AlpacaForkVersion:    []byte{0x05, 0x00, 0x00, 0x18},
	AlpacaForkEpoch:      mainnetAlpacaForkEpoch,

	// New values introduced in Altair hard fork 1.
	// Participation flag indices.
	TimelySourceFlagIndex: 0,
	TimelyTargetFlagIndex: 1,
	TimelyHeadFlagIndex:   2,

	// Incentivization weight values.
	TimelySourceWeight: 12,
	TimelyTargetWeight: 24,
	TimelyHeadWeight:   12,
	ProposerWeight:     8,
	LightLayerWeight:   8,
	WeightDenominator:  64,

	// Misc values.
	InactivityScoreBias:         4,
	InactivityScoreRecoveryRate: 1,

	// Updated penalty values.
	MinSlashingPenaltyQuotientAltair:     64,
	MinSlashingPenaltyQuotientBellatrix:  32,
	InactivityPenaltyRate:                2,
	InactivityPenaltyRatePrecision:       100,
	InactivityPenaltyDuration:            1575, // epochs, 1 week
	InactivityLeakPenaltyBuffer:          10,   // 10%
	InactivityLeakPenaltyBufferPrecision: 100,

	// Bellatrix
	TerminalBlockHashActivationEpoch: 18446744073709551615,
	TerminalBlockHash:                [32]byte{},
	TerminalTotalDifficulty:          "0",
	EthBurnAddressHex:                "0x0000000000000000000000000000000000000000",
	DefaultBuilderGasLimit:           uint64(30000000),

	// Mevboost circuit breaker
	MaxBuilderConsecutiveMissedSlots: 3,
	MaxBuilderEpochMissedSlots:       5,
	// Execution engine timeout value
	ExecutionEngineTimeoutValue: 8, // 8 seconds default based on: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#core

	// Subnet value
	BlobsidecarSubnetCount: 6,

	MinEpochsForBlobsSidecarsRequest: 4096,
	MaxRequestBlobSidecars:           768,
	MaxRequestBlocksDeneb:            128,

	// Values related to electra
	// TODO: Fix electra values
	MaxRequestDataColumnSidecars:           16384,
	DataColumnSidecarSubnetCount:           128,
	MinPerEpochChurnLimitAlpaca:            1_024_000_000_000,
	MinPerEpochActivationBalanceChurnLimit: 4_096_000_000_000,
	MaxEffectiveBalanceAlpaca:              16384_000_000_000,
	PendingDepositLimit:                    134_217_728,
	PendingPartialWithdrawalsLimit:         134_217_728,
	MinActivationBalance:                   256_000_000_000,
	MaxPendingPartialsPerWithdrawalsSweep:  8,
	MaxPendingDepositsPerEpoch:             64,
	FullExitRequestAmount:                  0,
	MaxWithdrawalRequestsPerPayload:        16,
	MaxDepositRequestsPerPayload:           8192, // 2**13 (= 8192)
	UnsetDepositRequestsStartIndex:         math.MaxUint64,

	// Values related to alpaca
	MinSlashingPenaltyQuotientAlpaca:  10,
	WhistleBlowerRewardQuotientAlpaca: 10,

	// PeerDAS
	NumberOfColumns:          128,
	MaxCellsInExtendedMatrix: 768,

	// Values related to networking parameters.
	GossipMaxSize:                   10 * 1 << 20, // 10 MiB
	MaxChunkSize:                    10 * 1 << 20, // 10 MiB
	AttestationSubnetCount:          64,
	AttestationPropagationSlotRange: 32,
	MaxRequestBlocks:                1 << 10, // 1024
	TtfbTimeout:                     5,
	RespTimeout:                     10,
	MaximumGossipClockDisparity:     500,
	MessageDomainInvalidSnappy:      [4]byte{00, 00, 00, 00},
	MessageDomainValidSnappy:        [4]byte{01, 00, 00, 00},
	MinEpochsForBlockRequests:       33024, // MIN_VALIDATOR_WITHDRAWABILITY_DELAY + CHURN_LIMIT_QUOTIENT / 2 (= 33024, ~5 months)
	EpochsPerSubnetSubscription:     256,
	AttestationSubnetExtraBits:      0,
	AttestationSubnetPrefixBits:     6,
	SubnetsPerNode:                  2,
	NodeIdBits:                      256,
}

// MainnetTestConfig provides a version of the mainnet config that has a different name
// and a different fork choice schedule. This can be used in cases where we want to use config values
// that are consistent with mainnet, but won't conflict or cause the hard-coded genesis to be loaded.
func MainnetTestConfig() *BeaconChainConfig {
	mn := MainnetConfig().Copy()
	mn.ConfigName = MainnetTestName
	FillTestVersions(mn, 128)
	return mn
}

// FillTestVersions replaces the fork schedule in the given BeaconChainConfig with test values, using the given
// byte argument as the high byte (common across forks).
func FillTestVersions(c *BeaconChainConfig, b byte) {
	c.GenesisForkVersion = make([]byte, fieldparams.VersionLength)
	c.AltairForkVersion = make([]byte, fieldparams.VersionLength)
	c.BellatrixForkVersion = make([]byte, fieldparams.VersionLength)
	c.CapellaForkVersion = make([]byte, fieldparams.VersionLength)
	c.DenebForkVersion = make([]byte, fieldparams.VersionLength)
	c.AlpacaForkVersion = make([]byte, fieldparams.VersionLength)

	c.GenesisForkVersion[fieldparams.VersionLength-1] = b
	c.AltairForkVersion[fieldparams.VersionLength-1] = b
	c.BellatrixForkVersion[fieldparams.VersionLength-1] = b
	c.CapellaForkVersion[fieldparams.VersionLength-1] = b
	c.DenebForkVersion[fieldparams.VersionLength-1] = b
	c.AlpacaForkVersion[fieldparams.VersionLength-1] = b

	c.GenesisForkVersion[0] = 0
	c.AltairForkVersion[0] = 1
	c.BellatrixForkVersion[0] = 2
	c.CapellaForkVersion[0] = 3
	c.DenebForkVersion[0] = 4
	c.AlpacaForkVersion[0] = 5
}
