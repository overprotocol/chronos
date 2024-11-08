package monitor

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProcessSlashings(t *testing.T) {
	tests := []struct {
		name      string
		block     *ethpb.BeaconBlock
		wantedErr string
	}{
		{
			name: "Proposer slashing a tracked index",
			block: &ethpb.BeaconBlock{
				Body: &ethpb.BeaconBlockBody{
					ProposerSlashings: []*ethpb.ProposerSlashing{
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									ProposerIndex: 2,
									Slot:          params.BeaconConfig().SlotsPerEpoch + 1,
								},
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									ProposerIndex: 2,
									Slot:          0,
								},
							},
						},
					},
				},
			},
			wantedErr: "\"Proposer slashing was included\" bodyRoot1= bodyRoot2= prefix=monitor proposerIndex=2",
		},
		{
			name: "Proposer slashing an untracked index",
			block: &ethpb.BeaconBlock{
				Body: &ethpb.BeaconBlockBody{
					ProposerSlashings: []*ethpb.ProposerSlashing{
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									ProposerIndex: 3,
									Slot:          params.BeaconConfig().SlotsPerEpoch + 4,
								},
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									ProposerIndex: 3,
									Slot:          0,
								},
							},
						},
					},
				},
			},
			wantedErr: "",
		},
		{
			name: "Attester slashing a tracked index",
			block: &ethpb.BeaconBlock{
				Body: &ethpb.BeaconBlockBody{
					AttesterSlashings: []*ethpb.AttesterSlashing{
						{
							Attestation_1: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									Source: &ethpb.Checkpoint{Epoch: 1},
								},
								AttestingIndices: []uint64{1, 3, 4},
							}),
							Attestation_2: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
								AttestingIndices: []uint64{1, 5, 6},
							}),
						},
					},
				},
			},
			wantedErr: "\"Attester slashing was included\" attestationSlot1=0 attestationSlot2=0 attesterIndex=1 " +
				"beaconBlockRoot1=0x000000000000 beaconBlockRoot2=0x000000000000 blockInclusionSlot=0 prefix=monitor sourceEpoch1=1 sourceEpoch2=0 targetEpoch1=0 targetEpoch2=0",
		},
		{
			name: "Attester slashing untracked index",
			block: &ethpb.BeaconBlock{
				Body: &ethpb.BeaconBlockBody{
					AttesterSlashings: []*ethpb.AttesterSlashing{
						{
							Attestation_1: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									Source: &ethpb.Checkpoint{Epoch: 1},
								},
								AttestingIndices: []uint64{1, 3, 4},
							}),
							Attestation_2: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
								AttestingIndices: []uint64{3, 5, 6},
							}),
						},
					},
				},
			},
			wantedErr: "",
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
			s := &Service{
				TrackedValidators: map[primitives.ValidatorIndex]bool{
					1: true,
					2: true,
				},
			}
			wb, err := blocks.NewBeaconBlock(tt.block)
			require.NoError(t, err)
			s.processSlashings(wb)
			if tt.wantedErr != "" {
				require.LogsContain(t, hook, tt.wantedErr)
			} else {
				require.LogsDoNotContain(t, hook, "slashing")
			}
		})
	}
}

func TestProcessProposedBlock(t *testing.T) {
	tests := []struct {
		name      string
		block     *ethpb.BeaconBlock
		wantedErr string
	}{
		{
			name: "Block proposed by tracked validator",
			block: &ethpb.BeaconBlock{
				Slot:          6,
				ProposerIndex: 12,
				ParentRoot:    bytesutil.PadTo([]byte("hello-world"), 32),
				StateRoot:     bytesutil.PadTo([]byte("state-world"), 32),
				Body:          &ethpb.BeaconBlockBody{},
			},
			wantedErr: "\"Proposed beacon block was included\" balanceChange=100000000 blockRoot=0x68656c6c6f2d newBalance=256000000000 parentRoot=0x68656c6c6f2d prefix=monitor proposerIndex=12 slot=6 version=0",
		},
		{
			name: "Block proposed by untracked validator",
			block: &ethpb.BeaconBlock{
				Slot:          6,
				ProposerIndex: 13,
				ParentRoot:    bytesutil.PadTo([]byte("hello-world"), 32),
				StateRoot:     bytesutil.PadTo([]byte("state-world"), 32),
				Body:          &ethpb.BeaconBlockBody{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
			s := setupService(t)
			beaconState, _ := util.DeterministicGenesisState(t, 256)
			var root [32]byte
			copy(root[:], "hello-world")
			wb, err := blocks.NewBeaconBlock(tt.block)
			require.NoError(t, err)
			s.processProposedBlock(beaconState, root, wb)
			if tt.wantedErr != "" {
				require.LogsContain(t, hook, tt.wantedErr)
			} else {
				require.LogsDoNotContain(t, hook, "included")
			}
		})
	}

}

func TestProcessBlock_AllEventsTrackedVals(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()

	genesis, keys := util.DeterministicGenesisStateAltair(t, 64)

	genConfig := util.DefaultBlockGenConfig()
	genConfig.NumProposerSlashings = 1
	b, err := util.GenerateFullBlockAltair(genesis, keys, genConfig, 1)
	require.NoError(t, err)
	s := setupService(t)

	pubKeys := make([][]byte, 3)
	pubKeys[0] = genesis.Validators()[0].PublicKey
	pubKeys[1] = genesis.Validators()[1].PublicKey
	pubKeys[2] = genesis.Validators()[2].PublicKey

	idx := b.Block.Body.ProposerSlashings[0].Header_1.Header.ProposerIndex
	s.RLock()
	if !s.trackedIndex(idx) {
		s.TrackedValidators[idx] = true
		s.latestPerformance[idx] = ValidatorLatestPerformance{
			balance: 255900000000,
		}
		s.aggregatedPerformance[idx] = ValidatorAggregatedPerformance{}
	}
	s.RUnlock()

	root, err := b.GetBlock().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, s.config.StateGen.SaveState(ctx, root, genesis))
	wanted1 := fmt.Sprintf("\"Proposed beacon block was included\" balanceChange=100000000 blockRoot=%#x newBalance=256000000000 parentRoot=0xdc6dd5a5b6a6 prefix=monitor proposerIndex=7 slot=1 version=1", bytesutil.Trunc(root[:]))
	wanted2 := fmt.Sprintf("\"Proposer slashing was included\" bodyRoot1=0x000100000000 bodyRoot2=0x000200000000 prefix=monitor proposerIndex=%d slashingSlot=0 slot=1", idx)
	wrapped, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	s.processBlock(ctx, wrapped)
	require.LogsContain(t, hook, wanted1)
	require.LogsContain(t, hook, wanted2)
}

func TestLogAggregatedPerformance(t *testing.T) {
	hook := logTest.NewGlobal()
	latestPerformance := map[primitives.ValidatorIndex]ValidatorLatestPerformance{
		1: {
			balance: 32000000000,
		},
	}
	aggregatedPerformance := map[primitives.ValidatorIndex]ValidatorAggregatedPerformance{
		1: {
			startEpoch:          0,
			startBalance:        31700000000,
			totalAttestedCount:  12,
			totalRequestedCount: 15,
			totalDistance:       14,
			totalCorrectHead:    8,
			totalCorrectSource:  11,
			totalCorrectTarget:  12,
			totalProposedCount:  1,
		},
	}
	s := &Service{
		latestPerformance:     latestPerformance,
		aggregatedPerformance: aggregatedPerformance,
	}

	s.logAggregatedPerformance()
	wanted := "\"Aggregated performance since launch\" attestationInclusion=\"80.00%\"" +
		" averageInclusionDistance=1.2 balanceChangePct=\"0.95%\" correctlyVotedHeadPct=\"66.67%\" " +
		"correctlyVotedSourcePct=\"91.67%\" correctlyVotedTargetPct=\"100.00%\" prefix=monitor startBalance=31700000000 " +
		"startEpoch=0 totalAggregations=0 totalProposedBlocks=1 totalRequested=15 " +
		"validatorIndex=1"
	require.LogsContain(t, hook, wanted)
}
