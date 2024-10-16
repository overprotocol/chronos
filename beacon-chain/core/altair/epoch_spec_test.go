package altair_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestProcessSyncCommitteeUpdates_CanRotate(t *testing.T) {
	s, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	h := &ethpb.BeaconBlockHeader{
		StateRoot:  bytesutil.PadTo([]byte{'a'}, 32),
		ParentRoot: bytesutil.PadTo([]byte{'b'}, 32),
		BodyRoot:   bytesutil.PadTo([]byte{'c'}, 32),
	}
	require.NoError(t, s.SetLatestBlockHeader(h))
	postState, err := altair.ProcessSyncCommitteeUpdates(context.Background(), s)
	require.NoError(t, err)
	current, err := postState.CurrentSyncCommittee()
	require.NoError(t, err)
	next, err := postState.NextSyncCommittee()
	require.NoError(t, err)
	require.DeepEqual(t, current, next)

	require.NoError(t, s.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	postState, err = altair.ProcessSyncCommitteeUpdates(context.Background(), s)
	require.NoError(t, err)
	c, err := postState.CurrentSyncCommittee()
	require.NoError(t, err)
	n, err := postState.NextSyncCommittee()
	require.NoError(t, err)
	require.DeepEqual(t, current, c)
	require.DeepEqual(t, next, n)

	require.NoError(t, s.SetSlot(primitives.Slot(params.BeaconConfig().EpochsPerSyncCommitteePeriod)*params.BeaconConfig().SlotsPerEpoch-1))
	postState, err = altair.ProcessSyncCommitteeUpdates(context.Background(), s)
	require.NoError(t, err)
	c, err = postState.CurrentSyncCommittee()
	require.NoError(t, err)
	n, err = postState.NextSyncCommittee()
	require.NoError(t, err)
	require.NotEqual(t, current, c)
	require.NotEqual(t, next, n)
	require.DeepEqual(t, next, c)

	// Test boundary condition.
	slot := params.BeaconConfig().SlotsPerEpoch * primitives.Slot(time.CurrentEpoch(s)+params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	require.NoError(t, s.SetSlot(slot))
	boundaryCommittee, err := altair.NextSyncCommittee(context.Background(), s)
	require.NoError(t, err)
	require.DeepNotEqual(t, boundaryCommittee, n)
}

func TestProcessParticipationFlagUpdates_CanRotate(t *testing.T) {
	s, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	c, err := s.CurrentEpochParticipation()
	require.NoError(t, err)
	require.DeepEqual(t, make([]byte, params.BeaconConfig().MaxValidatorsPerCommittee), c)
	p, err := s.PreviousEpochParticipation()
	require.NoError(t, err)
	require.DeepEqual(t, make([]byte, params.BeaconConfig().MaxValidatorsPerCommittee), p)

	newC := []byte{'a'}
	newP := []byte{'b'}
	require.NoError(t, s.SetCurrentParticipationBits(newC))
	require.NoError(t, s.SetPreviousParticipationBits(newP))
	c, err = s.CurrentEpochParticipation()
	require.NoError(t, err)
	require.DeepEqual(t, newC, c)
	p, err = s.PreviousEpochParticipation()
	require.NoError(t, err)
	require.DeepEqual(t, newP, p)

	s, err = altair.ProcessParticipationFlagUpdates(s)
	require.NoError(t, err)
	c, err = s.CurrentEpochParticipation()
	require.NoError(t, err)
	require.DeepEqual(t, make([]byte, params.BeaconConfig().MaxValidatorsPerCommittee), c)
	p, err = s.PreviousEpochParticipation()
	require.NoError(t, err)
	require.DeepEqual(t, newC, p)
}
