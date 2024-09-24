package checkpoint

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db"
)

// Initializer describes a type that is able to obtain the checkpoint sync data (BeaconState and SignedBeaconBlock)
// in some way and perform database setup to prepare the beacon node for syncing from the given checkpoint.
// See FileInitializer and APIInitializer.
type Initializer interface {
	Initialize(ctx context.Context, d db.Database) error
}

// isCheckpointStatePresent checks if the checkpoint and corresponding state exist in the database.
func isCheckpointStatePresent(ctx context.Context, d db.Database) (bool, error) {
	origin, err := d.OriginCheckpointBlockRoot(ctx)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return false, nil
		}
		return false, errors.Wrap(err, "error while checking database for origin root")
	}
	// Check corresponding state
	if d.HasState(ctx, origin) {
		return true, nil
	}
	return false, nil
}
