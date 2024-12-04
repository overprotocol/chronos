package params

// UseDolphinNetworkConfig uses the Dolphin beacon chain specific network config.
func UseDolphinNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.BootstrapNodes = []string{
		// Dolphin testnet boot nodes
		"enr:-LG4QIKxU43azrDAd777KngiPXD2Vd1_TlIFDq33rDZ95mNuFxcO-HbetclOxx0Ofdgmr6PXC5jnglaYMLyhRcQ0ariGAZOP_nHOh2F0dG5ldHOIAAAAAAAAAACCaWSCdjSCaXCEp6xM9IRvdmVykNBNsU8AAAAY__________-Jc2VjcDI1NmsxoQOM3AK4-wojli0G8baFWHRRPMNREBTUefi7ltQJXTpRZ4N1ZHCCyyA",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// DolphinConfig defines the config for the Dolphin beacon chain testnet.
func DolphinConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.GenesisValidatorsRoot = [32]byte{197, 106, 18, 116, 72, 45, 248, 71, 14, 95, 225, 167, 251, 5, 217, 99, 189, 20, 169, 20, 11, 236, 169, 89, 239, 170, 171, 48, 115, 61, 69, 71}
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
