package params

import (
	"math"
)

// UseDolphinNetworkConfig uses the Dolphin beacon chain specific network config.
func UseDolphinNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.BootstrapNodes = []string{
		// Dolphin testnet boot nodes
		"enr:-LG4QGaXDTDc5_-AvUXuWxoYlT2Ce9dSlLi4Kx0Wzv7PFBSFWqRubay-w-IY5lay30YpEbP6_yNQtXa1QcrRD1PSdYqGAZFLTRaKh2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEgMdLF4RvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQOr1euFU8IZdyGo8jbIzJD0Z8VcRnt9xrIF-aOrRvQjPYN1ZHCCyyA",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// DolphinConfig defines the config for the Dolphin beacon chain testnet.
func DolphinConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.GenesisValidatorsRoot = [32]byte{72, 53, 56, 66, 146, 92, 179, 239, 84, 134, 155, 20, 196, 84, 186, 245, 125, 16, 110, 201, 247, 155, 198, 125, 119, 186, 120, 204, 89, 247, 6, 37}
	cfg.ConfigName = DolphinName
	cfg.GenesisForkVersion = []byte{0x0, 0x00, 0x00, 0x28}
	cfg.DepositChainID = 541764
	cfg.DepositNetworkID = 541764
	cfg.AltairForkEpoch = 0
	cfg.AltairForkVersion = []byte{0x1, 0x00, 0x00, 0x28}
	cfg.BellatrixForkEpoch = 0
	cfg.BellatrixForkVersion = []byte{0x2, 0x00, 0x00, 0x28}
	cfg.CapellaForkEpoch = 10
	cfg.CapellaForkVersion = []byte{0x3, 0x00, 0x00, 0x28}
	cfg.DenebForkEpoch = math.MaxUint64
	cfg.DenebForkVersion = []byte{0x4, 0x00, 0x00, 0x28}
	cfg.ElectraForkEpoch = math.MaxUint64
	cfg.ElectraForkVersion = []byte{0x5, 0x00, 0x00, 0x28}
	cfg.IssuanceRate = [11]uint64{20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 0}
	cfg.MaxBoostYield = [11]uint64{0, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000}
	cfg.InitializeForkSchedule()
	cfg.InitializeDolphinDepositPlan()
	return cfg
}
