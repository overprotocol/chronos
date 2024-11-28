package params

import (
	"math"
)

// UseDolphinNetworkConfig uses the Dolphin beacon chain specific network config.
func UseDolphinNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.BootstrapNodes = []string{
		// Dolphin testnet boot nodes
		"enr:-LG4QARHcutpnwGL1ZLHhRL6ewUXmvnUoB7ChwFwaBTw55tZeli1OWQoS2e_u8NIN86aWJMZZX-jyPXZf8p8CNhZD_GGAZNxCueXh2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEp6xX3IRvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQI6QDS1wv6ednnpG2hfN70eU_dwJyFotF6xmxsmAJQ5yYN1ZHCCyyA",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// DolphinConfig defines the config for the Dolphin beacon chain testnet.
func DolphinConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
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
	cfg.IssuanceRate = [11]uint64{20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 0}
	cfg.MaxBoostYield = [11]uint64{0, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000, 10000000000}
	cfg.InitializeForkSchedule()
	cfg.InitializeDolphinDepositPlan()
	return cfg
}
