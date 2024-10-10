package params

import (
	"math"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
)

// UnderDevnetSpecConfig retrieves the mainnet undernet config used in spec tests.
func UnderDevnetSpecConfig() *BeaconChainConfig {
	underConfig := mainnetBeaconConfig.Copy()
	// Misc
	underConfig.GenesisDelay = 30 // 5 minutes
	// Signature domains
	underConfig.DomainBeaconProposer = bytesutil.ToBytes4(bytesutil.Bytes4(0))
	underConfig.DomainBeaconAttester = bytesutil.ToBytes4(bytesutil.Bytes4(1))
	underConfig.DomainRandao = bytesutil.ToBytes4(bytesutil.Bytes4(2))
	underConfig.DomainDeposit = bytesutil.ToBytes4(hexutil.MustDecode("0x03000000"))
	underConfig.DomainVoluntaryExit = bytesutil.ToBytes4(bytesutil.Bytes4(4))
	// underConfig.GenesisForkVersion = []byte{0, 0, 0, 4}
	underConfig.GenesisForkVersion = (hexutil.MustDecode("0x20000089"))
	underConfig.DepositContractTreeDepth = 32
	underConfig.FarFutureEpoch = math.MaxUint64
	underConfig.FarFutureSlot = math.MaxUint64
	// New Altair params
	// underConfig.AltairForkVersion = []byte{1, 0, 0, 4} // Highest byte set to 0x01 to avoid collisions with mainnet versioning
	underConfig.AltairForkVersion = (hexutil.MustDecode("0x20000090"))
	underConfig.AltairForkEpoch = math.MaxUint64 - 1
	// underConfig.BellatrixForkVersion = []byte{2, 0, 0, 4}
	underConfig.BellatrixForkVersion = (hexutil.MustDecode("0x20000091"))
	underConfig.BellatrixForkEpoch = math.MaxUint64 - 1
	// underConfig.CapellaForkVersion = []byte{3, 0, 0, 4}
	underConfig.CapellaForkVersion = (hexutil.MustDecode("0x20000092"))
	underConfig.CapellaForkEpoch = math.MaxUint64 - 1
	underConfig.DenebForkVersion = (hexutil.MustDecode("0x20000093"))
	underConfig.DenebForkEpoch = math.MaxUint64 - 1
	underConfig.ElectraForkVersion = (hexutil.MustDecode("0x20000094"))
	underConfig.ElectraForkEpoch = math.MaxUint64 - 1
	// Ethereum PoW parameters.
	underConfig.DepositChainID = 181818   // Chain ID of eth1 under.
	underConfig.DepositNetworkID = 181818 // Network ID of eth1 under.
	underConfig.DepositContractAddress = "000000000000000000000000000000000beac017"
	// 2**256-2**10 for fake minimal network
	underConfig.TerminalTotalDifficulty = "0"

	underConfig.ConfigName = UnderDevnetName
	underConfig.PresetBase = "under-devnet"
	underConfig.InitializeForkSchedule()
	return underConfig
}
