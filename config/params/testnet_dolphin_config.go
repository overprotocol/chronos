package params

import "math"

// UseDolphinNetworkConfig uses the Dolphin beacon chain specific network config.
func UseDolphinNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.BootstrapNodes = []string{
		// Dolphin testnet boot nodes
		"enr:-LG4QMRJx609REQqPEIEELWFqCvX94d2HuAV11dkFrxm3DG3AjRnsJZMRIrVjRajlYAH75NFonI5sMkGE9iv-CP8ccuGAZOQds9zh2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEp6xM9IRvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQP9UF-LVoudH_mS5_KrpTS-ntpgJZdaUAGVK7Rh4pG1DIN1ZHCCyyA",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// DolphinConfig defines the config for the Dolphin beacon chain testnet.
func DolphinConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.GenesisValidatorsRoot = [32]byte{152, 39, 204, 172, 238, 238, 74, 22, 247, 128, 145, 211, 206, 77, 206, 78, 69, 215, 198, 210, 121, 62, 87, 30, 139, 89, 220, 175, 243, 209, 128, 79}
	cfg.ConfigName = DolphinName
	cfg.GenesisForkVersion = []byte{0x0, 0x00, 0x00, 0x28}
	cfg.DepositChainID = 541764
	cfg.DepositNetworkID = 541764
	cfg.AltairForkEpoch = 0
	cfg.AltairForkVersion = []byte{0x1, 0x00, 0x00, 0x28}
	cfg.BellatrixForkEpoch = 0
	cfg.BellatrixForkVersion = []byte{0x2, 0x00, 0x00, 0x28}
	cfg.CapellaForkEpoch = 0
	cfg.CapellaForkVersion = []byte{0x3, 0x00, 0x00, 0x28}
	cfg.DenebForkEpoch = 0
	cfg.DenebForkVersion = []byte{0x4, 0x00, 0x00, 0x28}
	cfg.AlpacaForkEpoch = 0
	cfg.AlpacaForkVersion = []byte{0x5, 0x00, 0x00, 0x28}
	cfg.BadgerForkEpoch = math.MaxUint64
	cfg.BadgerForkVersion = []byte{0x6, 0x00, 0x00, 0x28}
	cfg.InitializeForkSchedule()
	cfg.InitializeDolphinDepositPlan()
	cfg.InitializeInactivityValues()
	return cfg
}
