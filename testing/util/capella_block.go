package util

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

// GenerateFullBlockCapella generates a fully valid Capella block with the requested parameters.
// Use BlockGenConfig to declare the conditions you would like the block generated under.
// This function modifies the passed state as follows:
func GenerateFullBlockCapella(
	bState state.BeaconState,
	privs []bls.SecretKey,
	conf *BlockGenConfig,
	slot primitives.Slot,
) (*ethpb.SignedBeaconBlockCapella, error) {
	ctx := context.Background()
	currentSlot := bState.Slot()
	if currentSlot > slot {
		return nil, fmt.Errorf("current slot in state is larger than given slot. %d > %d", currentSlot, slot)
	}
	bState = bState.Copy()

	if conf == nil {
		conf = &BlockGenConfig{}
	}

	var err error
	var pSlashings []*ethpb.ProposerSlashing
	numToGen := conf.NumProposerSlashings
	if numToGen > 0 {
		pSlashings, err = generateProposerSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d proposer slashings:", numToGen)
		}
	}

	numToGen = conf.NumAttesterSlashings
	var aSlashings []*ethpb.AttesterSlashing
	if numToGen > 0 {
		generated, err := generateAttesterSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
		aSlashings = make([]*ethpb.AttesterSlashing, len(generated))
		var ok bool
		for i, s := range generated {
			aSlashings[i], ok = s.(*ethpb.AttesterSlashing)
			if !ok {
				return nil, fmt.Errorf("attester slashing has the wrong type (expected %T, got %T)", &ethpb.AttesterSlashing{}, s)
			}
		}
	}

	numToGen = conf.NumAttestations
	var atts []*ethpb.Attestation
	if numToGen > 0 {
		generatedAtts, err := GenerateAttestations(bState, privs, numToGen, slot, false)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attestations:", numToGen)
		}
		atts = make([]*ethpb.Attestation, len(generatedAtts))
		var ok bool
		for i, a := range generatedAtts {
			atts[i], ok = a.(*ethpb.Attestation)
			if !ok {
				return nil, fmt.Errorf("attestation has the wrong type (expected %T, got %T)", &ethpb.Attestation{}, a)
			}
		}
	}

	numToGen = conf.NumDeposits
	var newDeposits []*ethpb.Deposit
	eth1Data := bState.Eth1Data()
	if numToGen > 0 {
		newDeposits, eth1Data, err = generateDepositsAndEth1Data(bState, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d deposits:", numToGen)
		}
	}

	numToGen = conf.NumVoluntaryExits
	var exits []*ethpb.SignedVoluntaryExit
	if numToGen > 0 {
		exits, err = generateVoluntaryExits(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
	}

	numToGen = conf.NumTransactions
	newTransactions := make([][]byte, numToGen)
	for i := uint64(0); i < numToGen; i++ {
		newTransactions[i] = bytesutil.Uint64ToBytesLittleEndian(i)
	}
	newWithdrawals := make([]*v1.Withdrawal, 0)

	random, err := helpers.RandaoMix(bState, time.CurrentEpoch(bState))
	if err != nil {
		return nil, errors.Wrap(err, "could not process randao mix")
	}

	timestamp, err := slots.ToTime(bState.GenesisTime(), slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get current timestamp")
	}

	stCopy := bState.Copy()
	stCopy, err = transition.ProcessSlots(context.Background(), stCopy, slot)
	if err != nil {
		return nil, err
	}

	parentExecution, err := stCopy.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	blockHash := indexToHash(uint64(slot))
	newExecutionPayloadCapella := &v1.ExecutionPayloadCapella{
		ParentHash:    parentExecution.BlockHash(),
		FeeRecipient:  make([]byte, 20),
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		ReceiptsRoot:  params.BeaconConfig().ZeroHash[:],
		LogsBloom:     make([]byte, 256),
		PrevRandao:    random,
		BlockNumber:   uint64(slot),
		ExtraData:     params.BeaconConfig().ZeroHash[:],
		BaseFeePerGas: params.BeaconConfig().ZeroHash[:],
		BlockHash:     blockHash[:],
		Timestamp:     uint64(timestamp.Unix()),
		Transactions:  newTransactions,
		Withdrawals:   newWithdrawals,
	}

	newHeader := bState.LatestBlockHeader()
	prevStateRoot, err := bState.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not hash state")
	}
	newHeader.StateRoot = prevStateRoot[:]
	parentRoot, err := newHeader.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not hash the new header")
	}

	if slot == currentSlot {
		slot = currentSlot + 1
	}

	reveal, err := RandaoReveal(stCopy, time.CurrentEpoch(stCopy), privs)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao reveal")
	}

	idx, err := helpers.BeaconProposerIndex(ctx, stCopy)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute beacon proposer index")
	}

	block := &ethpb.BeaconBlockCapella{
		Slot:          slot,
		ParentRoot:    parentRoot[:],
		ProposerIndex: idx,
		Body: &ethpb.BeaconBlockBodyCapella{
			Eth1Data:          eth1Data,
			RandaoReveal:      reveal,
			ProposerSlashings: pSlashings,
			AttesterSlashings: aSlashings,
			Attestations:      atts,
			VoluntaryExits:    exits,
			Deposits:          newDeposits,
			Graffiti:          make([]byte, fieldparams.RootLength),
			ExecutionPayload:  newExecutionPayloadCapella,
		},
	}

	// The fork can change after processing the state
	signature, err := BlockSignature(bState, block, privs)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block signature")
	}

	return &ethpb.SignedBeaconBlockCapella{Block: block, Signature: signature.Marshal()}, nil
}
