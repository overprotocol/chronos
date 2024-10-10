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
	// Electra Fork Epoch for mainnet config
	mainnetElectraForkEpoch = math.MaxUint64 // Far future / to be defined
)

var mainnetNetworkConfig = &NetworkConfig{
	ETH2Key:                    "over",
	AttSubnetKey:               "attnets",
	SyncCommsSubnetKey:         "syncnets",
	MinimumPeersInSubnetSearch: 20,
	ContractDeploymentBlock:    0, // Note: contract was deployed in genesis block.
	BootstrapNodes: []string{
		// Over Mainnet Bootnodes
		"enr:-LG4QMDxg9JWyQFDFDmNWYgsTBhH5dFmIW-X8q6g6S-3ZpDcMu6ouv4NnCOvZ9BGsIkWrwtx2iVaUAJn7dgS_TEA_XOGAZGN3idXh2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEj8Zt-YRvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQOuL8NQY7JaHKQ43e9HleHJNX0fBiGnX80b5y0z1fl82oN1ZHCCyyA", // Bootnode1
		"enr:-LG4QCu6n9asLF4GydPqGVMhGvM3QJ4CPdGTmxehYTnYWh17eh26of_NXeeh7f5YxMtR3MOnibbQ_iWo_WjREufzv-SGAZGN3icdh2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEmCriZYRvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQJUB5E3lpebYb4TgRatlNrvOxqhSmeX9ZwWOCEND5cllIN1ZHCCyyA", // Bootnode2
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
	TargetCommitteeSize:               128,
	MaxValidatorsPerCommittee:         2048,
	MaxCommitteesPerSlot:              64,
	MinPerEpochChurnLimit:             4,
	ChurnLimitQuotient:                1 << 16,
	ChurnLimitBias:                    1,
	ShuffleRoundCount:                 90,
	MinGenesisActiveValidatorCount:    16384,
	MinGenesisTime:                    1718690400, // Jun 19, 2024, 00 AM UTC+9.
	TargetAggregatorsPerCommittee:     16,
	HysteresisQuotient:                4,
	HysteresisDownwardMultiplier:      1,
	HysteresisUpwardMultiplier:        5,
	IssuanceRate:                      [11]uint64{20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 0},
	IssuancePrecision:                 1000,
	DepositPlanEarlyEnd:               4,
	DepositPlanLaterEnd:               10,
	RewardFeedbackPrecision:           1000000000000,
	RewardFeedbackThresholdReciprocal: 10,
	TargetChangeRate:                  1500000,
	MaxBoostYield:                     [11]uint64{10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000},

	// Gwei value constants.
	MinDepositAmount:          1 * 1e9,
	MaxEffectiveBalance:       256 * 1e9,
	EjectionBalance:           128 * 1e9,
	EffectiveBalanceIncrement: 8 * 1e9,
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

	// While eth1 mainnet block times are closer to 13s, we must conform with other clients in
	// order to vote on the correct eth1 blocks.
	//
	// Additional context: https://github.com/ethereum/consensus-specs/issues/2132
	// Bug prompting this change: https://github.com/prysmaticlabs/prysm/issues/7856
	// Future optimization: https://github.com/prysmaticlabs/prysm/issues/7739
	SecondsPerETH1Block: 12,

	// State list length constants.
	EpochsPerHistoricalVector: 65536,
	EpochsPerSlashingsVector:  8192,
	HistoricalRootsLimit:      16777216,
	ValidatorRegistryLimit:    1099511627776,

	// Reward and penalty quotients constants.
	BaseRewardFactor:               64,
	WhistleBlowerRewardQuotient:    512,
	ProposerRewardQuotient:         8,
	InactivityPenaltyQuotient:      67108864,
	MinSlashingPenaltyQuotient:     128,
	ProportionalSlashingMultiplier: 1,

	// Max operations per block constants.
	MaxProposerSlashings:             16,
	MaxAttesterSlashings:             2,
	MaxAttesterSlashingsElectra:      1,
	MaxAttestations:                  128,
	MaxAttestationsElectra:           8,
	MaxDeposits:                      16,
	MaxVoluntaryExits:                16,
	MaxBailOuts:                      16,
	MaxWithdrawalsPerPayload:         16,
	MaxBlsToExecutionChanges:         16,
	MaxValidatorsPerWithdrawalsSweep: 16384,

	// BLS domain values.
	DomainBeaconProposer:              bytesutil.Uint32ToBytes4(0x00000000),
	DomainBeaconAttester:              bytesutil.Uint32ToBytes4(0x01000000),
	DomainRandao:                      bytesutil.Uint32ToBytes4(0x02000000),
	DomainDeposit:                     bytesutil.Uint32ToBytes4(0x03000000),
	DomainVoluntaryExit:               bytesutil.Uint32ToBytes4(0x04000000),
	DomainSelectionProof:              bytesutil.Uint32ToBytes4(0x05000000),
	DomainAggregateAndProof:           bytesutil.Uint32ToBytes4(0x06000000),
	DomainSyncCommittee:               bytesutil.Uint32ToBytes4(0x07000000),
	DomainSyncCommitteeSelectionProof: bytesutil.Uint32ToBytes4(0x08000000),
	DomainContributionAndProof:        bytesutil.Uint32ToBytes4(0x09000000),
	DomainApplicationMask:             bytesutil.Uint32ToBytes4(0x00000001),
	DomainApplicationBuilder:          bytesutil.Uint32ToBytes4(0x00000001),
	DomainBLSToExecutionChange:        bytesutil.Uint32ToBytes4(0x0A000000),
	DomainConsolidation:               bytesutil.Uint32ToBytes4(0x0B000000),

	// Prysm constants.
	GenesisValidatorsRoot:          [32]byte{99, 42, 118, 239, 199, 87, 26, 107, 33, 162, 145, 86, 222, 195, 237, 225, 100, 124, 246, 131, 47, 17, 180, 161, 75, 90, 31, 0, 178, 164, 214, 126},
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
	BeaconStateFieldCount:          24,
	BeaconStateAltairFieldCount:    28,
	BeaconStateBellatrixFieldCount: 29,
	BeaconStateCapellaFieldCount:   32,
	BeaconStateDenebFieldCount:     32,
	BeaconStateElectraFieldCount:   41,

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
	ElectraForkVersion:   []byte{0x05, 0x00, 0x00, 0x18},
	ElectraForkEpoch:     mainnetElectraForkEpoch,

	// New values introduced in Altair hard fork 1.
	// Participation flag indices.
	TimelySourceFlagIndex: 0,
	TimelyTargetFlagIndex: 1,
	TimelyHeadFlagIndex:   2,

	// Incentivization weight values.
	TimelySourceWeight: 12,
	TimelyTargetWeight: 24,
	TimelyHeadWeight:   12,
	SyncRewardWeight:   0,
	ProposerWeight:     8,
	LightLayerWeight:   8,
	WeightDenominator:  64,

	// Validator related values.
	TargetAggregatorsPerSyncSubcommittee: 16,
	SyncCommitteeSubnetCount:             4,

	// Misc values.
	SyncCommitteeSize:            512,
	InactivityScoreBias:          4,
	InactivityScoreRecoveryRate:  16,
	BailOutScoreBias:             1000000000000000,
	BailOutScoreThreshold:        1575000000000000000,
	EpochsPerSyncCommitteePeriod: 256,

	// Updated penalty values.
	InactivityPenaltyQuotientAltair:         3 * 1 << 24, // 50331648
	MinSlashingPenaltyQuotientAltair:        64,
	ProportionalSlashingMultiplierAltair:    2,
	MinSlashingPenaltyQuotientBellatrix:     32,
	ProportionalSlashingMultiplierBellatrix: 3,
	InactivityPenaltyQuotientBellatrix:      1 << 24,

	// Light client
	MinSyncCommitteeParticipants: 1,
	MaxRequestLightClientUpdates: 128,

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

	MaxPerEpochActivationChurnLimit:  8,
	MinEpochsForBlobsSidecarsRequest: 4096,
	MaxRequestBlobSidecars:           768,
	MaxRequestBlocksDeneb:            128,

	// Values related to electra
	// TODO: Fix electra values
	MaxRequestDataColumnSidecars:          16384,
	DataColumnSidecarSubnetCount:          128,
	MinPerEpochChurnLimitElectra:          128_000_000_000,
	MaxPerEpochActivationExitChurnLimit:   256_000_000_000,
	MaxEffectiveBalanceElectra:            16384_000_000_000,
	MinSlashingPenaltyQuotientElectra:     4096,
	WhistleBlowerRewardQuotientElectra:    4096,
	PendingBalanceDepositLimit:            134_217_728,
	PendingPartialWithdrawalsLimit:        134_217_728,
	PendingConsolidationsLimit:            262_144,
	MinActivationBalance:                  256_000_000_000,
	MaxConsolidationsRequestsPerPayload:   1,
	MaxPendingPartialsPerWithdrawalsSweep: 8,
	FullExitRequestAmount:                 0,
	MaxWithdrawalRequestsPerPayload:       16,
	MaxDepositRequestsPerPayload:          8192, // 2**13 (= 8192)
	UnsetDepositRequestsStartIndex:        math.MaxUint64,

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
	c.ElectraForkVersion = make([]byte, fieldparams.VersionLength)

	c.GenesisForkVersion[fieldparams.VersionLength-1] = b
	c.AltairForkVersion[fieldparams.VersionLength-1] = b
	c.BellatrixForkVersion[fieldparams.VersionLength-1] = b
	c.CapellaForkVersion[fieldparams.VersionLength-1] = b
	c.DenebForkVersion[fieldparams.VersionLength-1] = b
	c.ElectraForkVersion[fieldparams.VersionLength-1] = b

	c.GenesisForkVersion[0] = 0
	c.AltairForkVersion[0] = 1
	c.BellatrixForkVersion[0] = 2
	c.CapellaForkVersion[0] = 3
	c.DenebForkVersion[0] = 4
	c.ElectraForkVersion[0] = 5
}
