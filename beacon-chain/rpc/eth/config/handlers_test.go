package config

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/network/forks"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestGetDepositContract(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig().Copy()
	config.DepositChainID = uint64(10)
	config.DepositContractAddress = "0x4242424242424242424242424242424242424242"
	params.OverrideBeaconConfig(config)

	request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v1/config/deposit_contract", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	GetDepositContract(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	response := structs.GetDepositContractResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &response))
	assert.Equal(t, "10", response.Data.ChainId)
	assert.Equal(t, "0x4242424242424242424242424242424242424242", response.Data.Address)
}

func TestGetSpec(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig().Copy()

	config.ConfigName = "ConfigName"
	config.PresetBase = "PresetBase"
	config.MaxCommitteesPerSlot = 1
	config.TargetCommitteeSize = 2
	config.MaxValidatorsPerCommittee = 3
	config.MinPerEpochChurnLimit = 4
	config.ChurnLimitQuotient = 5
	config.ShuffleRoundCount = 6
	config.MinGenesisActiveValidatorCount = 7
	config.MinGenesisTime = 8
	config.HysteresisQuotient = 9
	config.HysteresisDownwardMultiplier = 10
	config.HysteresisUpwardMultiplier = 11
	config.Eth1FollowDistance = 12
	config.TargetAggregatorsPerCommittee = 13
	config.RandomSubnetsPerValidator = 14
	config.EpochsPerRandomSubnetSubscription = 15
	config.SecondsPerETH1Block = 16
	config.DepositChainID = 17
	config.DepositNetworkID = 18
	config.DepositContractAddress = "DepositContractAddress"
	config.MinDepositAmount = 20
	config.MaxEffectiveBalance = 21
	config.EffectiveBalanceIncrement = 23
	config.GenesisForkVersion = []byte("GenesisForkVersion")
	config.AltairForkVersion = []byte("AltairForkVersion")
	config.AltairForkEpoch = 100
	config.BellatrixForkVersion = []byte("BellatrixForkVersion")
	config.BellatrixForkEpoch = 101
	config.CapellaForkVersion = []byte("CapellaForkVersion")
	config.CapellaForkEpoch = 102
	config.DenebForkVersion = []byte("DenebForkVersion")
	config.DenebForkEpoch = 103
	config.AlpacaForkVersion = []byte("AlpacaForkVersion")
	config.AlpacaForkEpoch = 104
	config.BadgerForkVersion = []byte("BadgerForkVersion")
	config.BadgerForkEpoch = 105
	config.BLSWithdrawalPrefixByte = byte('b')
	config.ETH1AddressWithdrawalPrefixByte = byte('c')
	config.GenesisDelay = 24
	config.SecondsPerSlot = 25
	config.MinAttestationInclusionDelay = 26
	config.SlotsPerEpoch = 27
	config.MinSeedLookahead = 28
	config.MaxSeedLookahead = 29
	config.EpochsPerEth1VotingPeriod = 30
	config.SlotsPerHistoricalRoot = 31
	config.MinValidatorWithdrawabilityDelay = 32
	config.ShardCommitteePeriod = 33
	config.MinEpochsToInactivityPenalty = 34
	config.EpochsPerHistoricalVector = 35
	config.MinSlashingWithdrawableDelay = 36
	config.HistoricalRootsLimit = 37
	config.ValidatorRegistryLimit = 38
	config.WhistleBlowerRewardQuotient = 40
	config.ProposerRewardQuotient = 41
	config.MinSlashingPenaltyQuotient = 44
	config.MaxProposerSlashings = 45
	config.MaxAttesterSlashings = 46
	config.MaxAttestations = 47
	config.MaxDeposits = 48
	config.MaxVoluntaryExits = 49
	config.TimelyHeadFlagIndex = 50
	config.TimelySourceFlagIndex = 51
	config.TimelyTargetFlagIndex = 52
	config.TimelyHeadWeight = 53
	config.TimelySourceWeight = 54
	config.TimelyTargetWeight = 55
	config.WeightDenominator = 56
	config.InactivityScoreBias = 57
	config.MinSlashingPenaltyQuotientAltair = 59
	config.InactivityScoreRecoveryRate = 60
	config.TerminalBlockHash = common.HexToHash("TerminalBlockHash")
	config.TerminalBlockHashActivationEpoch = 62
	config.TerminalTotalDifficulty = "63"
	config.DefaultFeeRecipient = common.HexToAddress("DefaultFeeRecipient")
	config.MaxWithdrawalsPerPayload = 65
	config.MaxValidatorsPerWithdrawalsSweep = 67
	config.ChurnLimitBias = 68
	config.IssuanceRate = [11]uint64{69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69}
	config.IssuancePrecision = 70
	config.DepositPlanEarlyEnd = 71
	config.DepositPlanEarlySlope = 72
	config.DepositPlanEarlyOffset = 73
	config.DepositPlanLaterEnd = 74
	config.DepositPlanLaterSlope = 75
	config.DepositPlanLaterOffset = 76
	config.DepositPlanFinal = 77
	config.RewardAdjustmentFactorDelta = 78
	config.RewardAdjustmentFactorPrecision = 79
	config.MaxRewardAdjustmentFactors = [11]uint64{80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80}
	config.MaxTokenSupply = 82
	config.EpochsPerYear = 83
	config.IssuancePerYear = 84
	config.LightLayerWeight = 85
	config.CompoundingWithdrawalPrefixByte = byte('d')
	config.PendingPartialWithdrawalsLimit = 90
	config.MinActivationBalance = 91
	config.PendingDepositLimit = 92
	config.MaxPendingPartialsPerWithdrawalsSweep = 93
	config.MaxPartialWithdrawalsPerPayload = 94
	config.FullExitRequestAmount = 95
	config.MaxAttesterSlashingsAlpaca = 96
	config.MaxAttestationsAlpaca = 97
	config.MaxWithdrawalRequestsPerPayload = 98
	config.MaxCellsInExtendedMatrix = 99
	config.UnsetDepositRequestsStartIndex = 100
	config.MaxDepositRequestsPerPayload = 101
	config.MaxPendingDepositsPerEpoch = 102
	config.MinSlashingPenaltyQuotientAlpaca = 103
	config.WhistleBlowerRewardQuotientAlpaca = 104
	config.InactivityPenaltyRate = 105
	config.InactivityPenaltyRatePrecision = 106
	config.InactivityPenaltyDuration = 107
	config.InactivityScorePenaltyThreshold = 108
	config.InactivityLeakPenaltyBuffer = 109
	config.InactivityLeakPenaltyBufferPrecision = 110
	config.InactivityLeakBailoutScoreThreshold = 111
	config.MaxDepositsAlpaca = 112

	var dbp [4]byte
	copy(dbp[:], []byte{'0', '0', '0', '1'})
	config.DomainBeaconProposer = dbp
	var dba [4]byte
	copy(dba[:], []byte{'0', '0', '0', '2'})
	config.DomainBeaconAttester = dba
	var dr [4]byte
	copy(dr[:], []byte{'0', '0', '0', '3'})
	config.DomainRandao = dr
	var dd [4]byte
	copy(dd[:], []byte{'0', '0', '0', '4'})
	config.DomainDeposit = dd
	var dve [4]byte
	copy(dve[:], []byte{'0', '0', '0', '5'})
	config.DomainVoluntaryExit = dve
	var dsp [4]byte
	copy(dsp[:], []byte{'0', '0', '0', '6'})
	config.DomainSelectionProof = dsp
	var daap [4]byte
	copy(daap[:], []byte{'0', '0', '0', '7'})
	config.DomainAggregateAndProof = daap
	var dam [4]byte
	copy(dam[:], []byte{'1', '0', '0', '0'})
	config.DomainApplicationMask = dam
	params.OverrideBeaconConfig(config)

	request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v1/config/spec", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	GetSpec(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	resp := structs.GetSpecResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]interface{})
	require.Equal(t, true, ok)

	assert.Equal(t, 159, len(data))
	for k, v := range data {
		t.Run(k, func(t *testing.T) {
			switch k {
			case "CONFIG_NAME":
				assert.Equal(t, "ConfigName", v)
			case "PRESET_BASE":
				assert.Equal(t, "PresetBase", v)
			case "MAX_COMMITTEES_PER_SLOT":
				assert.Equal(t, "1", v)
			case "TARGET_COMMITTEE_SIZE":
				assert.Equal(t, "2", v)
			case "MAX_VALIDATORS_PER_COMMITTEE":
				assert.Equal(t, "3", v)
			case "MIN_PER_EPOCH_CHURN_LIMIT":
				assert.Equal(t, "4", v)
			case "CHURN_LIMIT_QUOTIENT":
				assert.Equal(t, "5", v)
			case "SHUFFLE_ROUND_COUNT":
				assert.Equal(t, "6", v)
			case "MIN_GENESIS_ACTIVE_VALIDATOR_COUNT":
				assert.Equal(t, "7", v)
			case "MIN_GENESIS_TIME":
				assert.Equal(t, "8", v)
			case "HYSTERESIS_QUOTIENT":
				assert.Equal(t, "9", v)
			case "HYSTERESIS_DOWNWARD_MULTIPLIER":
				assert.Equal(t, "10", v)
			case "HYSTERESIS_UPWARD_MULTIPLIER":
				assert.Equal(t, "11", v)
			case "SAFE_SLOTS_TO_UPDATE_JUSTIFIED":
				assert.Equal(t, "0", v)
			case "ETH1_FOLLOW_DISTANCE":
				assert.Equal(t, "12", v)
			case "TARGET_AGGREGATORS_PER_COMMITTEE":
				assert.Equal(t, "13", v)
			case "RANDOM_SUBNETS_PER_VALIDATOR":
				assert.Equal(t, "14", v)
			case "EPOCHS_PER_RANDOM_SUBNET_SUBSCRIPTION":
				assert.Equal(t, "15", v)
			case "SECONDS_PER_ETH1_BLOCK":
				assert.Equal(t, "16", v)
			case "DEPOSIT_CHAIN_ID":
				assert.Equal(t, "17", v)
			case "DEPOSIT_NETWORK_ID":
				assert.Equal(t, "18", v)
			case "DEPOSIT_CONTRACT_ADDRESS":
				assert.Equal(t, "DepositContractAddress", v)
			case "MIN_DEPOSIT_AMOUNT":
				assert.Equal(t, "20", v)
			case "MAX_EFFECTIVE_BALANCE":
				assert.Equal(t, "21", v)
			case "EFFECTIVE_BALANCE_INCREMENT":
				assert.Equal(t, "23", v)
			case "GENESIS_FORK_VERSION":
				assert.Equal(t, "0x"+hex.EncodeToString([]byte("GenesisForkVersion")), v)
			case "ALTAIR_FORK_VERSION":
				assert.Equal(t, "0x"+hex.EncodeToString([]byte("AltairForkVersion")), v)
			case "ALTAIR_FORK_EPOCH":
				assert.Equal(t, "100", v)
			case "BELLATRIX_FORK_VERSION":
				assert.Equal(t, "0x"+hex.EncodeToString([]byte("BellatrixForkVersion")), v)
			case "BELLATRIX_FORK_EPOCH":
				assert.Equal(t, "101", v)
			case "CAPELLA_FORK_VERSION":
				assert.Equal(t, "0x"+hex.EncodeToString([]byte("CapellaForkVersion")), v)
			case "CAPELLA_FORK_EPOCH":
				assert.Equal(t, "102", v)
			case "DENEB_FORK_VERSION":
				assert.Equal(t, "0x"+hex.EncodeToString([]byte("DenebForkVersion")), v)
			case "DENEB_FORK_EPOCH":
				assert.Equal(t, "103", v)
			case "ALPACA_FORK_VERSION":
				assert.Equal(t, "0x"+hex.EncodeToString([]byte("AlpacaForkVersion")), v)
			case "ALPACA_FORK_EPOCH":
				assert.Equal(t, "104", v)
			case "BADGER_FORK_VERSION":
				assert.Equal(t, "0x"+hex.EncodeToString([]byte("BadgerForkVersion")), v)
			case "BADGER_FORK_EPOCH":
				assert.Equal(t, "105", v)
			case "MIN_ANCHOR_POW_BLOCK_DIFFICULTY":
				assert.Equal(t, "1000", v)
			case "BLS_WITHDRAWAL_PREFIX":
				assert.Equal(t, "0x62", v)
			case "ETH1_ADDRESS_WITHDRAWAL_PREFIX":
				assert.Equal(t, "0x63", v)
			case "GENESIS_DELAY":
				assert.Equal(t, "24", v)
			case "SECONDS_PER_SLOT":
				assert.Equal(t, "25", v)
			case "MIN_ATTESTATION_INCLUSION_DELAY":
				assert.Equal(t, "26", v)
			case "SLOTS_PER_EPOCH":
				assert.Equal(t, "27", v)
			case "MIN_SEED_LOOKAHEAD":
				assert.Equal(t, "28", v)
			case "MAX_SEED_LOOKAHEAD":
				assert.Equal(t, "29", v)
			case "EPOCHS_PER_ETH1_VOTING_PERIOD":
				assert.Equal(t, "30", v)
			case "SLOTS_PER_HISTORICAL_ROOT":
				assert.Equal(t, "31", v)
			case "MIN_VALIDATOR_WITHDRAWABILITY_DELAY":
				assert.Equal(t, "32", v)
			case "SHARD_COMMITTEE_PERIOD":
				assert.Equal(t, "33", v)
			case "MIN_EPOCHS_TO_INACTIVITY_PENALTY":
				assert.Equal(t, "34", v)
			case "EPOCHS_PER_HISTORICAL_VECTOR":
				assert.Equal(t, "35", v)
			case "MIN_SLASHING_WITHDRAWABLE_DELAY":
				assert.Equal(t, "36", v)
			case "HISTORICAL_ROOTS_LIMIT":
				assert.Equal(t, "37", v)
			case "VALIDATOR_REGISTRY_LIMIT":
				assert.Equal(t, "38", v)
			case "WHISTLEBLOWER_REWARD_QUOTIENT":
				assert.Equal(t, "40", v)
			case "PROPOSER_REWARD_QUOTIENT":
				assert.Equal(t, "41", v)
			case "HF1_INACTIVITY_PENALTY_QUOTIENT":
				assert.Equal(t, "41", v)
			case "MIN_SLASHING_PENALTY_QUOTIENT":
				assert.Equal(t, "44", v)
			case "HF1_MIN_SLASHING_PENALTY_QUOTIENT":
				assert.Equal(t, "45", v)
			case "HF1_PROPORTIONAL_SLASHING_MULTIPLIER":
				assert.Equal(t, "47", v)
			case "MAX_PROPOSER_SLASHINGS":
				assert.Equal(t, "45", v)
			case "MAX_ATTESTER_SLASHINGS":
				assert.Equal(t, "46", v)
			case "MAX_ATTESTATIONS":
				assert.Equal(t, "47", v)
			case "MAX_DEPOSITS":
				assert.Equal(t, "48", v)
			case "MAX_VOLUNTARY_EXITS":
				assert.Equal(t, "49", v)
			case "TIMELY_HEAD_FLAG_INDEX":
				assert.Equal(t, "0x32", v)
			case "TIMELY_SOURCE_FLAG_INDEX":
				assert.Equal(t, "0x33", v)
			case "TIMELY_TARGET_FLAG_INDEX":
				assert.Equal(t, "0x34", v)
			case "TIMELY_HEAD_WEIGHT":
				assert.Equal(t, "53", v)
			case "TIMELY_SOURCE_WEIGHT":
				assert.Equal(t, "54", v)
			case "TIMELY_TARGET_WEIGHT":
				assert.Equal(t, "55", v)
			case "WEIGHT_DENOMINATOR":
				assert.Equal(t, "56", v)
			case "INACTIVITY_SCORE_BIAS":
				assert.Equal(t, "57", v)
			case "MIN_SLASHING_PENALTY_QUOTIENT_ALTAIR":
				assert.Equal(t, "59", v)
			case "INACTIVITY_SCORE_RECOVERY_RATE":
				assert.Equal(t, "60", v)
			case "PROPOSER_WEIGHT":
				assert.Equal(t, "8", v)
			case "DOMAIN_BEACON_PROPOSER":
				assert.Equal(t, "0x30303031", v)
			case "DOMAIN_BEACON_ATTESTER":
				assert.Equal(t, "0x30303032", v)
			case "DOMAIN_RANDAO":
				assert.Equal(t, "0x30303033", v)
			case "DOMAIN_DEPOSIT":
				assert.Equal(t, "0x30303034", v)
			case "DOMAIN_VOLUNTARY_EXIT":
				assert.Equal(t, "0x30303035", v)
			case "DOMAIN_SELECTION_PROOF":
				assert.Equal(t, "0x30303036", v)
			case "DOMAIN_AGGREGATE_AND_PROOF":
				assert.Equal(t, "0x30303037", v)
			case "DOMAIN_APPLICATION_MASK":
				assert.Equal(t, "0x31303030", v)
			case "DOMAIN_APPLICATION_BUILDER":
				assert.Equal(t, "0x00000001", v)
			case "DOMAIN_BLOB_SIDECAR":
				assert.Equal(t, "0x00000000", v)
			case "TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH":
				assert.Equal(t, "62", v)
			case "TERMINAL_BLOCK_HASH":
				s, ok := v.(string)
				require.Equal(t, true, ok)
				assert.Equal(t, common.HexToHash("TerminalBlockHash"), common.HexToHash(s))
			case "TERMINAL_TOTAL_DIFFICULTY":
				assert.Equal(t, "63", v)
			case "DefaultFeeRecipient":
				assert.Equal(t, common.HexToAddress("DefaultFeeRecipient"), v)
			case "MIN_SLASHING_PENALTY_QUOTIENT_BELLATRIX":
				assert.Equal(t, "32", v)
			case "INACTIVITY_PENALTY_QUOTIENT_BELLATRIX":
				assert.Equal(t, "16777216", v)
			case "PROPOSER_SCORE_BOOST":
				assert.Equal(t, "40", v)
			case "INTERVALS_PER_SLOT":
				assert.Equal(t, "3", v)
			case "MAX_WITHDRAWALS_PER_PAYLOAD":
				assert.Equal(t, "65", v)
			case "MAX_VALIDATORS_PER_WITHDRAWALS_SWEEP":
				assert.Equal(t, "67", v)
			case "REORG_MAX_EPOCHS_SINCE_FINALIZATION":
				assert.Equal(t, "2", v)
			case "REORG_WEIGHT_THRESHOLD":
				assert.Equal(t, "20", v)
			case "REORG_PARENT_WEIGHT_THRESHOLD":
				assert.Equal(t, "160", v)
			case "MAX_PER_EPOCH_ACTIVATION_CHURN_LIMIT":
				assert.Equal(t, "8", v)
			case "SAFE_SLOTS_TO_IMPORT_OPTIMISTICALLY":
			case "NODE_ID_BITS":
				assert.Equal(t, "256", v)
			case "ATTESTATION_SUBNET_EXTRA_BITS":
				assert.Equal(t, "0", v)
			case "ATTESTATION_SUBNET_PREFIX_BITS":
				assert.Equal(t, "6", v)
			case "SUBNETS_PER_NODE":
				assert.Equal(t, "2", v)
			case "EPOCHS_PER_SUBNET_SUBSCRIPTION":
				assert.Equal(t, "256", v)
			case "MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS":
				assert.Equal(t, "4096", v)
			case "MAX_REQUEST_BLOB_SIDECARS":
				assert.Equal(t, "768", v)
			case "MESSAGE_DOMAIN_INVALID_SNAPPY":
				assert.Equal(t, "0x00000000", v)
			case "MESSAGE_DOMAIN_VALID_SNAPPY":
				assert.Equal(t, "0x01000000", v)
			case "ATTESTATION_PROPAGATION_SLOT_RANGE":
				assert.Equal(t, "32", v)
			case "RESP_TIMEOUT":
				assert.Equal(t, "10", v)
			case "TTFB_TIMEOUT":
				assert.Equal(t, "5", v)
			case "MIN_EPOCHS_FOR_BLOCK_REQUESTS":
				assert.Equal(t, "33024", v)
			case "GOSSIP_MAX_SIZE":
				assert.Equal(t, "10485760", v)
			case "MAX_CHUNK_SIZE":
				assert.Equal(t, "10485760", v)
			case "ATTESTATION_SUBNET_COUNT":
				assert.Equal(t, "64", v)
			case "MAXIMUM_GOSSIP_CLOCK_DISPARITY":
				assert.Equal(t, "500", v)
			case "MAX_REQUEST_BLOCKS":
				assert.Equal(t, "1024", v)
			case "MAX_REQUEST_BLOCKS_DENEB":
				assert.Equal(t, "128", v)
			case "NUMBER_OF_COLUMNS":
				assert.Equal(t, "128", v)
			case "MIN_PER_EPOCH_CHURN_LIMIT_ALPACA":
				assert.Equal(t, "1024000000000", v)
			case "MIN_PER_EPOCH_ACTIVATION_BALANCE_CHURN_LIMIT":
				assert.Equal(t, "4096000000000", v)
			case "DATA_COLUMN_SIDECAR_SUBNET_COUNT":
				assert.Equal(t, "128", v)
			case "MAX_REQUEST_DATA_COLUMN_SIDECARS":
				assert.Equal(t, "16384", v)
			case "CHURN_LIMIT_BIAS":
				assert.Equal(t, "68", v)
			case "ISSUANCE_RATE":
				assert.Equal(t, "[69,69,69,69,69,69,69,69,69,69,69]", v)
			case "ISSUANCE_PRECISION":
				assert.Equal(t, "70", v)
			case "DEPOSIT_PLAN_EARLY_END":
				assert.Equal(t, "71", v)
			case "DEPOSIT_PLAN_EARLY_SLOPE":
				assert.Equal(t, "72", v)
			case "DEPOSIT_PLAN_EARLY_OFFSET":
				assert.Equal(t, "73", v)
			case "DEPOSIT_PLAN_LATER_END":
				assert.Equal(t, "74", v)
			case "DEPOSIT_PLAN_LATER_SLOPE":
				assert.Equal(t, "75", v)
			case "DEPOSIT_PLAN_LATER_OFFSET":
				assert.Equal(t, "76", v)
			case "DEPOSIT_PLAN_FINAL":
				assert.Equal(t, "77", v)
			case "REWARD_ADJUSTMENT_FACTOR_DELTA":
				assert.Equal(t, "78", v)
			case "REWARD_ADJUSTMENT_FACTOR_PRECISION":
				assert.Equal(t, "79", v)
			case "MAX_REWARD_ADJUSTMENT_FACTORS":
				assert.Equal(t, "[80,80,80,80,80,80,80,80,80,80,80]", v)
			case "REWARD_FEEDBACK_PRECISION":
				assert.Equal(t, "89", v)
			case "REWARD_FEEDBACK_THRESHOLD_RECIPROCAL":
				assert.Equal(t, "90", v)
			case "MAX_BOOST_YIELD":
				assert.Equal(t, "[91,91,91,91,91,91,91,91,91,91,91]", v)
			case "TARGET_CHANGE_RATE":
				assert.Equal(t, "92", v)
			case "MAX_TOKEN_SUPPLY":
				assert.Equal(t, "82", v)
			case "EPOCHS_PER_YEAR":
				assert.Equal(t, "83", v)
			case "ISSUANCE_PER_YEAR":
				assert.Equal(t, "84", v)
			case "LIGHT_LAYER_WEIGHT":
				assert.Equal(t, "85", v)
			case "MAX_EFFECTIVE_BALANCE_ALPACA":
				assert.Equal(t, "16384000000000", v)
			case "COMPOUNDING_WITHDRAWAL_PREFIX":
				assert.Equal(t, "0x64", v)
			case "PENDING_PARTIAL_WITHDRAWALS_LIMIT":
				assert.Equal(t, "90", v)
			case "MIN_ACTIVATION_BALANCE":
				assert.Equal(t, "91", v)
			case "PENDING_DEPOSITS_LIMIT":
				assert.Equal(t, "92", v)
			case "MAX_PENDING_PARTIALS_PER_WITHDRAWALS_SWEEP":
				assert.Equal(t, "93", v)
			case "MAX_PARTIAL_WITHDRAWALS_PER_PAYLOAD":
				assert.Equal(t, "94", v)
			case "FULL_EXIT_REQUEST_AMOUNT":
				assert.Equal(t, "95", v)
			case "MAX_ATTESTER_SLASHINGS_ALPACA":
				assert.Equal(t, "96", v)
			case "MAX_ATTESTATIONS_ALPACA":
				assert.Equal(t, "97", v)
			case "MAX_WITHDRAWAL_REQUESTS_PER_PAYLOAD":
				assert.Equal(t, "98", v)
			case "MAX_CELLS_IN_EXTENDED_MATRIX":
				assert.Equal(t, "99", v)
			case "UNSET_DEPOSIT_REQUESTS_START_INDEX":
				assert.Equal(t, "100", v)
			case "MAX_DEPOSIT_REQUESTS_PER_PAYLOAD":
				assert.Equal(t, "101", v)
			case "MAX_PENDING_DEPOSITS_PER_EPOCH":
				assert.Equal(t, "102", v)
			case "MAX_DEPOSITS_ALPACA":
				assert.Equal(t, "112", v)
			case "MIN_SLASHING_PENALTY_QUOTIENT_ALPACA":
				assert.Equal(t, "103", v)
			case "WHISTLEBLOWER_REWARD_QUOTIENT_ALPACA":
				assert.Equal(t, "104", v)
			case "INACTIVITY_PENALTY_RATE":
				assert.Equal(t, "105", v)
			case "INACTIVITY_PENALTY_RATE_PRECISION":
				assert.Equal(t, "106", v)
			case "INACTIVITY_PENALTY_DURATION":
				assert.Equal(t, "107", v)
			case "INACTIVITY_SCORE_PENALTY_THRESHOLD":
				assert.Equal(t, "108", v)
			case "INACTIVITY_LEAK_PENALTY_BUFFER":
				assert.Equal(t, "109", v)
			case "INACTIVITY_LEAK_PENALTY_BUFFER_PRECISION":
				assert.Equal(t, "110", v)
			case "INACTIVITY_LEAK_BAILOUT_SCORE_THRESHOLD":
				assert.Equal(t, "111", v)
			default:
				t.Errorf("Incorrect key: %s", k)
			}
		})
	}
}

func TestForkSchedule_Ok(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		genesisForkVersion := []byte("Genesis")
		firstForkVersion, firstForkEpoch := []byte("Firs"), primitives.Epoch(100)
		secondForkVersion, secondForkEpoch := []byte("Seco"), primitives.Epoch(200)
		thirdForkVersion, thirdForkEpoch := []byte("Thir"), primitives.Epoch(300)

		params.SetupTestConfigCleanup(t)
		config := params.BeaconConfig().Copy()
		config.GenesisForkVersion = genesisForkVersion
		// Create fork schedule adding keys in non-sorted order.
		schedule := make(map[[4]byte]primitives.Epoch, 3)
		schedule[bytesutil.ToBytes4(secondForkVersion)] = secondForkEpoch
		schedule[bytesutil.ToBytes4(firstForkVersion)] = firstForkEpoch
		schedule[bytesutil.ToBytes4(thirdForkVersion)] = thirdForkEpoch
		config.ForkVersionSchedule = schedule
		params.OverrideBeaconConfig(config)

		request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v1/config/fork_schedule", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		GetForkSchedule(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetForkScheduleResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.Equal(t, 3, len(resp.Data))
		fork := resp.Data[0]
		assert.DeepEqual(t, hexutil.Encode(genesisForkVersion), fork.PreviousVersion)
		assert.DeepEqual(t, hexutil.Encode(firstForkVersion), fork.CurrentVersion)
		assert.Equal(t, fmt.Sprintf("%d", firstForkEpoch), fork.Epoch)
		fork = resp.Data[1]
		assert.DeepEqual(t, hexutil.Encode(firstForkVersion), fork.PreviousVersion)
		assert.DeepEqual(t, hexutil.Encode(secondForkVersion), fork.CurrentVersion)
		assert.Equal(t, fmt.Sprintf("%d", secondForkEpoch), fork.Epoch)
		fork = resp.Data[2]
		assert.DeepEqual(t, hexutil.Encode(secondForkVersion), fork.PreviousVersion)
		assert.DeepEqual(t, hexutil.Encode(thirdForkVersion), fork.CurrentVersion)
		assert.Equal(t, fmt.Sprintf("%d", thirdForkEpoch), fork.Epoch)
	})
	t.Run("correct number of forks", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v1/config/fork_schedule", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		GetForkSchedule(writer, request)
		require.Equal(t, http.StatusOK, writer.Code)
		resp := &structs.GetForkScheduleResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		os := forks.NewOrderedSchedule(params.BeaconConfig())
		assert.Equal(t, os.Len(), len(resp.Data))
	})
}
