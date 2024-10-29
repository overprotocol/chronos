package altair

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
)

// ProcessParticipationFlagUpdates processes participation flag updates by rotating current to previous.
//
// nolint:dupword
// Spec code:
// def process_participation_flag_updates(state: BeaconState) -> None:
//
//	state.previous_epoch_participation = state.current_epoch_participation
//	state.current_epoch_participation = [ParticipationFlags(0b0000_0000) for _ in range(len(state.validators))]
func ProcessParticipationFlagUpdates(beaconState state.BeaconState) (state.BeaconState, error) {
	c, err := beaconState.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	if err := beaconState.SetPreviousParticipationBits(c); err != nil {
		return nil, err
	}
	if err := beaconState.SetCurrentParticipationBits(make([]byte, beaconState.NumValidators())); err != nil {
		return nil, err
	}
	return beaconState, nil
}
