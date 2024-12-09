package params

import (
	"math"
	"testing"
)

// SetupTestConfigCleanup preserves configurations allowing to modify them within tests without any
// restrictions, everything is restored after the test.
func SetupTestConfigCleanup(t testing.TB) {
	prevDefaultBeaconConfig := mainnetBeaconConfig.Copy()
	temp := configs.getActive().Copy()
	undo, err := SetActiveWithUndo(temp)
	if err != nil {
		t.Fatal(err)
	}
	prevNetworkCfg := networkConfig.Copy()
	t.Cleanup(func() {
		mainnetBeaconConfig = prevDefaultBeaconConfig
		err = undo()
		if err != nil {
			t.Fatal(err)
		}
		networkConfig = prevNetworkCfg
	})
}

// SetActiveTestCleanup sets an active config,
// and adds a test cleanup hook to revert to the default config after the test completes.
func SetActiveTestCleanup(t *testing.T, cfg *BeaconChainConfig) {
	undo, err := SetActiveWithUndo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err = undo()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func SetupForkEpochConfigForTest() {
	cfg := BeaconConfig().Copy()
	// original fork epoch
	// https://github.com/prysmaticlabs/prysm/blob/3413d05b3421c27579cf7a186cb9b142ffbb8346/config/params/mainnet_config.go#L25
	cfg.AltairForkEpoch = 72740
	cfg.BellatrixForkEpoch = 144896
	cfg.CapellaForkEpoch = 194048
	cfg.DenebForkEpoch = 269568
	cfg.AlpacaForkEpoch = math.MaxUint64
	cfg.InitializeForkSchedule()
	OverrideBeaconConfig(cfg)
}
