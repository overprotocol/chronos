package altair_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

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
