package params

// UseDolphinNetworkConfig uses the Dolphin beacon chain specific network config.
func UseDolphinNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.BootstrapNodes = []string{
		// Dolphin testnet boot nodes
		"enr:-LG4QA6NXs27TzpZbIC97Ksesg8dW-Y_CbKlHeqHyb7bLdxEYbSsaVhjVjfqgEtokkPOLzO1FGFMsN0c9DeOFuLGAMaGAZOQVo7uh2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEp6xM9IRvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQJvTlHWJvTBS0OJ7cEqydw3RLR67DNiQAkR_g0XoABu2IN1ZHCCyyA",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// DolphinConfig defines the config for the Dolphin beacon chain testnet.
func DolphinConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.GenesisValidatorsRoot = [32]byte{180, 154, 221, 57, 193, 238, 8, 239, 30, 212, 34, 252, 18, 14, 158, 200, 167, 131, 219, 243, 181, 118, 69, 47, 246, 172, 83, 107, 150, 104, 235, 145}
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
	cfg.InitializeForkSchedule()
	cfg.InitializeDolphinDepositPlan()
	cfg.InitializeInactivityValues()
	return cfg
}
