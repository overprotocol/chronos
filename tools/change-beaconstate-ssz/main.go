package main

import (
	_ "embed"
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/io/file"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

var (
	inputPath  = flag.String("input-path", "", "The input file path (/path/to/old.ssz)")
	outputPath = flag.String("output-path", "", "The output file path (/path/to/new.ssz)")
	oldSSZ     []byte
	isSnappy   bool
)

var beaconStateCurrentVersion = fieldSpec{
	// 52 = 8 (genesis_time) + 32 (genesis_validators_root) + 8 (slot) + 4 (previous_version)
	offset: 52,
	t:      typeBytes4,
}

var OldBeaconConfig = &params.BeaconChainConfig{
	// Fork related values.
	GenesisForkVersion:   []byte{0, 0, 0, 10},
	AltairForkVersion:    []byte{1, 0, 0, 10},
	BellatrixForkVersion: []byte{2, 0, 0, 10},
	CapellaForkVersion:   []byte{3, 0, 0, 10},
	DenebForkVersion:     []byte{4, 0, 0, 10},
}

func main() {
	flag.Parse()
	enc, err := file.ReadFileAsBytes(*inputPath)
	if err != nil {
		panic(err)
	}
	// try decode snappy
	isSnappy = true
	oldSSZ, err = snappy.Decode(nil, enc)
	if err != nil {
		isSnappy = false
		oldSSZ = enc
	}

	currentVersion, err := beaconStateCurrentVersion.bytes4(oldSSZ)
	if err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(currentVersion[:]))

	var fork int
	switch currentVersion {
	case bytesutil.ToBytes4(OldBeaconConfig.GenesisForkVersion):
		fork = version.Phase0
	case bytesutil.ToBytes4(OldBeaconConfig.AltairForkVersion):
		fork = version.Altair
	case bytesutil.ToBytes4(OldBeaconConfig.BellatrixForkVersion):
		fork = version.Bellatrix
	case bytesutil.ToBytes4(OldBeaconConfig.CapellaForkVersion):
		fork = version.Capella
	case bytesutil.ToBytes4(OldBeaconConfig.DenebForkVersion):
		fork = version.Deneb
	case bytesutil.ToBytes4(OldBeaconConfig.ElectraForkVersion):
		fork = version.Electra

	default:
		panic("unknown fork version")
	}

	fmt.Println("Fork version:", fork)

	var output []byte
	switch fork {
	case version.Phase0:
		st := &ethpb.BeaconStateOld{}
		err = st.UnmarshalSSZ(oldSSZ)
		if err != nil {
			panic(err)
		}
		newSt := &ethpb.BeaconState{}

		// construct new state from old state
		newSt.GenesisTime = st.GenesisTime
		newSt.GenesisValidatorsRoot = st.GenesisValidatorsRoot
		newSt.Slot = st.Slot
		newSt.Fork = st.Fork
		newSt.LatestBlockHeader = st.LatestBlockHeader
		newSt.BlockRoots = st.BlockRoots
		newSt.StateRoots = st.StateRoots
		newSt.HistoricalRoots = st.HistoricalRoots
		newSt.Eth1Data = st.Eth1Data
		newSt.Eth1DataVotes = st.Eth1DataVotes
		newSt.Eth1DepositIndex = st.Eth1DepositIndex
		newSt.Validators = st.Validators
		newSt.Balances = st.Balances
		newSt.RandaoMixes = st.RandaoMixes
		newSt.Slashings = st.Slashings
		newSt.PreviousEpochAttestations = st.PreviousEpochAttestations
		newSt.CurrentEpochAttestations = st.CurrentEpochAttestations
		newSt.JustificationBits = st.JustificationBits
		newSt.PreviousJustifiedCheckpoint = st.PreviousJustifiedCheckpoint
		newSt.CurrentJustifiedCheckpoint = st.CurrentJustifiedCheckpoint
		newSt.FinalizedCheckpoint = st.FinalizedCheckpoint

		// Fill: RewardAdjustmentFactor, PreviousEpochReserve, CurrentEpochReserve
		newSt.RewardAdjustmentFactor = 0
		newSt.PreviousEpochReserve = 0
		newSt.CurrentEpochReserve = 0

		output, err = newSt.MarshalSSZ()
		if err != nil {
			panic(err)
		}

	case version.Altair:
		st := &ethpb.BeaconStateOldAltair{}
		err = st.UnmarshalSSZ(oldSSZ)
		if err != nil {
			panic(err)
		}
		newSt := &ethpb.BeaconStateAltair{}

		// construct new state from old state
		newSt.GenesisTime = st.GenesisTime
		newSt.GenesisValidatorsRoot = st.GenesisValidatorsRoot
		newSt.Slot = st.Slot
		newSt.Fork = st.Fork
		newSt.LatestBlockHeader = st.LatestBlockHeader
		newSt.BlockRoots = st.BlockRoots
		newSt.StateRoots = st.StateRoots
		newSt.HistoricalRoots = st.HistoricalRoots
		newSt.Eth1Data = st.Eth1Data
		newSt.Eth1DataVotes = st.Eth1DataVotes
		newSt.Eth1DepositIndex = st.Eth1DepositIndex
		newSt.Validators = st.Validators
		newSt.Balances = st.Balances
		newSt.RandaoMixes = st.RandaoMixes
		newSt.Slashings = st.Slashings
		newSt.JustificationBits = st.JustificationBits
		newSt.PreviousJustifiedCheckpoint = st.PreviousJustifiedCheckpoint
		newSt.CurrentJustifiedCheckpoint = st.CurrentJustifiedCheckpoint
		newSt.FinalizedCheckpoint = st.FinalizedCheckpoint

		// Fill: RewardAdjustmentFactor, PreviousEpochReserve, CurrentEpochReserve
		newSt.RewardAdjustmentFactor = 0
		newSt.PreviousEpochReserve = 0
		newSt.CurrentEpochReserve = 0

		output, err = newSt.MarshalSSZ()
		if err != nil {
			panic(err)
		}

	case version.Bellatrix:
		st := &ethpb.BeaconStateOldBellatrix{}
		err = st.UnmarshalSSZ(oldSSZ)
		if err != nil {
			panic(err)
		}
		newSt := &ethpb.BeaconStateBellatrix{}

		// construct new state from old state
		newSt.GenesisTime = st.GenesisTime
		newSt.GenesisValidatorsRoot = st.GenesisValidatorsRoot
		newSt.Slot = st.Slot
		newSt.Fork = st.Fork
		newSt.LatestBlockHeader = st.LatestBlockHeader
		newSt.BlockRoots = st.BlockRoots
		newSt.StateRoots = st.StateRoots
		newSt.HistoricalRoots = st.HistoricalRoots
		newSt.Eth1Data = st.Eth1Data
		newSt.Eth1DataVotes = st.Eth1DataVotes
		newSt.Eth1DepositIndex = st.Eth1DepositIndex
		newSt.Validators = st.Validators
		newSt.Balances = st.Balances
		newSt.RandaoMixes = st.RandaoMixes
		newSt.Slashings = st.Slashings
		newSt.JustificationBits = st.JustificationBits
		newSt.PreviousJustifiedCheckpoint = st.PreviousJustifiedCheckpoint
		newSt.CurrentJustifiedCheckpoint = st.CurrentJustifiedCheckpoint
		newSt.FinalizedCheckpoint = st.FinalizedCheckpoint

		// Fill: RewardAdjustmentFactor, PreviousEpochReserve, CurrentEpochReserve
		newSt.RewardAdjustmentFactor = 0
		newSt.PreviousEpochReserve = 0
		newSt.CurrentEpochReserve = 0

		output, err = newSt.MarshalSSZ()
		if err != nil {
			panic(err)
		}

	case version.Capella:
		st := &ethpb.BeaconStateOldCapella{}
		err = st.UnmarshalSSZ(oldSSZ)
		if err != nil {
			panic(err)
		}
		newSt := &ethpb.BeaconStateCapella{}

		// construct new state from old state
		newSt.GenesisTime = st.GenesisTime
		newSt.GenesisValidatorsRoot = st.GenesisValidatorsRoot
		newSt.Slot = st.Slot
		newSt.Fork = st.Fork
		newSt.LatestBlockHeader = st.LatestBlockHeader
		newSt.BlockRoots = st.BlockRoots
		newSt.StateRoots = st.StateRoots
		newSt.HistoricalRoots = st.HistoricalRoots
		newSt.Eth1Data = st.Eth1Data
		newSt.Eth1DataVotes = st.Eth1DataVotes
		newSt.Eth1DepositIndex = st.Eth1DepositIndex
		newSt.Validators = st.Validators
		newSt.Balances = st.Balances
		newSt.RandaoMixes = st.RandaoMixes
		newSt.Slashings = st.Slashings
		newSt.JustificationBits = st.JustificationBits
		newSt.PreviousJustifiedCheckpoint = st.PreviousJustifiedCheckpoint
		newSt.CurrentJustifiedCheckpoint = st.CurrentJustifiedCheckpoint
		newSt.FinalizedCheckpoint = st.FinalizedCheckpoint

		// Fill: RewardAdjustmentFactor, PreviousEpochReserve, CurrentEpochReserve
		newSt.RewardAdjustmentFactor = 0
		newSt.PreviousEpochReserve = 0
		newSt.CurrentEpochReserve = 0

		output, err = newSt.MarshalSSZ()
		if err != nil {
			panic(err)
		}

	// TODO: Fix Deneb and Electra
	case version.Deneb:
		st := &ethpb.BeaconStateOldDeneb{}
		err = st.UnmarshalSSZ(oldSSZ)
		if err != nil {
			panic(err)
		}
		newSt := &ethpb.BeaconStateDeneb{}

		// construct new state from old state
		newSt.GenesisTime = st.GenesisTime
		newSt.GenesisValidatorsRoot = st.GenesisValidatorsRoot
		newSt.Slot = st.Slot
		newSt.Fork = st.Fork
		newSt.LatestBlockHeader = st.LatestBlockHeader
		newSt.BlockRoots = st.BlockRoots
		newSt.StateRoots = st.StateRoots
		newSt.HistoricalRoots = st.HistoricalRoots
		newSt.Eth1Data = st.Eth1Data
		newSt.Eth1DataVotes = st.Eth1DataVotes
		newSt.Eth1DepositIndex = st.Eth1DepositIndex
		newSt.Validators = st.Validators
		newSt.Balances = st.Balances
		newSt.RandaoMixes = st.RandaoMixes
		newSt.Slashings = st.Slashings
		newSt.JustificationBits = st.JustificationBits
		newSt.PreviousJustifiedCheckpoint = st.PreviousJustifiedCheckpoint
		newSt.CurrentJustifiedCheckpoint = st.CurrentJustifiedCheckpoint
		newSt.FinalizedCheckpoint = st.FinalizedCheckpoint

		// Fill: RewardAdjustmentFactor, PreviousEpochReserve, CurrentEpochReserve
		newSt.RewardAdjustmentFactor = 0
		newSt.PreviousEpochReserve = 0
		newSt.CurrentEpochReserve = 0

		output, err = newSt.MarshalSSZ()
		if err != nil {
			panic(err)
		}

	case version.Electra:
		st := &ethpb.BeaconStateOldElectra{}
		err = st.UnmarshalSSZ(oldSSZ)
		if err != nil {
			panic(err)
		}
		newSt := &ethpb.BeaconStateElectra{}

		// construct new state from old state
		newSt.GenesisTime = st.GenesisTime
		newSt.GenesisValidatorsRoot = st.GenesisValidatorsRoot
		newSt.Slot = st.Slot
		newSt.Fork = st.Fork
		newSt.LatestBlockHeader = st.LatestBlockHeader
		newSt.BlockRoots = st.BlockRoots
		newSt.StateRoots = st.StateRoots
		newSt.HistoricalRoots = st.HistoricalRoots
		newSt.Eth1Data = st.Eth1Data
		newSt.Eth1DataVotes = st.Eth1DataVotes
		newSt.Eth1DepositIndex = st.Eth1DepositIndex
		newSt.Validators = st.Validators
		newSt.Balances = st.Balances
		newSt.RandaoMixes = st.RandaoMixes
		newSt.Slashings = st.Slashings
		newSt.JustificationBits = st.JustificationBits
		newSt.PreviousJustifiedCheckpoint = st.PreviousJustifiedCheckpoint
		newSt.CurrentJustifiedCheckpoint = st.CurrentJustifiedCheckpoint
		newSt.FinalizedCheckpoint = st.FinalizedCheckpoint

		// Fill: RewardAdjustmentFactor, PreviousEpochReserve, CurrentEpochReserve
		newSt.RewardAdjustmentFactor = 0
		newSt.PreviousEpochReserve = 0
		newSt.CurrentEpochReserve = 0

		output, err = newSt.MarshalSSZ()
		if err != nil {
			panic(err)
		}

	default:
		panic("unknown fork version")
	}

	// encode snappy
	if isSnappy {
		output = snappy.Encode(nil, output)
	}
	err = file.WriteFile(*outputPath, output)
	if err != nil {
		panic(err)
	}
}
