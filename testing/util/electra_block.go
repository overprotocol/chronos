package util

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
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

// GenerateFullBlockElectra generates a fully valid Electra block with the requested parameters.
// Use BlockGenConfig to declare the conditions you would like the block generated under.
// This function modifies the passed state as follows:
func GenerateFullBlockElectra(
	bState state.BeaconState,
	privs []bls.SecretKey,
	conf *BlockGenConfig,
	slot primitives.Slot,
) (*ethpb.SignedBeaconBlockElectra, error) {
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
	var aSlashings []*ethpb.AttesterSlashingElectra
	if numToGen > 0 {
		generated, err := generateAttesterSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
		aSlashings = make([]*ethpb.AttesterSlashingElectra, len(generated))
		var ok bool
		for i, s := range generated {
			aSlashings[i], ok = s.(*ethpb.AttesterSlashingElectra)
			if !ok {
				return nil, fmt.Errorf("attester slashing has the wrong type (expected %T, got %T)", &ethpb.AttesterSlashingElectra{}, s)
			}
		}
	}

	numToGen = conf.NumAttestations
	var atts []*ethpb.AttestationElectra
	if numToGen > 0 {
		generatedAtts, err := GenerateAttestations(bState, privs, numToGen, slot, false)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attestations:", numToGen)
		}
		atts = make([]*ethpb.AttestationElectra, len(generatedAtts))
		var ok bool
		for i, a := range generatedAtts {
			atts[i], ok = a.(*ethpb.AttestationElectra)
			if !ok {
				return nil, fmt.Errorf("attestation has the wrong type (expected %T, got %T)", &ethpb.AttestationElectra{}, a)
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

	if conf.NumWithdrawals > 0 {
		stCopy, err = generatePendingPartialWithdrawals(stCopy, conf.NumWithdrawals)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d withdrawals:", conf.NumWithdrawals)
		}
	}

	expectedWithdrawal, err := generateExpectedWithdrawals(stCopy)
	if err != nil {
		return nil, errors.Wrap(err, "failed generating expected withdrawals")
	}

	depositRequests := make([]*v1.DepositRequest, 0)
	if conf.NumDepositRequests > 0 {
		depositRequests, err = generateDepositRequests(bState, privs, conf.NumDepositRequests)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d deposit requests:", conf.NumDepositRequests)
		}
	}

	withdrawalRequests := make([]*v1.WithdrawalRequest, 0)
	if conf.NumWithdrawalRequests > 0 {
		withdrawalRequests, err = generateWithdrawalRequests(bState, privs, conf.NumWithdrawalRequests)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d withdrawal requests:", conf.NumWithdrawalRequests)
		}
	}

	executionRequests := &v1.ExecutionRequests{
		Withdrawals: withdrawalRequests,
		Deposits:    depositRequests,
	}

	parentExecution, err := stCopy.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	blockHash := indexToHash(uint64(slot))
	newExecutionPayloadElectra := &v1.ExecutionPayloadDeneb{
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
		Withdrawals:   expectedWithdrawal,
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

	block := &ethpb.BeaconBlockElectra{
		Slot:          slot,
		ParentRoot:    parentRoot[:],
		ProposerIndex: idx,
		Body: &ethpb.BeaconBlockBodyElectra{
			Eth1Data:          eth1Data,
			RandaoReveal:      reveal,
			ProposerSlashings: pSlashings,
			AttesterSlashings: aSlashings,
			Attestations:      atts,
			VoluntaryExits:    exits,
			Deposits:          newDeposits,
			Graffiti:          make([]byte, fieldparams.RootLength),
			ExecutionPayload:  newExecutionPayloadElectra,
			ExecutionRequests: executionRequests,
		},
	}

	// The fork can change after processing the state
	signature, err := BlockSignature(bState, block, privs)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block signature")
	}

	return &ethpb.SignedBeaconBlockElectra{Block: block, Signature: signature.Marshal()}, nil
}

func generatePendingPartialWithdrawals(bState state.BeaconState, num uint64) (state.BeaconState, error) {
	for i := uint64(0); i < num; i++ {
		valIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		err = bState.AppendPendingPartialWithdrawal(&ethpb.PendingPartialWithdrawal{
			Index:             valIndex,
			Amount:            1,
			WithdrawableEpoch: slots.ToEpoch(bState.Slot()),
		})
		if err != nil {
			return nil, err
		}
	}
	return bState, nil
}

func generateExpectedWithdrawals(bState state.BeaconState) ([]*v1.Withdrawal, error) {
	expectedWithdrawals, _, _, err := bState.ExpectedWithdrawals()
	if err != nil {
		return nil, err
	}
	return expectedWithdrawals, nil
}

func generateWithdrawalRequests(
	bState state.BeaconState,
	privs []bls.SecretKey,
	numRequests uint64,
) ([]*v1.WithdrawalRequest, error) {
	withdrawalRequests := make([]*v1.WithdrawalRequest, numRequests)
	for i := uint64(0); i < numRequests; i++ {
		valIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		// Get a random index
		nBig, err := rand.Int(rand.Reader, big.NewInt(60000))
		if err != nil {
			return nil, err
		}
		amount := nBig.Uint64() // random amount created
		bal, err := bState.BalanceAtIndex(valIndex)
		if err != nil {
			return nil, err
		}
		amounts := []uint64{
			amount, // some smaller amount
			bal,    // the entire balance
		}
		// Get a random index
		nBig, err = rand.Int(rand.Reader, big.NewInt(int64(len(amounts))))
		if err != nil {
			return nil, err
		}
		randomIndex := nBig.Uint64()
		withdrawalRequests[i] = &v1.WithdrawalRequest{
			ValidatorPubkey: privs[valIndex].PublicKey().Marshal(),
			SourceAddress:   make([]byte, common.AddressLength),
			Amount:          amounts[randomIndex],
		}
	}
	return withdrawalRequests, nil
}

func generateDepositRequests(
	bState state.BeaconState,
	privs []bls.SecretKey,
	numRequests uint64,
) ([]*v1.DepositRequest, error) {
	depositRequests := make([]*v1.DepositRequest, numRequests)
	for i := uint64(0); i < numRequests; i++ {
		valIndex, err := randValIndex(bState)
		if err != nil {
			return nil, err
		}
		// Get a random index
		nBig, err := rand.Int(rand.Reader, big.NewInt(60000))
		if err != nil {
			return nil, err
		}
		amount := nBig.Uint64() // random amount created
		prefixes := []byte{params.BeaconConfig().CompoundingWithdrawalPrefixByte, 0, params.BeaconConfig().BLSWithdrawalPrefixByte}
		withdrawalCred := make([]byte, 32)
		// Get a random index
		nBig, err = rand.Int(rand.Reader, big.NewInt(int64(len(prefixes))))
		if err != nil {
			return nil, err
		}
		randPrefixIndex := nBig.Uint64()
		withdrawalCred[0] = prefixes[randPrefixIndex]

		depositMessage := &ethpb.DepositMessage{
			PublicKey:             privs[valIndex].PublicKey().Marshal(),
			Amount:                amount,
			WithdrawalCredentials: withdrawalCred,
		}
		domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
		if err != nil {
			return nil, err
		}
		sr, err := signing.ComputeSigningRoot(depositMessage, domain)
		if err != nil {
			return nil, err
		}
		sig := privs[i].Sign(sr[:])
		depositRequests[i] = &v1.DepositRequest{
			Pubkey:                depositMessage.PublicKey,
			Index:                 uint64(valIndex),
			WithdrawalCredentials: depositMessage.WithdrawalCredentials,
			Amount:                depositMessage.Amount,
			Signature:             sig.Marshal(),
		}
	}
	return depositRequests, nil
}
