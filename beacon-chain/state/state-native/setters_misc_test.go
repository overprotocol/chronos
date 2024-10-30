package state_native_test

import (
	"testing"

	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func BenchmarkAppendHistoricalSummaries(b *testing.B) {
	st, err := state_native.InitializeFromProtoCapella(&ethpb.BeaconStateCapella{})
	require.NoError(b, err)

	max := params.BeaconConfig().HistoricalRootsLimit
	if max < 2 {
		b.Fatalf("HistoricalRootsLimit is less than 2: %d", max)
	}

	for i := uint64(0); i < max-2; i++ {
		err := st.AppendHistoricalSummaries(&ethpb.HistoricalSummary{})
		require.NoError(b, err)
	}

	ref := st.Copy()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := ref.AppendHistoricalSummaries(&ethpb.HistoricalSummary{})
		require.NoError(b, err)
		ref = st.Copy()
	}
}
