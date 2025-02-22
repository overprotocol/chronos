package benchmark

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestPreGenFullBlock(t *testing.T) {
	_, err := PreGenFullBlock()
	require.NoError(t, err)
}

func TestPreGenState1Epoch(t *testing.T) {
	t.Skip("Skipping test: ssz file must be generated again")
	_, err := PreGenState1Epoch()
	require.NoError(t, err)
}

func TestPreGenstateFullEpochs(t *testing.T) {
	t.Skip("Skipping test: ssz file must be generated again")
	_, err := PreGenstateFullEpochs()
	require.NoError(t, err)
}
