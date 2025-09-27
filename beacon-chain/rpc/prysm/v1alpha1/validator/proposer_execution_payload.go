package validator

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/api/client/builder"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	payloadattribute "github.com/prysmaticlabs/prysm/v5/consensus-types/payload-attribute"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
)

var (
	// payloadIDCacheMiss tracks the number of payload ID requests that aren't present in the cache.
	payloadIDCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payload_id_cache_miss",
		Help: "The number of payload id get requests that aren't present in the cache.",
	})
	// payloadIDCacheHit tracks the number of payload ID requests that are present in the cache.
	payloadIDCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payload_id_cache_hit",
		Help: "The number of payload id get requests that are present in the cache.",
	})
)

func setFeeRecipientIfBurnAddress(val *cache.TrackedValidator) {
	if val.FeeRecipient == primitives.ExecutionAddress([20]byte{}) && val.Index == 0 {
		val.FeeRecipient = primitives.ExecutionAddress(params.BeaconConfig().DefaultFeeRecipient)
	}
}

// This returns the local execution payload of a given slot. The function has full awareness of pre and post merge.
func (vs *Server) getLocalPayload(ctx context.Context, blk interfaces.ReadOnlyBeaconBlock, st state.BeaconState) (*consensusblocks.GetPayloadResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.getLocalPayload")
	defer span.End()

	if blk.Version() < version.Bellatrix {
		return nil, nil
	}

	slot := blk.Slot()
	vIdx := blk.ProposerIndex()
	headRoot := blk.ParentRoot()

	return vs.getLocalPayloadFromEngine(ctx, st, headRoot, slot, vIdx)
}

// This returns the local execution payload of a slot, proposer ID, and parent root assuming payload Is cached.
// If the payload ID is not cached, the function will prepare a new payload through local EL engine and return it by using the head state.
func (vs *Server) getLocalPayloadFromEngine(
	ctx context.Context,
	st state.BeaconState,
	parentRoot [32]byte,
	slot primitives.Slot,
	proposerId primitives.ValidatorIndex) (*consensusblocks.GetPayloadResponse, error) {
	logFields := logrus.Fields{
		"validatorIndex": proposerId,
		"slot":           slot,
		"headRoot":       fmt.Sprintf("%#x", parentRoot),
	}
	payloadId, ok := vs.PayloadIDCache.PayloadID(slot, parentRoot)

	val, tracked := vs.TrackedValidatorsCache.Validator(proposerId)
	if !tracked {
		logrus.WithFields(logFields).Warn("could not find tracked proposer index")
	}
	setFeeRecipientIfBurnAddress(&val)

	if ok && payloadId != [8]byte{} {
		// Payload ID is cache hit. Return the cached payload ID.
		var pid primitives.PayloadID
		copy(pid[:], payloadId[:])
		payloadIDCacheHit.Inc()
		res, err := vs.ExecutionEngineCaller.GetPayload(ctx, pid, slot)
		if err == nil {
			warnIfFeeRecipientDiffers(val.FeeRecipient[:], res.ExecutionData.FeeRecipient())
			return res, nil
		}
		// TODO: TestServer_getExecutionPayloadContextTimeout expects this behavior.
		// We need to figure out if it is actually important to "retry" by falling through to the code below when
		// we get a timeout when trying to retrieve the cached payload id.
		if !errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.Wrap(err, "could not get cached payload from execution client")
		}
	}
	log.WithFields(logFields).Debug("Payload ID cache miss")
	parentHash, err := vs.getParentBlockHash(ctx, st, slot)
	switch {
	case errors.Is(err, errActivationNotReached) || errors.Is(err, errNoTerminalBlockHash):
		return consensusblocks.NewGetPayloadResponse(emptyPayload())
	case err != nil:
		return nil, err
	}
	payloadIDCacheMiss.Inc()

	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return nil, err
	}

	finalizedBlockHash := [32]byte{}
	justifiedBlockHash := [32]byte{}
	// Blocks before Bellatrix don't have execution payloads. Use zeros as the hash.
	if st.Version() >= version.Bellatrix {
		finalizedBlockHash = vs.FinalizationFetcher.FinalizedBlockHash()
		justifiedBlockHash = vs.FinalizationFetcher.UnrealizedJustifiedPayloadBlockHash()
	}

	f := &enginev1.ForkchoiceState{
		HeadBlockHash:      parentHash,
		SafeBlockHash:      justifiedBlockHash[:],
		FinalizedBlockHash: finalizedBlockHash[:],
	}

	t, err := slots.ToTime(st.GenesisTime(), slot)
	if err != nil {
		return nil, err
	}
	var attr payloadattribute.Attributer
	switch st.Version() {
	case version.Deneb, version.Alpaca, version.Badger:
		withdrawals, _, _, err := st.ExpectedWithdrawals()
		if err != nil {
			return nil, err
		}
		attr, err = payloadattribute.New(&enginev1.PayloadAttributesV3{
			Timestamp:             uint64(t.Unix()),
			PrevRandao:            random,
			SuggestedFeeRecipient: val.FeeRecipient[:],
			Withdrawals:           withdrawals,
			ParentBeaconBlockRoot: parentRoot[:],
		})
		if err != nil {
			return nil, err
		}
	case version.Capella:
		withdrawals, _, _, err := st.ExpectedWithdrawals()
		if err != nil {
			return nil, err
		}
		attr, err = payloadattribute.New(&enginev1.PayloadAttributesV2{
			Timestamp:             uint64(t.Unix()),
			PrevRandao:            random,
			SuggestedFeeRecipient: val.FeeRecipient[:],
			Withdrawals:           withdrawals,
		})
		if err != nil {
			return nil, err
		}
	case version.Bellatrix:
		attr, err = payloadattribute.New(&enginev1.PayloadAttributes{
			Timestamp:             uint64(t.Unix()),
			PrevRandao:            random,
			SuggestedFeeRecipient: val.FeeRecipient[:],
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown beacon state version")
	}
	payloadID, _, err := vs.ExecutionEngineCaller.ForkchoiceUpdated(ctx, f, attr)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare payload")
	}
	if payloadID == nil {
		return nil, fmt.Errorf("nil payload with block hash: %#x", parentHash)
	}
	res, err := vs.ExecutionEngineCaller.GetPayload(ctx, *payloadID, slot)
	if err != nil {
		return nil, err
	}

	warnIfFeeRecipientDiffers(val.FeeRecipient[:], res.ExecutionData.FeeRecipient())
	log.WithField("value", res.Bid).Debug("Received execution payload from local engine")
	return res, nil
}

// warnIfFeeRecipientDiffers logs a warning if the fee recipient in the payload (eg the EL engine get payload response) does not
// match what was expected (eg the fee recipient previously used to request preparation of the payload).
func warnIfFeeRecipientDiffers(want, got []byte) {
	if !bytes.Equal(want, got) {
		logrus.WithFields(logrus.Fields{
			"wantedFeeRecipient": fmt.Sprintf("%#x", want),
			"received":           fmt.Sprintf("%#x", got),
		}).Warn("Fee recipient address from execution client is not what was expected. " +
			"It is possible someone has compromised your client to try and take your transaction fees")
	}
}

// This returns the valid terminal block hash with an existence bool value.
//
// Spec code:
// def get_terminal_pow_block(pow_chain: Dict[Hash32, PowBlock]) -> Optional[PowBlock]:
//
//	if TERMINAL_BLOCK_HASH != Hash32():
//	    # Terminal block hash override takes precedence over terminal total difficulty
//	    if TERMINAL_BLOCK_HASH in pow_chain:
//	        return pow_chain[TERMINAL_BLOCK_HASH]
//	    else:
//	        return None
//
//	return get_pow_block_at_terminal_total_difficulty(pow_chain)
func (vs *Server) getTerminalBlockHashIfExists(ctx context.Context, transitionTime uint64) ([]byte, bool, error) {
	terminalBlockHash := params.BeaconConfig().TerminalBlockHash
	// Terminal block hash override takes precedence over terminal total difficulty.
	if params.BeaconConfig().TerminalBlockHash != params.BeaconConfig().ZeroHash {
		exists, _, err := vs.Eth1BlockFetcher.BlockExists(ctx, terminalBlockHash)
		if err != nil {
			return nil, false, err
		}
		if !exists {
			return nil, false, nil
		}

		return terminalBlockHash.Bytes(), true, nil
	}

	return vs.ExecutionEngineCaller.GetTerminalBlockHash(ctx, transitionTime)
}

func (vs *Server) getBuilderPayloadAndBlobs(ctx context.Context,
	slot primitives.Slot,
	vIdx primitives.ValidatorIndex) (builder.Bid, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.getBuilderPayloadAndBlobs")
	defer span.End()

	if slots.ToEpoch(slot) < params.BeaconConfig().BellatrixForkEpoch {
		return nil, nil
	}
	canUseBuilder, err := vs.canUseBuilder(ctx, slot, vIdx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if we can use the builder")
	}
	span.SetAttributes(trace.BoolAttribute("canUseBuilder", canUseBuilder))
	if !canUseBuilder {
		return nil, nil
	}

	return vs.getPayloadHeaderFromBuilder(ctx, slot, vIdx)
}

var errActivationNotReached = errors.New("activation epoch not reached")
var errNoTerminalBlockHash = errors.New("no terminal block hash")

// getParentBlockHash retrieves the parent block hash of the block at the given slot.
// The function's behavior varies depending on the state version and whether the merge has been completed.
//
// For states of version Capella or later, the block hash is directly retrieved from the state's latest execution payload header.
//
// If the merge transition has been completed, the parent block hash is also retrieved from the state's latest execution payload header.
//
// If the activation epoch has not been reached, an errActivationNotReached error is returned.
//
// Otherwise, the terminal block hash is fetched based on the slot's time, and an error is returned if it doesn't exist.
func (vs *Server) getParentBlockHash(ctx context.Context, st state.BeaconState, slot primitives.Slot) ([]byte, error) {
	if st.Version() >= version.Capella {
		return getParentBlockHashPostCapella(st)
	}

	mergeComplete, err := blocks.IsMergeTransitionComplete(st)
	if err != nil {
		return nil, err
	}
	if mergeComplete {
		return getParentBlockHashPostMerge(st)
	}

	if activationEpochNotReached(slot) {
		return nil, errActivationNotReached
	}

	return getParentBlockHashPreMerge(ctx, vs, st, slot)
}

// getParentBlockHashPostCapella retrieves the parent block hash for states of version Capella or later.
func getParentBlockHashPostCapella(st state.BeaconState) ([]byte, error) {
	header, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "could not get post capella payload header")
	}
	return header.BlockHash(), nil
}

// getParentBlockHashPostMerge retrieves the parent block hash after the merge has completed.
func getParentBlockHashPostMerge(st state.BeaconState) ([]byte, error) {
	header, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "could not get post merge payload header")
	}
	return header.BlockHash(), nil
}

// getParentBlockHashPreMerge retrieves the parent block hash before the merge has completed.
func getParentBlockHashPreMerge(ctx context.Context, vs *Server, st state.BeaconState, slot primitives.Slot) ([]byte, error) {
	t, err := slots.ToTime(st.GenesisTime(), slot)
	if err != nil {
		return nil, err
	}

	parentHash, hasTerminalBlock, err := vs.getTerminalBlockHashIfExists(ctx, uint64(t.Unix()))
	if err != nil {
		return nil, err
	}
	if !hasTerminalBlock {
		return nil, errNoTerminalBlockHash
	}
	return parentHash, nil
}

// activationEpochNotReached returns true if activation epoch has not been reach.
// Which satisfy the following conditions in spec:
//
//	  is_terminal_block_hash_set = TERMINAL_BLOCK_HASH != Hash32()
//	  is_activation_epoch_reached = get_current_epoch(state) >= TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH
//	  if is_terminal_block_hash_set and not is_activation_epoch_reached:
//		return True
func activationEpochNotReached(slot primitives.Slot) bool {
	terminalBlockHashSet := bytesutil.ToBytes32(params.BeaconConfig().TerminalBlockHash.Bytes()) != [32]byte{}
	if terminalBlockHashSet {
		return params.BeaconConfig().TerminalBlockHashActivationEpoch > slots.ToEpoch(slot)
	}
	return false
}

// calculateValidBlockHash calculates a valid block hash for empty execution payload
func calculateValidBlockHash(parentHash []byte, feeRecipient []byte, stateRoot []byte, receiptsRoot []byte,
	logsBloom []byte, prevRandao []byte, blockNumber uint64, gasLimit uint64, gasUsed uint64,
	timestamp uint64, extraData []byte, baseFeePerGas *big.Int, transactions [][]byte, withdrawals interface{}) []byte {

	// Create an Ethereum block header structure for hash calculation
	header := &types.Header{
		ParentHash:  common.BytesToHash(parentHash),
		UncleHash:   types.EmptyUncleHash, // Always empty for PoS
		Coinbase:    common.BytesToAddress(feeRecipient),
		Root:        common.BytesToHash(stateRoot),
		TxHash:      types.DeriveSha(types.Transactions{}, new(types.TrieStackHasher)), // Empty transactions
		ReceiptHash: common.BytesToHash(receiptsRoot),
		Bloom:       types.BytesToBloom(logsBloom),
		Difficulty:  big.NewInt(0), // Always 0 for PoS
		Number:      new(big.Int).SetUint64(blockNumber),
		GasLimit:    gasLimit,
		GasUsed:     gasUsed,
		Time:        timestamp,
		Extra:       extraData,
		MixDigest:   common.BytesToHash(prevRandao),
		Nonce:       types.BlockNonce{}, // Always zero for PoS
		BaseFee:     baseFeePerGas,
	}

	blockHash := header.Hash()
	return blockHash.Bytes()
}

func emptyPayload() *enginev1.ExecutionPayload {
	// Use current time for timestamp
	timestamp := uint64(time.Now().Unix())

	// Basic empty payload fields
	parentHash := make([]byte, fieldparams.RootLength)
	feeRecipient := params.BeaconConfig().DefaultFeeRecipient.Bytes()
	stateRoot := types.EmptyRootHash.Bytes()
	receiptsRoot := types.EmptyRootHash.Bytes()
	logsBloom := make([]byte, fieldparams.LogsBloomLength)
	prevRandao := make([]byte, fieldparams.RootLength)
	extraData := []byte("single-validator-recovery")
	baseFeePerGas := make([]byte, fieldparams.RootLength) // Will be converted to big.Int
	transactions := make([][]byte, 0)

	// Calculate valid block hash
	baseFee := new(big.Int).SetBytes(baseFeePerGas)
	if baseFee.Cmp(big.NewInt(0)) == 0 {
		baseFee = big.NewInt(1000000000) // 1 gwei default
	}

	blockHash := calculateValidBlockHash(parentHash, feeRecipient, stateRoot, receiptsRoot,
		logsBloom, prevRandao, 1, 30000000, 0, timestamp, extraData, baseFee, transactions, nil)

	return &enginev1.ExecutionPayload{
		ParentHash:    parentHash,
		FeeRecipient:  feeRecipient,
		StateRoot:     stateRoot,
		ReceiptsRoot:  receiptsRoot,
		LogsBloom:     logsBloom,
		PrevRandao:    prevRandao,
		BlockNumber:   1,
		GasLimit:      30000000,
		GasUsed:       0,
		Timestamp:     timestamp,
		ExtraData:     extraData,
		BaseFeePerGas: baseFee.Bytes(),
		BlockHash:     blockHash,
		Transactions:  transactions,
	}
}

func emptyPayloadCapella() *enginev1.ExecutionPayloadCapella {
	// Use current time for timestamp
	timestamp := uint64(time.Now().Unix())

	// Basic empty payload fields
	parentHash := make([]byte, fieldparams.RootLength)
	feeRecipient := params.BeaconConfig().DefaultFeeRecipient.Bytes()
	stateRoot := types.EmptyRootHash.Bytes()
	receiptsRoot := types.EmptyRootHash.Bytes()
	logsBloom := make([]byte, fieldparams.LogsBloomLength)
	prevRandao := make([]byte, fieldparams.RootLength)
	extraData := []byte("single-validator-recovery")
	transactions := make([][]byte, 0)
	withdrawals := make([]*enginev1.Withdrawal, 0)

	// Calculate valid block hash
	baseFee := big.NewInt(1000000000) // 1 gwei default
	blockHash := calculateValidBlockHash(parentHash, feeRecipient, stateRoot, receiptsRoot,
		logsBloom, prevRandao, 1, 30000000, 0, timestamp, extraData, baseFee, transactions, withdrawals)

	return &enginev1.ExecutionPayloadCapella{
		ParentHash:    parentHash,
		FeeRecipient:  feeRecipient,
		StateRoot:     stateRoot,
		ReceiptsRoot:  receiptsRoot,
		LogsBloom:     logsBloom,
		PrevRandao:    prevRandao,
		BlockNumber:   1,
		GasLimit:      30000000,
		GasUsed:       0,
		Timestamp:     timestamp,
		ExtraData:     extraData,
		BaseFeePerGas: baseFee.Bytes(),
		BlockHash:     blockHash,
		Transactions:  transactions,
		Withdrawals:   withdrawals,
	}
}

func emptyPayloadDeneb() *enginev1.ExecutionPayloadDeneb {
	// Use current time for timestamp
	timestamp := uint64(time.Now().Unix())

	// Basic empty payload fields
	parentHash := make([]byte, fieldparams.RootLength)
	feeRecipient := params.BeaconConfig().DefaultFeeRecipient.Bytes()
	stateRoot := types.EmptyRootHash.Bytes()
	receiptsRoot := types.EmptyRootHash.Bytes()
	logsBloom := make([]byte, fieldparams.LogsBloomLength)
	prevRandao := make([]byte, fieldparams.RootLength)
	extraData := []byte("single-validator-recovery")
	transactions := make([][]byte, 0)
	withdrawals := make([]*enginev1.Withdrawal, 0)

	// Calculate valid block hash
	baseFee := big.NewInt(1000000000) // 1 gwei default
	blockHash := calculateValidBlockHash(parentHash, feeRecipient, stateRoot, receiptsRoot,
		logsBloom, prevRandao, 1, 30000000, 0, timestamp, extraData, baseFee, transactions, withdrawals)

	return &enginev1.ExecutionPayloadDeneb{
		ParentHash:    parentHash,
		FeeRecipient:  feeRecipient,
		StateRoot:     stateRoot,
		ReceiptsRoot:  receiptsRoot,
		LogsBloom:     logsBloom,
		PrevRandao:    prevRandao,
		BlockNumber:   1,
		GasLimit:      30000000,
		GasUsed:       0,
		Timestamp:     timestamp,
		ExtraData:     extraData,
		BaseFeePerGas: baseFee.Bytes(),
		BlockHash:     blockHash,
		Transactions:  transactions,
		Withdrawals:   withdrawals,
	}
}

// emptyPayloadWithContext creates an empty payload using beacon state context
func emptyPayloadWithContext(head state.BeaconState, slot primitives.Slot) *enginev1.ExecutionPayload {
	// Get execution payload header from beacon state for context
	executionPayloadHeader, err := head.LatestExecutionPayloadHeader()
	if err != nil {
		logrus.WithError(err).Warn("Could not get execution payload header, using basic empty payload")
		return emptyPayload()
	}

	// Use current time for timestamp but ensure it's after parent
	timestamp := uint64(time.Now().Unix())
	parentTimestamp := executionPayloadHeader.Timestamp()
	if timestamp <= parentTimestamp {
		timestamp = parentTimestamp + 12 // Add 12 seconds (typical block time)
	}

	// Get proper values from execution payload header
	parentHash := executionPayloadHeader.BlockHash()
	blockNumber := executionPayloadHeader.BlockNumber() + 1
	baseFeePerGas := executionPayloadHeader.BaseFeePerGas()

	// Basic empty payload fields with proper context
	feeRecipient := params.BeaconConfig().DefaultFeeRecipient.Bytes()
	stateRoot := types.EmptyRootHash.Bytes()
	receiptsRoot := types.EmptyRootHash.Bytes()
	logsBloom := make([]byte, fieldparams.LogsBloomLength)
	prevRandao := make([]byte, fieldparams.RootLength)
	extraData := []byte("single-validator-recovery")
	transactions := make([][]byte, 0)

	// Calculate valid block hash with proper context
	baseFee := new(big.Int).SetBytes(bytesutil.ReverseByteOrder(baseFeePerGas))
	if baseFee.Cmp(big.NewInt(0)) == 0 {
		baseFee = big.NewInt(1000000000) // 1 gwei fallback
	}

	blockHash := calculateValidBlockHash(parentHash, feeRecipient, stateRoot, receiptsRoot,
		logsBloom, prevRandao, blockNumber, 30000000, 0, timestamp, extraData, baseFee, transactions, nil)

	return &enginev1.ExecutionPayload{
		ParentHash:    parentHash,
		FeeRecipient:  feeRecipient,
		StateRoot:     stateRoot,
		ReceiptsRoot:  receiptsRoot,
		LogsBloom:     logsBloom,
		PrevRandao:    prevRandao,
		BlockNumber:   blockNumber,
		GasLimit:      30000000,
		GasUsed:       0,
		Timestamp:     timestamp,
		ExtraData:     extraData,
		BaseFeePerGas: baseFeePerGas,
		BlockHash:     blockHash,
		Transactions:  transactions,
	}
}

// emptyPayloadCapellaWithContext creates an empty Capella payload using beacon state context
func emptyPayloadCapellaWithContext(head state.BeaconState, slot primitives.Slot) *enginev1.ExecutionPayloadCapella {
	// Get execution payload header from beacon state for context
	executionPayloadHeader, err := head.LatestExecutionPayloadHeader()
	if err != nil {
		logrus.WithError(err).Warn("Could not get execution payload header, using basic empty payload")
		return emptyPayloadCapella()
	}

	// Use current time for timestamp but ensure it's after parent
	timestamp := uint64(time.Now().Unix())
	parentTimestamp := executionPayloadHeader.Timestamp()
	if timestamp <= parentTimestamp {
		timestamp = parentTimestamp + 12 // Add 12 seconds
	}

	// Get proper values from execution payload header
	parentHash := executionPayloadHeader.BlockHash()
	blockNumber := executionPayloadHeader.BlockNumber() + 1
	baseFeePerGas := executionPayloadHeader.BaseFeePerGas()

	// Basic empty payload fields with proper context
	feeRecipient := params.BeaconConfig().DefaultFeeRecipient.Bytes()
	stateRoot := types.EmptyRootHash.Bytes()
	receiptsRoot := types.EmptyRootHash.Bytes()
	logsBloom := make([]byte, fieldparams.LogsBloomLength)
	prevRandao := make([]byte, fieldparams.RootLength)
	extraData := []byte("single-validator-recovery")
	transactions := make([][]byte, 0)
	withdrawals := make([]*enginev1.Withdrawal, 0)

	// Calculate valid block hash with proper context
	baseFee := new(big.Int).SetBytes(bytesutil.ReverseByteOrder(baseFeePerGas))
	if baseFee.Cmp(big.NewInt(0)) == 0 {
		baseFee = big.NewInt(1000000000)
	}

	blockHash := calculateValidBlockHash(parentHash, feeRecipient, stateRoot, receiptsRoot,
		logsBloom, prevRandao, blockNumber, 30000000, 0, timestamp, extraData, baseFee, transactions, withdrawals)

	return &enginev1.ExecutionPayloadCapella{
		ParentHash:    parentHash,
		FeeRecipient:  feeRecipient,
		StateRoot:     stateRoot,
		ReceiptsRoot:  receiptsRoot,
		LogsBloom:     logsBloom,
		PrevRandao:    prevRandao,
		BlockNumber:   blockNumber,
		GasLimit:      30000000,
		GasUsed:       0,
		Timestamp:     timestamp,
		ExtraData:     extraData,
		BaseFeePerGas: baseFeePerGas,
		BlockHash:     blockHash,
		Transactions:  transactions,
		Withdrawals:   withdrawals,
	}
}

// emptyPayloadDenebWithContext creates an empty Deneb payload using beacon state context
func emptyPayloadDenebWithContext(head state.BeaconState, slot primitives.Slot) *enginev1.ExecutionPayloadDeneb {
	// Get execution payload header from beacon state for context
	executionPayloadHeader, err := head.LatestExecutionPayloadHeader()
	if err != nil {
		logrus.WithError(err).Warn("Could not get execution payload header, using basic empty payload")
		return emptyPayloadDeneb()
	}

	// Use current time for timestamp but ensure it's after parent
	timestamp := uint64(time.Now().Unix())
	parentTimestamp := executionPayloadHeader.Timestamp()
	if timestamp <= parentTimestamp {
		timestamp = parentTimestamp + 12 // Add 12 seconds
	}

	// Get proper values from execution payload header
	parentHash := executionPayloadHeader.BlockHash()
	blockNumber := executionPayloadHeader.BlockNumber() + 1
	baseFeePerGas := executionPayloadHeader.BaseFeePerGas()

	// Basic empty payload fields with proper context
	feeRecipient := params.BeaconConfig().DefaultFeeRecipient.Bytes()
	stateRoot := types.EmptyRootHash.Bytes()
	receiptsRoot := types.EmptyRootHash.Bytes()
	logsBloom := make([]byte, fieldparams.LogsBloomLength)
	prevRandao := make([]byte, fieldparams.RootLength)
	extraData := []byte("single-validator-recovery")
	transactions := make([][]byte, 0)
	withdrawals := make([]*enginev1.Withdrawal, 0)

	// Calculate valid block hash with proper context
	baseFee := new(big.Int).SetBytes(bytesutil.ReverseByteOrder(baseFeePerGas))
	if baseFee.Cmp(big.NewInt(0)) == 0 {
		baseFee = big.NewInt(1000000000)
	}

	blockHash := calculateValidBlockHash(parentHash, feeRecipient, stateRoot, receiptsRoot,
		logsBloom, prevRandao, blockNumber, 30000000, 0, timestamp, extraData, baseFee, transactions, withdrawals)

	return &enginev1.ExecutionPayloadDeneb{
		ParentHash:    parentHash,
		FeeRecipient:  feeRecipient,
		StateRoot:     stateRoot,
		ReceiptsRoot:  receiptsRoot,
		LogsBloom:     logsBloom,
		PrevRandao:    prevRandao,
		BlockNumber:   blockNumber,
		GasLimit:      30000000,
		GasUsed:       0,
		Timestamp:     timestamp,
		ExtraData:     extraData,
		BaseFeePerGas: baseFeePerGas,
		BlockHash:     blockHash,
		Transactions:  transactions,
		Withdrawals:   withdrawals,
	}
}
