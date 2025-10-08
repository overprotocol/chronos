package checkpointsync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz/detect"
	"github.com/prysmaticlabs/prysm/v5/io/file"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var modifyFlags = struct {
	StateFile  string
	BlockFile  string
	TargetSlot uint64
	OutputDir  string
}{}

var modifyCmd = &cli.Command{
	Name:    "modify",
	Aliases: []string{"mod"},
	Usage:   "Modify checkpoint state slot for recovery purposes",
	Action: func(cliCtx *cli.Context) error {
		if err := cliActionModify(cliCtx); err != nil {
			log.WithError(err).Fatal("Could not modify checkpoint data")
		}
		return nil
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "state-file",
			Usage:       "Path to the SSZ-encoded state file",
			Destination: &modifyFlags.StateFile,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "block-file",
			Usage:       "Path to the SSZ-encoded block file",
			Destination: &modifyFlags.BlockFile,
			Required:    true,
		},
		&cli.Uint64Flag{
			Name:        "target-slot",
			Usage:       "Target slot number to set in the modified state",
			Destination: &modifyFlags.TargetSlot,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "output-dir",
			Usage:       "Output directory for modified files",
			Destination: &modifyFlags.OutputDir,
			Value:       ".",
		},
	},
}

func cliActionModify(_ *cli.Context) error {
	ctx := context.Background()
	f := modifyFlags

	log.WithFields(log.Fields{
		"stateFile":  f.StateFile,
		"blockFile":  f.BlockFile,
		"targetSlot": f.TargetSlot,
		"outputDir":  f.OutputDir,
	}).Info("Starting checkpoint modification")

	// Read state file
	stateBytes, err := file.ReadFileAsBytes(f.StateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Detect version and unmarshal state
	vu, err := detect.FromState(stateBytes)
	if err != nil {
		return fmt.Errorf("failed to detect state version: %w", err)
	}

	log.WithFields(log.Fields{
		"configName": vu.Config.ConfigName,
		"fork":       version.String(vu.Fork),
	}).Info("Detected state configuration")

	st, err := vu.UnmarshalBeaconState(stateBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	originalSlot := st.Slot()
	targetSlot := primitives.Slot(f.TargetSlot)

	// Log original checkpoint information
	origFinalized := st.FinalizedCheckpoint()
	origJustified := st.CurrentJustifiedCheckpoint()
	origPrevJustified := st.PreviousJustifiedCheckpoint()

	log.WithFields(log.Fields{
		"originalSlot":             originalSlot,
		"targetSlot":               targetSlot,
		"origFinalizedEpoch":       origFinalized.Epoch,
		"origFinalizedRoot":        fmt.Sprintf("%#x", origFinalized.Root),
		"origJustifiedEpoch":       origJustified.Epoch,
		"origJustifiedRoot":        fmt.Sprintf("%#x", origJustified.Root),
		"origPrevJustifiedEpoch":   origPrevJustified.Epoch,
		"origPrevJustifiedRoot":    fmt.Sprintf("%#x", origPrevJustified.Root),
	}).Info("Modifying state for recovery")

	// Read and process the block
	blockBytes, err := file.ReadFileAsBytes(f.BlockFile)
	if err != nil {
		return fmt.Errorf("failed to read block file: %w", err)
	}

	block, err := vu.UnmarshalBeaconBlock(blockBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal block: %w", err)
	}

	blockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return fmt.Errorf("failed to compute block root: %w", err)
	}

	// IMPORTANT: Compute checkpoint state root BEFORE modifying slot
	// This is the original state root at the checkpoint slot
	checkpointStateRoot, err := st.HashTreeRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to compute checkpoint state root: %w", err)
	}

	// Step 1: Update slot
	if err := st.SetSlot(targetSlot); err != nil {
		return fmt.Errorf("failed to set target slot: %w", err)
	}
	log.WithField("slot", targetSlot).Info("Updated slot")

	// Compute the state root at the target slot
	// This will be used as the StateRoot in LatestBlockHeader
	modifiedStateRoot, err := st.HashTreeRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to compute modified state root: %w", err)
	}

	// Step 2: Update LatestBlockHeader
	// CRITICAL: LatestBlockHeader should represent the block at slot (targetSlot - 1)
	// So that the next block (at targetSlot) can reference it as parent

	// Get the body root from the checkpoint block
	bodyRoot, err := block.Block().Body().HashTreeRoot()
	if err != nil {
		return fmt.Errorf("failed to compute body root: %w", err)
	}

	// Create a header for the "virtual" block at targetSlot - 1
	// This represents a block that would have existed at slot targetSlot-1
	// but we're using the checkpoint block's contents
	// IMPORTANT: StateRoot must match the actual state at targetSlot
	newHeader := &ethpb.BeaconBlockHeader{
		Slot:          targetSlot - 1,         // Target slot - 1
		ProposerIndex: 0,                      // Dummy proposer index
		ParentRoot:    blockRoot[:],           // Points to checkpoint block
		StateRoot:     modifiedStateRoot[:],   // Actual state root at targetSlot
		BodyRoot:      bodyRoot[:],            // Checkpoint block body
	}

	if err := st.SetLatestBlockHeader(newHeader); err != nil {
		return fmt.Errorf("failed to update latest block header: %w", err)
	}

	// Now compute what the hash of this header will be - this is what the next block will reference
	virtualBlockRoot, err := newHeader.HashTreeRoot()
	if err != nil {
		return fmt.Errorf("failed to compute virtual block root: %w", err)
	}

	log.WithFields(log.Fields{
		"headerSlot":       newHeader.Slot,
		"virtualBlockRoot": fmt.Sprintf("%#x", virtualBlockRoot),
		"parentRoot":       fmt.Sprintf("%#x", blockRoot),
	}).Info("Updated LatestBlockHeader")

	// Step 3: Update BlockRoots
	// Fill the gap between original slot and target slot
	blockRoots := st.BlockRoots()

	// Fill slots from originalSlot+1 to targetSlot-2 with checkpoint block root
	for slot := originalSlot + 1; slot < targetSlot-1; slot++ {
		index := slot % primitives.Slot(len(blockRoots))
		blockRoots[index] = blockRoot[:]
	}

	// CRITICAL: Set targetSlot-1 to point to the virtual block root
	// This is what the next block (at targetSlot) will use as parent
	virtualBlockIndex := (targetSlot - 1) % primitives.Slot(len(blockRoots))
	blockRoots[virtualBlockIndex] = virtualBlockRoot[:]

	if err := st.SetBlockRoots(blockRoots); err != nil {
		return fmt.Errorf("failed to set block roots: %w", err)
	}
	log.WithFields(log.Fields{
		"filledSlots":       targetSlot - originalSlot - 2,
		"virtualBlockIndex": virtualBlockIndex,
		"virtualBlockSlot":  targetSlot - 1,
		"virtualBlockRoot":  fmt.Sprintf("%#x", virtualBlockRoot),
	}).Info("Updated BlockRoots")

	// Step 4: Update StateRoots
	// Use checkpoint state root for all intermediate slots
	stateRoots := st.StateRoots()

	// Fill slots from originalSlot+1 to targetSlot-1 with the checkpoint state root
	for slot := originalSlot + 1; slot < targetSlot; slot++ {
		index := slot % primitives.Slot(len(stateRoots))
		stateRoots[index] = checkpointStateRoot[:]
	}

	if err := st.SetStateRoots(stateRoots); err != nil {
		return fmt.Errorf("failed to set state roots: %w", err)
	}
	log.WithFields(log.Fields{
		"filledSlots":         targetSlot - originalSlot - 1,
		"checkpointStateRoot": fmt.Sprintf("%#x", checkpointStateRoot),
	}).Info("Updated StateRoots")

	// Step 5: Update checkpoints to reference the original checkpoint block
	// CRITICAL: When InsertNode is called during checkpoint sync initialization,
	// it tries to get balances for the justified checkpoint root.
	// Since we only have the checkpoint block in DB (no other blocks),
	// the checkpoints MUST point to the actual checkpoint block root that will be saved.
	targetEpoch := primitives.Epoch(targetSlot / primitives.Slot(32)) // Assuming 32 slots per epoch

	// Update finalized checkpoint to point to the original checkpoint block
	// This is the only block that exists in the DB during checkpoint-only initialization
	newFinalizedCheckpoint := &ethpb.Checkpoint{
		Epoch: targetEpoch - 1, // Finalized is typically one epoch behind
		Root:  blockRoot[:],    // Original checkpoint block root
	}
	if err := st.SetFinalizedCheckpoint(newFinalizedCheckpoint); err != nil {
		return fmt.Errorf("failed to set finalized checkpoint: %w", err)
	}

	// Update justified checkpoints to point to the same block
	// Since we're doing checkpoint-only recovery, we use the checkpoint block for both
	newJustifiedCheckpoint := &ethpb.Checkpoint{
		Epoch: targetEpoch - 1,
		Root:  blockRoot[:], // Original checkpoint block root
	}
	if err := st.SetCurrentJustifiedCheckpoint(newJustifiedCheckpoint); err != nil {
		return fmt.Errorf("failed to set current justified checkpoint: %w", err)
	}
	if err := st.SetPreviousJustifiedCheckpoint(newJustifiedCheckpoint); err != nil {
		return fmt.Errorf("failed to set previous justified checkpoint: %w", err)
	}

	log.WithFields(log.Fields{
		"finalizedEpoch":  newFinalizedCheckpoint.Epoch,
		"finalizedRoot":   fmt.Sprintf("%#x", newFinalizedCheckpoint.Root),
		"justifiedEpoch":  newJustifiedCheckpoint.Epoch,
		"justifiedRoot":   fmt.Sprintf("%#x", newJustifiedCheckpoint.Root),
	}).Info("Updated state checkpoints")

	// Marshal modified state
	modifiedStateBytes, err := st.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("failed to marshal modified state: %w", err)
	}

	// Compute final state root for logging
	stateRoot, err := st.HashTreeRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to compute state root: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(f.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate output file names
	stateFileName := fmt.Sprintf("state.ssz")
	blockFileName := fmt.Sprintf("block_%d.ssz",
		targetSlot-1)

	stateOutputPath := filepath.Join(f.OutputDir, stateFileName)
	blockOutputPath := filepath.Join(f.OutputDir, blockFileName)

	// Write modified state
	if err := file.WriteFile(stateOutputPath, modifiedStateBytes); err != nil {
		return fmt.Errorf("failed to write modified state: %w", err)
	}
	log.WithField("path", stateOutputPath).Info("Saved modified state")

	// Modify block to match LatestBlockHeader
	// CRITICAL: Block's slot, parent_root, and state_root must match LatestBlockHeader
	// so that hash_tree_root(block) == hash_tree_root(LatestBlockHeader)
	block.SetSlot(targetSlot - 1)
	block.SetParentRoot(blockRoot[:])
	block.SetStateRoot(modifiedStateRoot[:]) // Actual state root at targetSlot

	// Verify that block root matches virtualBlockRoot
	modifiedBlockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return fmt.Errorf("failed to compute modified block root: %w", err)
	}

	log.WithFields(log.Fields{
		"blockSlot":         block.Block().Slot(),
		"modifiedBlockRoot": fmt.Sprintf("%#x", modifiedBlockRoot),
		"virtualBlockRoot":  fmt.Sprintf("%#x", virtualBlockRoot),
		"match":             modifiedBlockRoot == virtualBlockRoot,
	}).Info("Modified block")

	modifiedBlockBytes, err := block.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("failed to marshal modified block: %w", err)
	}

	// Write modified block
	if err := file.WriteFile(blockOutputPath, modifiedBlockBytes); err != nil {
		return fmt.Errorf("failed to write block: %w", err)
	}
	log.WithFields(log.Fields{
		"path":         blockOutputPath,
		"originalSlot": originalSlot,
		"modifiedSlot": targetSlot - 1,
	}).Info("Saved modified block")

	// Create metadata JSON file
	metadata := map[string]interface{}{
		"finalizedSlot":       targetSlot,
		"lastModified":        time.Now().Unix(),
		"status":              "updated",
		"blockSlot":           targetSlot - 1,
		"originalBlockSlot":   originalSlot,
		"virtualBlockRoot":    fmt.Sprintf("%#x", virtualBlockRoot),
		"checkpointBlockRoot": fmt.Sprintf("%#x", blockRoot),
		"checkpointStateRoot": fmt.Sprintf("%#x", checkpointStateRoot),
	}

	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataPath := filepath.Join(f.OutputDir, "metadata.json")
	if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	log.WithField("path", metadataPath).Info("Saved checkpoint metadata")

	log.WithFields(log.Fields{
		"originalSlot": originalSlot,
		"modifiedSlot": targetSlot,
		"stateRoot":    fmt.Sprintf("%#x", stateRoot),
		"blockRoot":    fmt.Sprintf("%#x", blockRoot),
		"stateFile":    stateOutputPath,
		"blockFile":    blockOutputPath,
		"metadataFile": metadataPath,
	}).Info("Checkpoint modification completed successfully")

	return nil
}
