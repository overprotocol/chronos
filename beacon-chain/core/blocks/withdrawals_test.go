package blocks_test

import (
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func TestProcessBlindWithdrawals(t *testing.T) {
	const (
		currentEpoch             = primitives.Epoch(10)
		epochInFuture            = primitives.Epoch(12)
		epochInPast              = primitives.Epoch(8)
		numValidators            = 128
		notWithdrawableIndex     = 127
		notPartiallyWithdrawable = 126
		maxSweep                 = uint64(80)
	)
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalance

	type args struct {
		Name                            string
		NextWithdrawalValidatorIndex    primitives.ValidatorIndex
		NextWithdrawalIndex             uint64
		FullWithdrawalIndices           []primitives.ValidatorIndex
		PendingPartialWithdrawalIndices []primitives.ValidatorIndex
		Withdrawals                     []*enginev1.Withdrawal
	}
	type control struct {
		NextWithdrawalValidatorIndex primitives.ValidatorIndex
		NextWithdrawalIndex          uint64
		ExpectedError                bool
		Balances                     map[uint64]uint64
	}
	type Test struct {
		Args    args
		Control control
	}
	executionAddress := func(i primitives.ValidatorIndex) []byte {
		wc := make([]byte, 20)
		wc[19] = byte(i)
		return wc
	}
	withdrawalAmount := func(i primitives.ValidatorIndex) uint64 {
		return maxEffectiveBalance + uint64(i)*100000
	}
	fullWithdrawal := func(i primitives.ValidatorIndex, idx uint64) *enginev1.Withdrawal {
		return &enginev1.Withdrawal{
			Index:          idx,
			ValidatorIndex: i,
			Address:        executionAddress(i),
			Amount:         withdrawalAmount(i),
		}
	}
	PendingPartialWithdrawal := func(i primitives.ValidatorIndex, idx uint64) *enginev1.Withdrawal {
		return &enginev1.Withdrawal{
			Index:          idx,
			ValidatorIndex: i,
			Address:        executionAddress(i),
			Amount:         withdrawalAmount(i) - maxEffectiveBalance,
		}
	}
	tests := []Test{
		{
			Args: args{
				Name:                         "success no withdrawals",
				NextWithdrawalValidatorIndex: 10,
				NextWithdrawalIndex:          3,
			},
			Control: control{
				NextWithdrawalValidatorIndex: 90,
				NextWithdrawalIndex:          3,
			},
		},
		{
			Args: args{
				Name:                         "success one full withdrawal",
				NextWithdrawalIndex:          3,
				NextWithdrawalValidatorIndex: 5,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{70},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(70, 3),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 85,
				NextWithdrawalIndex:          4,
				Balances:                     map[uint64]uint64{70: 0},
			},
		},
		{
			Args: args{
				Name:                            "success one partial withdrawal",
				NextWithdrawalIndex:             21,
				NextWithdrawalValidatorIndex:    120,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{7},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(7, 21),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 72,
				NextWithdrawalIndex:          22,
				Balances:                     map[uint64]uint64{7: maxEffectiveBalance},
			},
		},
		{
			Args: args{
				Name:                         "success many full withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(28, 24),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 84,
				NextWithdrawalIndex:          25,
				Balances:                     map[uint64]uint64{7: 0, 19: 0, 28: 0},
			},
		},
		{
			Args: args{
				Name:                         "Less than max sweep at end",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{80, 81, 82, 83},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(80, 22), fullWithdrawal(81, 23), fullWithdrawal(82, 24),
					fullWithdrawal(83, 25),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 84,
				NextWithdrawalIndex:          26,
				Balances:                     map[uint64]uint64{80: 0, 81: 0, 82: 0, 83: 0},
			},
		},
		{
			Args: args{
				Name:                         "Less than max sweep and beginning",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{4, 5, 6},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(4, 22), fullWithdrawal(5, 23), fullWithdrawal(6, 24),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 84,
				NextWithdrawalIndex:          25,
				Balances:                     map[uint64]uint64{4: 0, 5: 0, 6: 0},
			},
		},
		{
			Args: args{
				Name:                            "success many partial withdrawals",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    4,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{7, 19, 28},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(7, 22), PendingPartialWithdrawal(19, 23), PendingPartialWithdrawal(28, 24),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 84,
				NextWithdrawalIndex:          25,
				Balances: map[uint64]uint64{
					7:  maxEffectiveBalance,
					19: maxEffectiveBalance,
					28: maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                            "success many withdrawals",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    88,
				FullWithdrawalIndices:           []primitives.ValidatorIndex{7, 19, 28},
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{2, 1, 89, 15},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(89, 22), PendingPartialWithdrawal(1, 23), PendingPartialWithdrawal(2, 24),
					fullWithdrawal(7, 25), PendingPartialWithdrawal(15, 26), fullWithdrawal(19, 27),
					fullWithdrawal(28, 28),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 40,
				NextWithdrawalIndex:          29,
				Balances: map[uint64]uint64{
					7: 0, 19: 0, 28: 0,
					2: maxEffectiveBalance, 1: maxEffectiveBalance, 89: maxEffectiveBalance,
					15: maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                         "success more than max fully withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 0,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 29, 35, 89},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(1, 22), fullWithdrawal(2, 23), fullWithdrawal(3, 24),
					fullWithdrawal(4, 25), fullWithdrawal(5, 26), fullWithdrawal(6, 27),
					fullWithdrawal(7, 28), fullWithdrawal(8, 29), fullWithdrawal(9, 30),
					fullWithdrawal(21, 31), fullWithdrawal(22, 32), fullWithdrawal(23, 33),
					fullWithdrawal(24, 34), fullWithdrawal(25, 35), fullWithdrawal(26, 36),
					fullWithdrawal(27, 37),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 28,
				NextWithdrawalIndex:          38,
				Balances: map[uint64]uint64{
					1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0, 7: 0, 8: 0, 9: 0,
					21: 0, 22: 0, 23: 0, 24: 0, 25: 0, 26: 0, 27: 0,
				},
			},
		},
		{
			Args: args{
				Name:                            "success more than max partially withdrawals",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    0,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 29, 35, 89},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(1, 22), PendingPartialWithdrawal(2, 23), PendingPartialWithdrawal(3, 24),
					PendingPartialWithdrawal(4, 25), PendingPartialWithdrawal(5, 26), PendingPartialWithdrawal(6, 27),
					PendingPartialWithdrawal(7, 28), PendingPartialWithdrawal(8, 29), PendingPartialWithdrawal(9, 30),
					PendingPartialWithdrawal(21, 31), PendingPartialWithdrawal(22, 32), PendingPartialWithdrawal(23, 33),
					PendingPartialWithdrawal(24, 34), PendingPartialWithdrawal(25, 35), PendingPartialWithdrawal(26, 36),
					PendingPartialWithdrawal(27, 37),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 28,
				NextWithdrawalIndex:          38,
				Balances: map[uint64]uint64{
					1:  maxEffectiveBalance,
					2:  maxEffectiveBalance,
					3:  maxEffectiveBalance,
					4:  maxEffectiveBalance,
					5:  maxEffectiveBalance,
					6:  maxEffectiveBalance,
					7:  maxEffectiveBalance,
					8:  maxEffectiveBalance,
					9:  maxEffectiveBalance,
					21: maxEffectiveBalance,
					22: maxEffectiveBalance,
					23: maxEffectiveBalance,
					24: maxEffectiveBalance,
					25: maxEffectiveBalance,
					26: maxEffectiveBalance,
					27: maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                            "failure wrong number of partial withdrawal",
				NextWithdrawalIndex:             21,
				NextWithdrawalValidatorIndex:    37,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{7},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(7, 21), PendingPartialWithdrawal(9, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid withdrawal index",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(28, 25),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid validator index",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(27, 24),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid withdrawal amount",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), PendingPartialWithdrawal(28, 24),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure validator not fully withdrawable",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{notWithdrawableIndex},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(notWithdrawableIndex, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                            "failure validator not partially withdrawable",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    4,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{notPartiallyWithdrawable},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(notPartiallyWithdrawable, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
	}

	checkPostState := func(t *testing.T, expected control, st state.BeaconState) {
		l, err := st.NextWithdrawalValidatorIndex()
		require.NoError(t, err)
		require.Equal(t, expected.NextWithdrawalValidatorIndex, l)

		n, err := st.NextWithdrawalIndex()
		require.NoError(t, err)
		require.Equal(t, expected.NextWithdrawalIndex, n)
		balances := st.Balances()
		for idx, bal := range expected.Balances {
			require.Equal(t, bal, balances[idx])
		}
	}

	prepareValidators := func(st *ethpb.BeaconStateCapella, arguments args) (state.BeaconState, error) {
		validators := make([]*ethpb.Validator, numValidators)
		st.Balances = make([]uint64, numValidators)
		for i := range validators {
			v := &ethpb.Validator{}
			v.EffectiveBalance = maxEffectiveBalance
			v.ExitEpoch = epochInFuture
			v.WithdrawalCredentials = make([]byte, 32)
			v.WithdrawalCredentials[31] = byte(i)
			v.PrincipalBalance = maxEffectiveBalance
			st.Balances[i] = v.EffectiveBalance - uint64(rand.Intn(1000))
			validators[i] = v
		}
		for _, idx := range arguments.FullWithdrawalIndices {
			if idx != notWithdrawableIndex {
				validators[idx].ExitEpoch = epochInPast
			}
			st.Balances[idx] = withdrawalAmount(idx)
			validators[idx].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		}
		for _, idx := range arguments.PendingPartialWithdrawalIndices {
			validators[idx].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			st.Balances[idx] = withdrawalAmount(idx)
		}
		st.Validators = validators
		return state_native.InitializeFromProtoCapella(st)
	}

	for _, test := range tests {
		t.Run(test.Args.Name, func(t *testing.T) {
			saved := params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep
			params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep = maxSweep
			params.BeaconConfig().MinValidatorWithdrawabilityDelay = 2
			if test.Args.Withdrawals == nil {
				test.Args.Withdrawals = make([]*enginev1.Withdrawal, 0)
			}
			if test.Args.FullWithdrawalIndices == nil {
				test.Args.FullWithdrawalIndices = make([]primitives.ValidatorIndex, 0)
			}
			if test.Args.PendingPartialWithdrawalIndices == nil {
				test.Args.PendingPartialWithdrawalIndices = make([]primitives.ValidatorIndex, 0)
			}
			slot, err := slots.EpochStart(currentEpoch)
			require.NoError(t, err)
			spb := &ethpb.BeaconStateCapella{
				Slot:                         slot,
				NextWithdrawalValidatorIndex: test.Args.NextWithdrawalValidatorIndex,
				NextWithdrawalIndex:          test.Args.NextWithdrawalIndex,
			}
			st, err := prepareValidators(spb, test.Args)
			require.NoError(t, err)
			wdRoot, err := ssz.WithdrawalSliceRoot(test.Args.Withdrawals, fieldparams.MaxWithdrawalsPerPayload)
			require.NoError(t, err)
			p, err := consensusblocks.WrappedExecutionPayloadHeaderCapella(&enginev1.ExecutionPayloadHeaderCapella{WithdrawalsRoot: wdRoot[:]})
			require.NoError(t, err)
			post, err := blocks.ProcessWithdrawals(st, p)
			if test.Control.ExpectedError {
				require.NotNil(t, err)
			} else {
				require.NoError(t, err)
				checkPostState(t, test.Control, post)
			}
			params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep = saved
		})
	}
}

func TestProcessWithdrawals(t *testing.T) {
	const (
		currentEpoch             = primitives.Epoch(10)
		epochInFuture            = primitives.Epoch(12)
		epochInPast              = primitives.Epoch(8)
		numValidators            = 128
		notWithdrawableIndex     = 127
		notPartiallyWithdrawable = 126
		maxSweep                 = uint64(80)
	)
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalance

	type args struct {
		Name                            string
		NextWithdrawalValidatorIndex    primitives.ValidatorIndex
		NextWithdrawalIndex             uint64
		FullWithdrawalIndices           []primitives.ValidatorIndex
		PendingPartialWithdrawalIndices []primitives.ValidatorIndex
		Withdrawals                     []*enginev1.Withdrawal
		PendingPartialWithdrawals       []*ethpb.PendingPartialWithdrawal // Electra
	}
	type control struct {
		NextWithdrawalValidatorIndex primitives.ValidatorIndex
		NextWithdrawalIndex          uint64
		ExpectedError                bool
		Balances                     map[uint64]uint64
	}
	type Test struct {
		Args    args
		Control control
	}
	executionAddress := func(i primitives.ValidatorIndex) []byte {
		wc := make([]byte, 20)
		wc[19] = byte(i)
		return wc
	}
	withdrawalAmount := func(i primitives.ValidatorIndex) uint64 {
		return maxEffectiveBalance + uint64(i)*100000
	}
	fullWithdrawal := func(i primitives.ValidatorIndex, idx uint64) *enginev1.Withdrawal {
		return &enginev1.Withdrawal{
			Index:          idx,
			ValidatorIndex: i,
			Address:        executionAddress(i),
			Amount:         withdrawalAmount(i),
		}
	}
	PendingPartialWithdrawal := func(i primitives.ValidatorIndex, idx uint64) *enginev1.Withdrawal {
		return &enginev1.Withdrawal{
			Index:          idx,
			ValidatorIndex: i,
			Address:        executionAddress(i),
			Amount:         withdrawalAmount(i) - maxEffectiveBalance,
		}
	}
	tests := []Test{
		{
			Args: args{
				Name:                         "success no withdrawals",
				NextWithdrawalValidatorIndex: 10,
				NextWithdrawalIndex:          3,
			},
			Control: control{
				NextWithdrawalValidatorIndex: 90,
				NextWithdrawalIndex:          3,
			},
		},
		{
			Args: args{
				Name:                         "success one full withdrawal",
				NextWithdrawalIndex:          3,
				NextWithdrawalValidatorIndex: 5,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{70},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(70, 3),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 85,
				NextWithdrawalIndex:          4,
				Balances:                     map[uint64]uint64{70: 0},
			},
		},
		{
			Args: args{
				Name:                            "success one partial withdrawal",
				NextWithdrawalIndex:             21,
				NextWithdrawalValidatorIndex:    120,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{7},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(7, 21),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 72,
				NextWithdrawalIndex:          22,
				Balances:                     map[uint64]uint64{7: maxEffectiveBalance},
			},
		},
		{
			Args: args{
				Name:                         "success many full withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(28, 24),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 84,
				NextWithdrawalIndex:          25,
				Balances:                     map[uint64]uint64{7: 0, 19: 0, 28: 0},
			},
		},
		{
			Args: args{
				Name:                         "less than max sweep at end",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{80, 81, 82, 83},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(80, 22), fullWithdrawal(81, 23), fullWithdrawal(82, 24),
					fullWithdrawal(83, 25),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 84,
				NextWithdrawalIndex:          26,
				Balances:                     map[uint64]uint64{80: 0, 81: 0, 82: 0, 83: 0},
			},
		},
		{
			Args: args{
				Name:                         "less than max sweep and beginning",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{4, 5, 6},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(4, 22), fullWithdrawal(5, 23), fullWithdrawal(6, 24),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 84,
				NextWithdrawalIndex:          25,
				Balances:                     map[uint64]uint64{4: 0, 5: 0, 6: 0},
			},
		},
		{
			Args: args{
				Name:                            "success many partial withdrawals",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    4,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{7, 19, 28},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(7, 22), PendingPartialWithdrawal(19, 23), PendingPartialWithdrawal(28, 24),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 84,
				NextWithdrawalIndex:          25,
				Balances: map[uint64]uint64{
					7:  maxEffectiveBalance,
					19: maxEffectiveBalance,
					28: maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                            "success many withdrawals",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    88,
				FullWithdrawalIndices:           []primitives.ValidatorIndex{7, 19, 28},
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{2, 1, 89, 15},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(89, 22), PendingPartialWithdrawal(1, 23), PendingPartialWithdrawal(2, 24),
					fullWithdrawal(7, 25), PendingPartialWithdrawal(15, 26), fullWithdrawal(19, 27),
					fullWithdrawal(28, 28),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 40,
				NextWithdrawalIndex:          29,
				Balances: map[uint64]uint64{
					7: 0, 19: 0, 28: 0,
					2: maxEffectiveBalance, 1: maxEffectiveBalance, 89: maxEffectiveBalance,
					15: maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                            "success many withdrawals with pending partial withdrawals in state",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    88,
				FullWithdrawalIndices:           []primitives.ValidatorIndex{7, 19, 28},
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{2, 1, 89, 15},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(89, 22), PendingPartialWithdrawal(1, 23), PendingPartialWithdrawal(2, 24),
					fullWithdrawal(7, 25), PendingPartialWithdrawal(15, 26), fullWithdrawal(19, 27),
					fullWithdrawal(28, 28),
				},
				PendingPartialWithdrawals: []*ethpb.PendingPartialWithdrawal{
					{
						Index:  11,
						Amount: withdrawalAmount(11) - maxEffectiveBalance,
					},
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 40,
				NextWithdrawalIndex:          29,
				Balances: map[uint64]uint64{
					7: 0, 19: 0, 28: 0,
					2: maxEffectiveBalance, 1: maxEffectiveBalance, 89: maxEffectiveBalance,
					15: maxEffectiveBalance,
				},
			},
		},

		{
			Args: args{
				Name:                         "success more than max fully withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 0,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 29, 35, 89},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(1, 22), fullWithdrawal(2, 23), fullWithdrawal(3, 24),
					fullWithdrawal(4, 25), fullWithdrawal(5, 26), fullWithdrawal(6, 27),
					fullWithdrawal(7, 28), fullWithdrawal(8, 29), fullWithdrawal(9, 30),
					fullWithdrawal(21, 31), fullWithdrawal(22, 32), fullWithdrawal(23, 33),
					fullWithdrawal(24, 34), fullWithdrawal(25, 35), fullWithdrawal(26, 36),
					fullWithdrawal(27, 37),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 28,
				NextWithdrawalIndex:          38,
				Balances: map[uint64]uint64{
					1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0, 7: 0, 8: 0, 9: 0,
					21: 0, 22: 0, 23: 0, 24: 0, 25: 0, 26: 0, 27: 0,
				},
			},
		},
		{
			Args: args{
				Name:                            "success more than max partially withdrawals",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    0,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 29, 35, 89},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(1, 22), PendingPartialWithdrawal(2, 23), PendingPartialWithdrawal(3, 24),
					PendingPartialWithdrawal(4, 25), PendingPartialWithdrawal(5, 26), PendingPartialWithdrawal(6, 27),
					PendingPartialWithdrawal(7, 28), PendingPartialWithdrawal(8, 29), PendingPartialWithdrawal(9, 30),
					PendingPartialWithdrawal(21, 31), PendingPartialWithdrawal(22, 32), PendingPartialWithdrawal(23, 33),
					PendingPartialWithdrawal(24, 34), PendingPartialWithdrawal(25, 35), PendingPartialWithdrawal(26, 36),
					PendingPartialWithdrawal(27, 37),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 28,
				NextWithdrawalIndex:          38,
				Balances: map[uint64]uint64{
					1:  maxEffectiveBalance,
					2:  maxEffectiveBalance,
					3:  maxEffectiveBalance,
					4:  maxEffectiveBalance,
					5:  maxEffectiveBalance,
					6:  maxEffectiveBalance,
					7:  maxEffectiveBalance,
					8:  maxEffectiveBalance,
					9:  maxEffectiveBalance,
					21: maxEffectiveBalance,
					22: maxEffectiveBalance,
					23: maxEffectiveBalance,
					24: maxEffectiveBalance,
					25: maxEffectiveBalance,
					26: maxEffectiveBalance,
					27: maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                            "failure wrong number of partial withdrawal",
				NextWithdrawalIndex:             21,
				NextWithdrawalValidatorIndex:    37,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{7},
				Withdrawals: []*enginev1.Withdrawal{
					PendingPartialWithdrawal(7, 21), PendingPartialWithdrawal(9, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid withdrawal index",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(28, 25),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid validator index",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(27, 24),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid withdrawal amount",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), PendingPartialWithdrawal(28, 24),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure validator not fully withdrawable",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []primitives.ValidatorIndex{notWithdrawableIndex},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(notWithdrawableIndex, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                            "failure validator not partially withdrawable",
				NextWithdrawalIndex:             22,
				NextWithdrawalValidatorIndex:    4,
				PendingPartialWithdrawalIndices: []primitives.ValidatorIndex{notPartiallyWithdrawable},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(notPartiallyWithdrawable, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
	}

	checkPostState := func(t *testing.T, expected control, st state.BeaconState) {
		l, err := st.NextWithdrawalValidatorIndex()
		require.NoError(t, err)
		require.Equal(t, expected.NextWithdrawalValidatorIndex, l)

		n, err := st.NextWithdrawalIndex()
		require.NoError(t, err)
		require.Equal(t, expected.NextWithdrawalIndex, n)
		balances := st.Balances()
		for idx, bal := range expected.Balances {
			require.Equal(t, bal, balances[idx])
		}
	}

	prepareValidators := func(st state.BeaconState, arguments args) error {
		validators := make([]*ethpb.Validator, numValidators)
		if err := st.SetBalances(make([]uint64, numValidators)); err != nil {
			return err
		}
		for i := range validators {
			v := &ethpb.Validator{}
			v.EffectiveBalance = maxEffectiveBalance
			v.ExitEpoch = epochInFuture
			v.WithdrawalCredentials = make([]byte, 32)
			v.WithdrawalCredentials[31] = byte(i)
			v.PrincipalBalance = v.EffectiveBalance
			if err := st.UpdateBalancesAtIndex(primitives.ValidatorIndex(i), v.EffectiveBalance-uint64(rand.Intn(1000))); err != nil {
				return err
			}
			validators[i] = v
		}
		for _, idx := range arguments.FullWithdrawalIndices {
			if idx != notWithdrawableIndex {
				validators[idx].ExitEpoch = epochInPast
			}
			if err := st.UpdateBalancesAtIndex(idx, withdrawalAmount(idx)); err != nil {
				return err
			}
			validators[idx].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		}
		for _, idx := range arguments.PendingPartialWithdrawalIndices {
			validators[idx].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			if err := st.UpdateBalancesAtIndex(idx, withdrawalAmount(idx)); err != nil {
				return err
			}
		}
		return st.SetValidators(validators)
	}

	for _, test := range tests {
		t.Run(test.Args.Name, func(t *testing.T) {
			for _, fork := range []int{version.Capella, version.Alpaca} {
				t.Run(version.String(fork), func(t *testing.T) {
					saved := params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep
					params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep = maxSweep
					params.BeaconConfig().MinValidatorWithdrawabilityDelay = 2
					if test.Args.Withdrawals == nil {
						test.Args.Withdrawals = make([]*enginev1.Withdrawal, 0)
					}
					if test.Args.FullWithdrawalIndices == nil {
						test.Args.FullWithdrawalIndices = make([]primitives.ValidatorIndex, 0)
					}
					if test.Args.PendingPartialWithdrawalIndices == nil {
						test.Args.PendingPartialWithdrawalIndices = make([]primitives.ValidatorIndex, 0)
					}
					slot, err := slots.EpochStart(currentEpoch)
					require.NoError(t, err)
					var st state.BeaconState
					var p interfaces.ExecutionData
					switch fork {
					case version.Capella:
						spb := &ethpb.BeaconStateCapella{
							Slot:                         slot,
							NextWithdrawalValidatorIndex: test.Args.NextWithdrawalValidatorIndex,
							NextWithdrawalIndex:          test.Args.NextWithdrawalIndex,
						}
						st, err = state_native.InitializeFromProtoUnsafeCapella(spb)
						require.NoError(t, err)
						p, err = consensusblocks.WrappedExecutionPayloadCapella(&enginev1.ExecutionPayloadCapella{Withdrawals: test.Args.Withdrawals})
						require.NoError(t, err)
					case version.Alpaca:
						spb := &ethpb.BeaconStateElectra{
							Slot:                         slot,
							NextWithdrawalValidatorIndex: test.Args.NextWithdrawalValidatorIndex,
							NextWithdrawalIndex:          test.Args.NextWithdrawalIndex,
							PendingPartialWithdrawals:    test.Args.PendingPartialWithdrawals,
						}
						st, err = state_native.InitializeFromProtoUnsafeElectra(spb)
						require.NoError(t, err)
						p, err = consensusblocks.WrappedExecutionPayloadDeneb(&enginev1.ExecutionPayloadDeneb{Withdrawals: test.Args.Withdrawals})
						require.NoError(t, err)
					default:
						t.Fatalf("Add a beacon state setup for version %s", version.String(fork))
					}
					err = prepareValidators(st, test.Args)
					require.NoError(t, err)
					post, err := blocks.ProcessWithdrawals(st, p)
					if test.Control.ExpectedError {
						require.NotNil(t, err)
					} else {
						require.NoError(t, err)
						checkPostState(t, test.Control, post)
					}
					params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep = saved

				})
			}
		})
	}
}
