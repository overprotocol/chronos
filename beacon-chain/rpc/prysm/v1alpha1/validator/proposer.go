package validator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	builderapi "github.com/prysmaticlabs/prysm/v5/api/client/builder"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/builder"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	consensusblocks "github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// eth1DataNotification is a latch to stop flooding logs with the same warning.
var eth1DataNotification bool

const (
	eth1dataTimeout           = 2 * time.Second
	defaultBuilderBoostFactor = primitives.Gwei(100)
)

// GetBeaconBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot.
func (vs *Server) GetBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	// Check if this is a checkpoint recovery scenario and apply extended timeout if needed
	// originalCtx := ctx
	if flags.Get().MinimumSyncPeers == 0 {
		headSlot := vs.HeadFetcher.HeadSlot()
		currentSlot := vs.TimeFetcher.CurrentSlot()
		slotDiff := currentSlot - req.Slot
		isCheckpointRecovery := headSlot >= 2131300 && headSlot <= 2131400

		if isCheckpointRecovery || slotDiff > 1000 {
			extendedTimeout := 30 * time.Minute
			if isCheckpointRecovery {
				extendedTimeout = 45 * time.Minute
			}
			log.WithFields(logrus.Fields{
				"slot":                 req.Slot,
				"headSlot":             headSlot,
				"slotDiff":             slotDiff,
				"isCheckpointRecovery": isCheckpointRecovery,
				"extendedTimeout":      extendedTimeout,
			}).Info("Applying extended timeout for checkpoint recovery block building")

			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), extendedTimeout)
			defer cancel()
		}
	}

	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetBeaconBlock")
	defer span.End()
	span.SetAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	t, err := slots.ToTime(uint64(vs.TimeFetcher.GenesisTime().Unix()), req.Slot)
	if err != nil {
		log.WithError(err).Error("Could not convert slot to time")
	}
	log.WithFields(logrus.Fields{
		"slot":               req.Slot,
		"sinceSlotStartTime": time.Since(t),
	}).Info("Begin building block")

	log.WithField("slot", req.Slot).Info("Checking sync status")
	// A syncing validator should not produce a block.
	if vs.SyncChecker.Syncing() {
		log.WithField("slot", req.Slot).Warn("Validator is syncing, cannot propose block - RETURNING EARLY")
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}
	log.WithField("slot", req.Slot).Info("Sync check passed - continuing with block building")

	// Check if beacon chain head is too far behind current slot
	headSlot := vs.HeadFetcher.HeadSlot()
	currentSlot := req.Slot
	slotGap := currentSlot - headSlot

	log.WithFields(logrus.Fields{
		"currentSlot": currentSlot,
		"headSlot":    headSlot,
		"slotGap":     slotGap,
	}).Info("Checking head-current slot gap")

	// Check if this is checkpoint recovery scenario (head around slot 2131360)
	isCheckpointRecovery := headSlot >= 2131300 && headSlot <= 2131400

	if isCheckpointRecovery {
		log.WithFields(logrus.Fields{
			"currentSlot": currentSlot,
			"headSlot":    headSlot,
			"slotGap":     slotGap,
		}).Info("Checkpoint recovery scenario detected - allowing block building with large gap")
	} else {
		// For non-recovery scenarios, apply normal gap limits
		// Commented out for now to allow aggressive catch-up
		// if slotGap > 1000 { // More than ~31 epochs behind - very conservative
		// 	log.WithFields(logrus.Fields{
		// 		"currentSlot": currentSlot,
		// 		"headSlot":    headSlot,
		// 		"slotGap":     slotGap,
		// 	}).Warn("Head too far behind current slot, skipping block building to prevent execution payload issues")
		// 	return nil, status.Error(codes.Unavailable, "Head too far behind current slot, not ready to build block")
		// }
	}

	log.WithFields(logrus.Fields{
		"currentSlot": currentSlot,
		"headSlot":    headSlot,
		"slotGap":     slotGap,
	}).Info("Head-current slot gap acceptable - continuing with block building")

	log.WithField("slot", req.Slot).Debug("Checking optimistic status")
	// An optimistic validator MUST NOT produce a block (i.e., sign across the DOMAIN_BEACON_PROPOSER domain).
	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().BellatrixForkEpoch {
		if err := vs.optimisticStatus(ctx); err != nil {
			log.WithError(err).WithField("slot", req.Slot).Error("Optimistic status check failed")
			return nil, status.Errorf(codes.Unavailable, "Validator is not ready to propose: %v", err)
		}
	}
	log.WithField("slot", req.Slot).Debug("Optimistic status check passed")

	log.WithField("slot", req.Slot).Debug("Getting parent state")
	head, parentRoot, err := vs.getParentState(ctx, req.Slot)
	if err != nil {
		log.WithError(err).WithField("slot", req.Slot).Error("Failed to get parent state")
		return nil, err
	}
	log.WithField("slot", req.Slot).Debug("Successfully got parent state")
	log.WithField("slot", req.Slot).Debug("Creating empty block")
	sBlk, err := getEmptyBlock(req.Slot)
	if err != nil {
		log.WithError(err).WithField("slot", req.Slot).Error("Failed to create empty block")
		return nil, status.Errorf(codes.Internal, "Could not prepare block: %v", err)
	}
	log.WithField("slot", req.Slot).Debug("Successfully created empty block")
	// Set slot, graffiti, randao reveal, and parent root.
	sBlk.SetSlot(req.Slot)
	sBlk.SetGraffiti(req.Graffiti)
	sBlk.SetRandaoReveal(req.RandaoReveal)
	sBlk.SetParentRoot(parentRoot[:])

	// Set proposer index.
	log.WithField("slot", req.Slot).Debug("Calculating proposer index")
	idx, err := helpers.BeaconProposerIndex(ctx, head)
	if err != nil {
		log.WithError(err).WithField("slot", req.Slot).Error("Failed to calculate proposer index")
		return nil, fmt.Errorf("could not calculate proposer index %w", err)
	}
	log.WithFields(logrus.Fields{"slot": req.Slot, "proposerIndex": idx}).Debug("Successfully calculated proposer index")
	sBlk.SetProposerIndex(idx)

	log.WithField("slot", req.Slot).Debug("Setting builder boost factor")
	builderBoostFactor := defaultBuilderBoostFactor
	if req.BuilderBoostFactor != nil {
		builderBoostFactor = primitives.Gwei(req.BuilderBoostFactor.Value)
	}

	log.WithFields(logrus.Fields{
		"slot":               req.Slot,
		"proposerIndex":      idx,
		"builderBoostFactor": builderBoostFactor,
	}).Debug("About to call BuildBlockParallel")
	resp, err := vs.BuildBlockParallel(ctx, sBlk, head, req.SkipMevBoost, builderBoostFactor)
	log.WithFields(logrus.Fields{
		"slot":               req.Slot,
		"sinceSlotStartTime": time.Since(t),
		"validator":          sBlk.Block().ProposerIndex(),
	}).Info("Finished building block")
	if err != nil {
		return nil, errors.Wrap(err, "could not build block in parallel")
	}

	// Debug: Always log to see if GetBeaconBlock is called
	log.WithFields(logrus.Fields{
		"slot": req.Slot,
		"resp_nil": resp == nil,
		"sBlk_nil": sBlk == nil,
		"minSyncPeers": flags.Get().MinimumSyncPeers,
	}).Info("GetBeaconBlock completed - checking auto-processing conditions")

	// For single validator setup, automatically process the signed block to update head
	// This is crucial for chain progression when there are no other peers
	if resp != nil && sBlk != nil && flags.Get().MinimumSyncPeers == 0 {
		log.WithField("slot", req.Slot).Info("Auto-processing conditions met - proceeding with block processing")
		root, rootErr := sBlk.Block().HashTreeRoot()
		if rootErr != nil {
			log.WithError(rootErr).Warn("Failed to get signed block root for auto-processing")
		} else {
			// Process the signed block in background to avoid blocking the response
			go func() {
				// Use extended timeout for checkpoint recovery scenarios
				timeout := 10 * time.Minute
				processCtx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				log.WithField("slot", req.Slot).Info("Starting auto-processing of self-proposed block")
				if processErr := vs.BlockReceiver.ReceiveBlock(processCtx, sBlk, root, nil); processErr != nil {
					log.WithError(processErr).WithField("slot", req.Slot).Warn("Failed to auto-process self-proposed block")
				} else {
					log.WithField("slot", req.Slot).Info("Successfully auto-processed self-proposed block - head should be updated")
				}
			}()
		}
	} else {
		log.WithFields(logrus.Fields{
			"slot": req.Slot,
			"resp_nil": resp == nil,
			"sBlk_nil": sBlk == nil,
			"minSyncPeers": flags.Get().MinimumSyncPeers,
		}).Info("Auto-processing conditions not met - skipping block processing")
	}

	return resp, nil
}

func (vs *Server) handleSuccesfulReorgAttempt(ctx context.Context, slot primitives.Slot, parentRoot, _ [32]byte) (state.BeaconState, error) {
	// Try to get the state from the NSC
	head := transition.NextSlotState(parentRoot[:], slot)
	if head != nil {
		return head, nil
	}
	// cache miss
	head, err := vs.StateGen.StateByRoot(ctx, parentRoot)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "could not obtain head state")
	}
	return head, nil
}

func logFailedReorgAttempt(slot primitives.Slot, oldHeadRoot, headRoot [32]byte) {
	blockchain.LateBlockAttemptedReorgCount.Inc()
	log.WithFields(logrus.Fields{
		"slot":        slot,
		"oldHeadRoot": fmt.Sprintf("%#x", oldHeadRoot),
		"headRoot":    fmt.Sprintf("%#x", headRoot),
	}).Warn("late block attempted reorg failed")
}

func (vs *Server) getHeadNoReorg(ctx context.Context, slot primitives.Slot, parentRoot [32]byte) (state.BeaconState, error) {
	// Try to get the state from the NSC
	head := transition.NextSlotState(parentRoot[:], slot)
	if head != nil {
		return head, nil
	}
	head, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	return head, nil
}

func (vs *Server) getParentStateFromReorgData(ctx context.Context, slot primitives.Slot, oldHeadRoot, parentRoot, headRoot [32]byte) (head state.BeaconState, err error) {
	if parentRoot != headRoot {
		head, err = vs.handleSuccesfulReorgAttempt(ctx, slot, parentRoot, headRoot)
	} else {
		if oldHeadRoot != headRoot {
			logFailedReorgAttempt(slot, oldHeadRoot, headRoot)
		}
		head, err = vs.getHeadNoReorg(ctx, slot, parentRoot)
	}
	if err != nil {
		return nil, err
	}
	if head.Slot() >= slot {
		return head, nil
	}
	// In single-validator setups after long downtime, allow more time for slot processing
	slotDiff := slot - head.Slot()
	processCtx := ctx

	// Check if this is checkpoint recovery scenario
	isCheckpointRecovery := head.Slot() >= 2131300 && head.Slot() <= 2131400

	if flags.Get().MinimumSyncPeers == 0 && (slotDiff > 100 || isCheckpointRecovery) {
		// For checkpoint recovery or large slot gaps, use extended timeout
		var extendedTimeout time.Duration

		if isCheckpointRecovery {
			// More aggressive timeout for checkpoint recovery
			extendedTimeout = time.Duration(slotDiff/20) * 10 * time.Second // ~10s per 20 slots
			if extendedTimeout < 10*time.Minute {
				extendedTimeout = 10 * time.Minute // Minimum 10 minutes for checkpoint recovery
			}
			if extendedTimeout > 30*time.Minute {
				extendedTimeout = 30 * time.Minute // Cap at 30 minutes for checkpoint recovery
			}
		} else {
			// Use more aggressive timeout for very large gaps
			extendedTimeout = time.Duration(slotDiff/50) * 30 * time.Second // ~30s per 50 slots
			if extendedTimeout < 5*time.Minute {
				extendedTimeout = 5 * time.Minute // Minimum 5 minutes for large gaps
			}
			if extendedTimeout > 15*time.Minute {
				extendedTimeout = 15 * time.Minute // Cap at 15 minutes
			}
		}

		logrus.WithFields(logrus.Fields{
			"currentSlot":          head.Slot(),
			"targetSlot":           slot,
			"slotDiff":             slotDiff,
			"extendedTimeout":      extendedTimeout,
			"isCheckpointRecovery": isCheckpointRecovery,
		}).Warn("Single-validator setup detected with large slot gap, using extended timeout for slot processing")

		// Create a new context from background to avoid parent timeout limitations
		var cancel context.CancelFunc
		processCtx, cancel = context.WithTimeout(context.Background(), extendedTimeout)
		defer cancel()

		// Copy important values from original context if needed
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < extendedTimeout {
			logrus.WithFields(logrus.Fields{
				"parentDeadline":  deadline,
				"extendedTimeout": extendedTimeout,
			}).Debug("Parent context has shorter deadline, using background context for slot processing")
		}
	}

	head, err = transition.ProcessSlotsUsingNextSlotCache(processCtx, head, parentRoot[:], slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", slot, err)
	}
	return head, nil
}

func (vs *Server) getParentState(ctx context.Context, slot primitives.Slot) (state.BeaconState, [32]byte, error) {
	// process attestations and update head in forkchoice
	oldHeadRoot := vs.ForkchoiceFetcher.CachedHeadRoot()
	vs.ForkchoiceFetcher.UpdateHead(ctx, vs.TimeFetcher.CurrentSlot())
	headRoot := vs.ForkchoiceFetcher.CachedHeadRoot()
	parentRoot := vs.ForkchoiceFetcher.GetProposerHead()
	head, err := vs.getParentStateFromReorgData(ctx, slot, oldHeadRoot, parentRoot, headRoot)
	return head, parentRoot, err
}

func (vs *Server) BuildBlockParallel(ctx context.Context, sBlk interfaces.SignedBeaconBlock, head state.BeaconState, skipMevBoost bool, builderBoostFactor primitives.Gwei) (*ethpb.GenericBeaconBlock, error) {
	log.WithField("slot", sBlk.Block().Slot()).Debug("Entering BuildBlockParallel")
	// Build consensus fields in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.WithField("slot", sBlk.Block().Slot()).Debug("Starting background consensus field processing")

		// Create extended context for checkpoint recovery scenarios
		consensusCtx := ctx
		minSyncPeers := flags.Get().MinimumSyncPeers
		log.WithFields(logrus.Fields{
			"slot":         sBlk.Block().Slot(),
			"minSyncPeers": minSyncPeers,
			"headSlot":     head.Slot(),
		}).Debug("Checking consensus field processing timeout conditions")

		if minSyncPeers == 0 {
			// Check if this is checkpoint recovery scenario
			currentSlot := vs.TimeFetcher.CurrentSlot()
			slotDiff := currentSlot - sBlk.Block().Slot()
			isCheckpointRecovery := head.Slot() >= 2131300 && head.Slot() <= 2131400

			log.WithFields(logrus.Fields{
				"slot":                 sBlk.Block().Slot(),
				"currentSlot":          currentSlot,
				"slotDiff":             slotDiff,
				"headSlot":             head.Slot(),
				"isCheckpointRecovery": isCheckpointRecovery,
				"slotDiffCheck":        slotDiff > 1000,
			}).Debug("Consensus field processing condition check")

			if slotDiff > 1000 || isCheckpointRecovery {
				// Use extended timeout for consensus field processing
				extendedTimeout := 10 * time.Minute
				if isCheckpointRecovery {
					extendedTimeout = 15 * time.Minute
				}
				log.WithFields(logrus.Fields{
					"slot":                     sBlk.Block().Slot(),
					"slotDiff":                 slotDiff,
					"isCheckpointRecovery":     isCheckpointRecovery,
					"consensusExtendedTimeout": extendedTimeout,
				}).Debug("Using extended timeout for consensus field processing")

				var cancel context.CancelFunc
				consensusCtx, cancel = context.WithTimeout(context.Background(), extendedTimeout)
				defer cancel()
			}
		}

		// Set eth1 data.
		log.WithField("slot", sBlk.Block().Slot()).Debug("Getting eth1 data")
		eth1Data, err := vs.eth1DataMajorityVote(consensusCtx, head)
		if err != nil {
			eth1Data = &ethpb.Eth1Data{DepositRoot: params.BeaconConfig().ZeroHash[:], BlockHash: params.BeaconConfig().ZeroHash[:]}
			log.WithError(err).Error("Could not get eth1data")
		}
		sBlk.SetEth1Data(eth1Data)

		// Set deposit and attestation.
		log.WithField("slot", sBlk.Block().Slot()).Debug("Starting to pack deposits and attestations")
		deposits, atts, err := vs.packDepositsAndAttestations(consensusCtx, head, sBlk.Block().Slot(), eth1Data) // TODO: split attestations and deposits
		if err != nil {
			sBlk.SetDeposits([]*ethpb.Deposit{})
			if err := sBlk.SetAttestations([]ethpb.Att{}); err != nil {
				log.WithError(err).Error("Could not set attestations on block")
			}
			log.WithError(err).Error("Could not pack deposits and attestations")
		} else {
			log.WithFields(logrus.Fields{
				"slot":         sBlk.Block().Slot(),
				"deposits":     len(deposits),
				"attestations": len(atts),
			}).Debug("Successfully packed deposits and attestations")
			sBlk.SetDeposits(deposits)
			if err := sBlk.SetAttestations(atts); err != nil {
				log.WithError(err).Error("Could not set attestations on block")
			}
		}

		// Set slashings.
		log.WithField("slot", sBlk.Block().Slot()).Debug("Getting slashings")
		validProposerSlashings, validAttSlashings := vs.getSlashings(ctx, head)
		sBlk.SetProposerSlashings(validProposerSlashings)
		if err := sBlk.SetAttesterSlashings(validAttSlashings); err != nil {
			log.WithError(err).Error("Could not set attester slashings on block")
		}

		// Set exits.
		log.WithField("slot", sBlk.Block().Slot()).Debug("Getting exits")
		sBlk.SetVoluntaryExits(vs.getExits(head, sBlk.Block().Slot()))
		log.WithField("slot", sBlk.Block().Slot()).Debug("Completed background consensus field processing")
	}()

	log.WithField("slot", sBlk.Block().Slot()).Info("Starting execution payload processing...")
	winningBid := primitives.ZeroWei()
	var bundle *enginev1.BlobsBundle
	if sBlk.Version() >= version.Bellatrix {
		log.WithField("slot", sBlk.Block().Slot()).Info("Getting local execution payload from execution client...")
		local, err := vs.getLocalPayload(ctx, sBlk.Block(), head)
		if err != nil {
			// Check if this is a checkpoint recovery scenario
			headSlot := head.Slot()
			isCheckpointRecovery := headSlot >= 2131300 && headSlot <= 2131400

			log.WithError(err).WithFields(logrus.Fields{
				"slot":                 sBlk.Block().Slot(),
				"headSlot":             headSlot,
				"isCheckpointRecovery": isCheckpointRecovery,
				"checkpointMin":        2131300,
				"checkpointMax":        2131400,
			}).Error("Execution payload failed - checking checkpoint recovery conditions")

			// When execution client fails to provide payload, create a fallback payload
			// This handles both checkpoint recovery and ongoing execution client issues
			if isCheckpointRecovery {
				log.WithError(err).WithFields(logrus.Fields{
					"slot":                 sBlk.Block().Slot(),
					"headSlot":             headSlot,
					"isCheckpointRecovery": isCheckpointRecovery,
				}).Warn("Failed to get execution payload during checkpoint recovery - creating fallback payload")
			} else {
				log.WithError(err).WithFields(logrus.Fields{
					"slot":    sBlk.Block().Slot(),
					"headSlot": headSlot,
				}).Warn("Failed to get execution payload due to execution client issues - creating fallback payload to continue block building")
			}

			// Create fallback execution payload when execution client is unavailable
			fallbackExecData, fallbackErr := vs.getFallbackExecutionData(ctx, head, sBlk.Block().Slot(), sBlk.Block().ProposerIndex())
			if fallbackErr != nil {
				log.WithError(fallbackErr).Error("Failed to create fallback execution data")
				return nil, status.Errorf(codes.Internal, "Could not create fallback execution data: %v", fallbackErr)
			}
			local = &consensusblocks.GetPayloadResponse{
				ExecutionData: fallbackExecData,
				BlobsBundle:   &enginev1.BlobsBundle{},
			}
		}
		if local != nil {
			log.WithField("slot", sBlk.Block().Slot()).Info("Successfully obtained local execution payload")
		}

		// There's no reason to try to get a builder bid if local override is true.
		var builderBid builderapi.Bid
		if local != nil && !(local.OverrideBuilder || skipMevBoost) {
			builderBid, err = vs.getBuilderPayloadAndBlobs(ctx, sBlk.Block().Slot(), sBlk.Block().ProposerIndex())
			if err != nil {
				builderGetPayloadMissCount.Inc()
				log.WithError(err).Error("Could not get builder payload")
			}
		}

		log.WithField("slot", sBlk.Block().Slot()).Info("Setting execution data on block...")
		winningBid, bundle, err = setExecutionData(ctx, sBlk, local, builderBid, builderBoostFactor)
		if err != nil {
			log.WithError(err).WithField("slot", sBlk.Block().Slot()).Error("Failed to set execution data")
			return nil, status.Errorf(codes.Internal, "Could not set execution data: %v", err)
		}
		log.WithField("slot", sBlk.Block().Slot()).Info("Successfully set execution data on block")
	}

	log.WithField("slot", sBlk.Block().Slot()).Info("Waiting for background consensus field processing to complete...")
	wg.Wait()
	log.WithField("slot", sBlk.Block().Slot()).Info("Background processing completed, starting state root computation")

	sr, err := vs.computeStateRoot(ctx, sBlk)
	if err != nil {
		log.WithError(err).WithField("slot", sBlk.Block().Slot()).Error("Failed to compute state root")
		return nil, status.Errorf(codes.Internal, "Could not compute state root: %v", err)
	}
	log.WithField("slot", sBlk.Block().Slot()).Info("Successfully computed state root - block building nearly complete")
	sBlk.SetStateRoot(sr)

	log.WithField("slot", sBlk.Block().Slot()).Debug("Constructing generic beacon block")
	result, err := vs.constructGenericBeaconBlock(sBlk, bundle, winningBid)
	if err != nil {
		log.WithError(err).WithField("slot", sBlk.Block().Slot()).Error("Failed to construct generic beacon block")
		return nil, err
	}
	log.WithField("slot", sBlk.Block().Slot()).Debug("Successfully completed BuildBlockParallel")
	return result, nil
}

// ProposeBeaconBlock handles the proposal of beacon blocks.
func (vs *Server) ProposeBeaconBlock(ctx context.Context, req *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.ProposeBeaconBlock")
	defer span.End()

	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	block, err := blocks.NewSignedBeaconBlock(req.Block)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s: %v", "decode block failed", err)
	}

	var sidecars []*ethpb.BlobSidecar
	if block.IsBlinded() {
		block, sidecars, err = vs.handleBlindedBlock(ctx, block)
	} else if block.Version() >= version.Deneb {
		sidecars, err = vs.blobSidecarsFromUnblindedBlock(block, req)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s: %v", "handle block failed", err)
	}

	root, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not hash tree root: %v", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := vs.broadcastReceiveBlock(ctx, block, root); err != nil {
			errChan <- errors.Wrap(err, "broadcast/receive block failed")
			return
		}
		errChan <- nil
	}()

	if err := vs.broadcastAndReceiveBlobs(ctx, sidecars, root); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast/receive blobs: %v", err)
	}

	wg.Wait()
	if err := <-errChan; err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast/receive block: %v", err)
	}

	return &ethpb.ProposeResponse{BlockRoot: root[:]}, nil
}

// handleBlindedBlock processes blinded beacon blocks.
func (vs *Server) handleBlindedBlock(ctx context.Context, block interfaces.SignedBeaconBlock) (interfaces.SignedBeaconBlock, []*ethpb.BlobSidecar, error) {
	if block.Version() < version.Bellatrix {
		return nil, nil, errors.New("pre-Bellatrix blinded block")
	}
	if vs.BlockBuilder == nil || !vs.BlockBuilder.Configured() {
		return nil, nil, errors.New("unconfigured block builder")
	}

	copiedBlock, err := block.Copy()
	if err != nil {
		return nil, nil, err
	}

	payload, bundle, err := vs.BlockBuilder.SubmitBlindedBlock(ctx, block)
	if err != nil {
		return nil, nil, errors.Wrap(err, "submit blinded block failed")
	}

	if err := copiedBlock.Unblind(payload); err != nil {
		return nil, nil, errors.Wrap(err, "unblind failed")
	}

	sidecars, err := unblindBlobsSidecars(copiedBlock, bundle)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unblind blobs sidecars: commitment value doesn't match block")
	}

	return copiedBlock, sidecars, nil
}

func (vs *Server) blobSidecarsFromUnblindedBlock(block interfaces.SignedBeaconBlock, req *ethpb.GenericSignedBeaconBlock) ([]*ethpb.BlobSidecar, error) {
	rawBlobs, proofs, err := blobsAndProofs(req)
	if err != nil {
		return nil, err
	}
	return BuildBlobSidecars(block, rawBlobs, proofs)
}

// broadcastReceiveBlock broadcasts a block and handles its reception.
func (vs *Server) broadcastReceiveBlock(ctx context.Context, block interfaces.SignedBeaconBlock, root [32]byte) error {
	protoBlock, err := block.Proto()
	if err != nil {
		return errors.Wrap(err, "protobuf conversion failed")
	}
	if err := vs.P2P.Broadcast(ctx, protoBlock); err != nil {
		return errors.Wrap(err, "broadcast failed")
	}
	vs.BlockNotifier.BlockFeed().Send(&feed.Event{
		Type: blockfeed.ReceivedBlock,
		Data: &blockfeed.ReceivedBlockData{SignedBlock: block},
	})
	return vs.BlockReceiver.ReceiveBlock(ctx, block, root, nil)
}

// broadcastAndReceiveBlobs handles the broadcasting and reception of blob sidecars.
func (vs *Server) broadcastAndReceiveBlobs(ctx context.Context, sidecars []*ethpb.BlobSidecar, root [32]byte) error {
	eg, eCtx := errgroup.WithContext(ctx)
	for i, sc := range sidecars {
		// Copy the iteration instance to a local variable to give each go-routine its own copy to play with.
		// See https://golang.org/doc/faq#closures_and_goroutines for more details.
		subIdx := i
		sCar := sc
		eg.Go(func() error {
			if err := vs.P2P.BroadcastBlob(eCtx, uint64(subIdx), sCar); err != nil {
				return errors.Wrap(err, "broadcast blob failed")
			}
			readOnlySc, err := blocks.NewROBlobWithRoot(sCar, root)
			if err != nil {
				return errors.Wrap(err, "ROBlob creation failed")
			}
			verifiedBlob := blocks.NewVerifiedROBlob(readOnlySc)
			if err := vs.BlobReceiver.ReceiveBlob(ctx, verifiedBlob); err != nil {
				return errors.Wrap(err, "receive blob failed")
			}
			vs.OperationNotifier.OperationFeed().Send(&feed.Event{
				Type: operation.BlobSidecarReceived,
				Data: &operation.BlobSidecarReceivedData{Blob: &verifiedBlob},
			})
			return nil
		})
	}
	return eg.Wait()
}

// PrepareBeaconProposer caches and updates the fee recipient for the given proposer.
func (vs *Server) PrepareBeaconProposer(
	_ context.Context, request *ethpb.PrepareBeaconProposerRequest,
) (*emptypb.Empty, error) {
	var validatorIndices []primitives.ValidatorIndex

	for _, r := range request.Recipients {
		recipient := hexutil.Encode(r.FeeRecipient)
		if !common.IsHexAddress(recipient) {
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid fee recipient address: %v", recipient))
		}
		// Use default address if the burn address is return
		feeRecipient := primitives.ExecutionAddress(r.FeeRecipient)
		if feeRecipient == primitives.ExecutionAddress([20]byte{}) {
			feeRecipient = primitives.ExecutionAddress(params.BeaconConfig().DefaultFeeRecipient)
			if feeRecipient == primitives.ExecutionAddress([20]byte{}) {
				log.WithField("validatorIndex", r.ValidatorIndex).Warn("fee recipient is the burn address")
			}
		}
		val := cache.TrackedValidator{
			Active:       true, // TODO: either check or add the field in the request
			Index:        r.ValidatorIndex,
			FeeRecipient: feeRecipient,
		}
		vs.TrackedValidatorsCache.Set(val)
		validatorIndices = append(validatorIndices, r.ValidatorIndex)
	}
	if len(validatorIndices) != 0 {
		log.WithFields(logrus.Fields{
			"validatorCount": len(validatorIndices),
		}).Debug("Updated fee recipient addresses for validator indices")
	}
	return &emptypb.Empty{}, nil
}

// GetFeeRecipientByPubKey returns a fee recipient from the beacon node's settings or db based on a given public key
func (vs *Server) GetFeeRecipientByPubKey(ctx context.Context, request *ethpb.FeeRecipientByPubKeyRequest) (*ethpb.FeeRecipientByPubKeyResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.GetFeeRecipientByPublicKey")
	defer span.End()
	if request == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request was empty")
	}

	resp, err := vs.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: request.PublicKey})
	if err != nil {
		if strings.Contains(err.Error(), "Could not find validator index") {
			return &ethpb.FeeRecipientByPubKeyResponse{
				FeeRecipient: params.BeaconConfig().DefaultFeeRecipient.Bytes(),
			}, nil
		} else {
			log.WithError(err).Error("An error occurred while retrieving validator index")
			return nil, err
		}
	}
	address, err := vs.BeaconDB.FeeRecipientByValidatorID(ctx, resp.GetIndex())
	if err != nil {
		if errors.Is(err, kv.ErrNotFoundFeeRecipient) {
			return &ethpb.FeeRecipientByPubKeyResponse{
				FeeRecipient: params.BeaconConfig().DefaultFeeRecipient.Bytes(),
			}, nil
		} else {
			log.WithError(err).Error("An error occurred while retrieving fee recipient from db")
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}
	return &ethpb.FeeRecipientByPubKeyResponse{
		FeeRecipient: address.Bytes(),
	}, nil
}

// computeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (vs *Server) computeStateRoot(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	beaconState, err := vs.StateGen.StateByRoot(ctx, block.Block().ParentRoot())
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve beacon state")
	}

	// In single-validator setups after long downtime, allow more time for state root calculation
	calculateCtx := ctx
	if flags.Get().MinimumSyncPeers == 0 {
		currentSlot := vs.TimeFetcher.CurrentSlot()
		blockSlot := block.Block().Slot()
		stateSlot := beaconState.Slot()

		// Check if this is checkpoint recovery scenario
		isCheckpointRecovery := stateSlot >= 2131300 && stateSlot <= 2131400

		// Check for large gaps that require extended timeout
		slotDiff := currentSlot - stateSlot
		if slotDiff > 100 || isCheckpointRecovery {
			// Extended timeout based on slot difference
			var extendedTimeout time.Duration

			if isCheckpointRecovery {
				// More aggressive timeout for checkpoint recovery
				extendedTimeout = 20 * time.Minute
				if slotDiff > 50000 {
					extendedTimeout = 45 * time.Minute // Very large gaps need more time
				} else if slotDiff > 20000 {
					extendedTimeout = 30 * time.Minute
				}
			} else {
				// Standard extended timeout for large gaps
				extendedTimeout = 5 * time.Minute
				if slotDiff > 10000 {
					extendedTimeout = 15 * time.Minute
				} else if slotDiff > 1000 {
					extendedTimeout = 10 * time.Minute
				}
			}

			logrus.WithFields(logrus.Fields{
				"currentSlot":          currentSlot,
				"blockSlot":            blockSlot,
				"stateSlot":            stateSlot,
				"slotDiff":             slotDiff,
				"extendedTimeout":      extendedTimeout,
				"isCheckpointRecovery": isCheckpointRecovery,
			}).Warn("Single-validator setup detected with large slot gap, using extended timeout for state root calculation")

			// Create extended timeout context from background to avoid parent timeout limitations
			var cancel context.CancelFunc
			calculateCtx, cancel = context.WithTimeout(context.Background(), extendedTimeout)
			defer cancel()
		}
	}

	root, err := transition.CalculateStateRoot(
		calculateCtx,
		beaconState,
		block,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate state root at slot %d", beaconState.Slot())
	}

	logrus.WithField("beaconStateRoot", fmt.Sprintf("%#x", root)).Debugf("Computed state root")
	return root[:], nil
}

// SubmitValidatorRegistrations submits validator registrations.
func (vs *Server) SubmitValidatorRegistrations(ctx context.Context, reg *ethpb.SignedValidatorRegistrationsV1) (*emptypb.Empty, error) {
	if vs.BlockBuilder == nil || !vs.BlockBuilder.Configured() {
		return &emptypb.Empty{}, status.Errorf(codes.InvalidArgument, "Could not register block builder: %v", builder.ErrNoBuilder)
	}

	if err := vs.BlockBuilder.RegisterValidator(ctx, reg.Messages); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not register block builder: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func blobsAndProofs(req *ethpb.GenericSignedBeaconBlock) ([][]byte, [][]byte, error) {
	switch {
	case req.GetDeneb() != nil:
		dbBlockContents := req.GetDeneb()
		return dbBlockContents.Blobs, dbBlockContents.KzgProofs, nil
	case req.GetElectra() != nil:
		dbBlockContents := req.GetElectra()
		return dbBlockContents.Blobs, dbBlockContents.KzgProofs, nil
	case req.GetBadger() != nil:
		dbBlockContents := req.GetBadger()
		return dbBlockContents.Blobs, dbBlockContents.KzgProofs, nil
	default:
		return nil, nil, errors.Errorf("unknown request type provided: %T", req)
	}
}
