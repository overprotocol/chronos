package electra_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestUpgradeToElectra(t *testing.T) {
	st, _ := util.DeterministicGenesisStateDeneb(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	vals := st.Validators()
	vals[0].ActivationEpoch = params.BeaconConfig().FarFutureEpoch
	require.NoError(t, st.SetValidators(vals))

	preForkState := st.Copy()
	mSt, err := electra.UpgradeToElectra(st)
	require.NoError(t, err)

	require.Equal(t, preForkState.GenesisTime(), mSt.GenesisTime())
	require.DeepSSZEqual(t, preForkState.GenesisValidatorsRoot(), mSt.GenesisValidatorsRoot())
	require.Equal(t, preForkState.Slot(), mSt.Slot())
	require.DeepSSZEqual(t, preForkState.LatestBlockHeader(), mSt.LatestBlockHeader())
	require.DeepSSZEqual(t, preForkState.BlockRoots(), mSt.BlockRoots())
	require.DeepSSZEqual(t, preForkState.StateRoots(), mSt.StateRoots())
	require.DeepSSZEqual(t, preForkState.Validators()[2:], mSt.Validators()[2:])
	require.DeepSSZEqual(t, preForkState.Balances()[2:], mSt.Balances()[2:])
	require.DeepSSZEqual(t, preForkState.Eth1Data(), mSt.Eth1Data())
	require.DeepSSZEqual(t, preForkState.Eth1DataVotes(), mSt.Eth1DataVotes())
	require.DeepSSZEqual(t, preForkState.Eth1DepositIndex(), mSt.Eth1DepositIndex())
	require.DeepSSZEqual(t, preForkState.RandaoMixes(), mSt.RandaoMixes())
	require.DeepSSZEqual(t, preForkState.JustificationBits(), mSt.JustificationBits())
	require.DeepSSZEqual(t, preForkState.PreviousJustifiedCheckpoint(), mSt.PreviousJustifiedCheckpoint())
	require.DeepSSZEqual(t, preForkState.CurrentJustifiedCheckpoint(), mSt.CurrentJustifiedCheckpoint())
	require.DeepSSZEqual(t, preForkState.FinalizedCheckpoint(), mSt.FinalizedCheckpoint())

	require.Equal(t, len(preForkState.Validators()), len(mSt.Validators()))

	preVal, err := preForkState.ValidatorAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance, preVal.EffectiveBalance)

	preVal2, err := preForkState.ValidatorAtIndex(1)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance, preVal2.EffectiveBalance)

	mVal, err := mSt.ValidatorAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), mVal.EffectiveBalance)

	mVal2, err := mSt.ValidatorAtIndex(1)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MinActivationBalance, mVal2.EffectiveBalance)

	numValidators := mSt.NumValidators()
	p, err := mSt.PreviousEpochParticipation()
	require.NoError(t, err)
	require.DeepSSZEqual(t, make([]byte, numValidators), p)
	p, err = mSt.CurrentEpochParticipation()
	require.NoError(t, err)
	require.DeepSSZEqual(t, make([]byte, numValidators), p)
	s, err := mSt.InactivityScores()
	require.NoError(t, err)
	require.DeepSSZEqual(t, make([]uint64, numValidators), s)

	f := mSt.Fork()
	require.DeepSSZEqual(t, &ethpb.Fork{
		PreviousVersion: st.Fork().CurrentVersion,
		CurrentVersion:  params.BeaconConfig().AlpacaForkVersion,
		Epoch:           time.CurrentEpoch(st),
	}, f)

	header, err := mSt.LatestExecutionPayloadHeader()
	require.NoError(t, err)
	protoHeader, ok := header.Proto().(*enginev1.ExecutionPayloadHeaderDeneb)
	require.Equal(t, true, ok)
	prevHeader, err := preForkState.LatestExecutionPayloadHeader()
	require.NoError(t, err)
	txRoot, err := prevHeader.TransactionsRoot()
	require.NoError(t, err)

	wdRoot, err := prevHeader.WithdrawalsRoot()
	require.NoError(t, err)
	wanted := &enginev1.ExecutionPayloadHeaderDeneb{
		ParentHash:       prevHeader.ParentHash(),
		FeeRecipient:     prevHeader.FeeRecipient(),
		StateRoot:        prevHeader.StateRoot(),
		ReceiptsRoot:     prevHeader.ReceiptsRoot(),
		LogsBloom:        prevHeader.LogsBloom(),
		PrevRandao:       prevHeader.PrevRandao(),
		BlockNumber:      prevHeader.BlockNumber(),
		GasLimit:         prevHeader.GasLimit(),
		GasUsed:          prevHeader.GasUsed(),
		Timestamp:        prevHeader.Timestamp(),
		ExtraData:        prevHeader.ExtraData(),
		BaseFeePerGas:    prevHeader.BaseFeePerGas(),
		BlockHash:        prevHeader.BlockHash(),
		TransactionsRoot: txRoot,
		WithdrawalsRoot:  wdRoot,
	}
	require.DeepEqual(t, wanted, protoHeader)

	nwi, err := mSt.NextWithdrawalIndex()
	require.NoError(t, err)
	require.Equal(t, uint64(0), nwi)

	lwvi, err := mSt.NextWithdrawalValidatorIndex()
	require.NoError(t, err)
	require.Equal(t, primitives.ValidatorIndex(0), lwvi)

	summaries, err := mSt.HistoricalSummaries()
	require.NoError(t, err)
	require.Equal(t, 0, len(summaries))

	startIndex, err := mSt.DepositRequestsStartIndex()
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().UnsetDepositRequestsStartIndex, startIndex)

	balance, err := mSt.DepositBalanceToConsume()
	require.NoError(t, err)
	require.Equal(t, primitives.Gwei(0), balance)

	tab, err := helpers.TotalActiveBalance(mSt)
	require.NoError(t, err)

	ebtc, err := mSt.ExitBalanceToConsume()
	require.NoError(t, err)
	require.Equal(t, helpers.ExitBalanceChurnLimit(primitives.Gwei(tab)), ebtc)

	eee, err := mSt.EarliestExitEpoch()
	require.NoError(t, err)
	require.Equal(t, primitives.Epoch(1), eee)

	pendingDeposits, err := mSt.PendingDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pendingDeposits))

	numPendingPartialWithdrawals, err := mSt.NumPendingPartialWithdrawals()
	require.NoError(t, err)
	require.Equal(t, uint64(0), numPendingPartialWithdrawals)
}
