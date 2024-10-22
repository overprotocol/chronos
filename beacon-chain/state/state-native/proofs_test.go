package state_native_test

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	statenative "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/container/trie"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestBeaconStateMerkleProofs_phase0_notsupported(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisState(t, 256)
	t.Run("current sync committee", func(t *testing.T) {
		_, err := st.CurrentSyncCommitteeProof(ctx)
		require.ErrorContains(t, "not supported", err)
	})
	t.Run("next sync committee", func(t *testing.T) {
		_, err := st.NextSyncCommitteeProof(ctx)
		require.ErrorContains(t, "not supported", err)
	})
	t.Run("finalized root", func(t *testing.T) {
		_, err := st.FinalizedRootProof(ctx)
		require.ErrorContains(t, "not supported", err)
	})
}
func TestBeaconStateMerkleProofs_altair(t *testing.T) {
	ctx := context.Background()
	altair, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	htr, err := altair.HashTreeRoot(ctx)
	require.NoError(t, err)
	t.Run("current sync committee", func(t *testing.T) {
		results := []string{
			"0xacff3e632bf8ff27b783ac48086a544d1e920512add91817790d355e09846cd0",
			"0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71",
			"0x260573b8c3368abde2e68d9282e184ff623c2067fdd2f6dd0b17ad6f6dac06ea",
			"0x43058e6846e144df0ba227c7a58e7d4c61c0882818c017209dbefbbc097a3821",
			"0x77e194167398c2332f6acc3b6b0311ac9a01d17ac1a19120c1a206a6e40f435f",
		}
		cscp, err := altair.CurrentSyncCommitteeProof(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, len(cscp))
		for i, bytes := range cscp {
			require.Equal(t, results[i], hexutil.Encode(bytes))
		}
	})
	t.Run("next sync committee", func(t *testing.T) {
		n_results := []string{
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0xf5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b",
			"0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71",
			"0x2bd2408b7bd7bc2d3dca7f5feb032195af06ceab3848d3b4a88bf9fadd85112e",
			"0x77e194167398c2332f6acc3b6b0311ac9a01d17ac1a19120c1a206a6e40f435f",
		}
		nscp, err := altair.NextSyncCommitteeProof(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, len(nscp))
		for i, bytes := range nscp {
			require.Equal(t, n_results[i], hexutil.Encode(bytes))
		}
	})
	t.Run("finalized root", func(t *testing.T) {
		finalizedRoot := altair.FinalizedCheckpoint().Root
		proof, err := altair.FinalizedRootProof(ctx)
		require.NoError(t, err)
		gIndex := statenative.FinalizedRootGeneralizedIndex()
		valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, gIndex, proof)
		require.Equal(t, true, valid)
	})
	t.Run("recomputes root on dirty fields", func(t *testing.T) {
		currentRoot, err := altair.HashTreeRoot(ctx)
		require.NoError(t, err)
		cpt := altair.FinalizedCheckpoint()
		require.NoError(t, err)

		// Edit the checkpoint.
		cpt.Epoch = 100
		require.NoError(t, altair.SetFinalizedCheckpoint(cpt))

		// Produce a proof for the finalized root.
		proof, err := altair.FinalizedRootProof(ctx)
		require.NoError(t, err)

		// We expect the previous step to have triggered
		// a recomputation of dirty fields in the beacon state, resulting
		// in a new hash tree root as the finalized checkpoint had previously
		// changed and should have been marked as a dirty state field.
		// The proof validity should be false for the old root, but true for the new.
		finalizedRoot := altair.FinalizedCheckpoint().Root
		gIndex := statenative.FinalizedRootGeneralizedIndex()
		valid := trie.VerifyMerkleProof(currentRoot[:], finalizedRoot, gIndex, proof)
		require.Equal(t, false, valid)

		newRoot, err := altair.HashTreeRoot(ctx)
		require.NoError(t, err)

		valid = trie.VerifyMerkleProof(newRoot[:], finalizedRoot, gIndex, proof)
		require.Equal(t, true, valid)
	})
}

func TestBeaconStateMerkleProofs_bellatrix(t *testing.T) {
	ctx := context.Background()
	bellatrix, err := util.NewBeaconStateBellatrix()
	require.NoError(t, err)
	htr, err := bellatrix.HashTreeRoot(ctx)
	require.NoError(t, err)
	t.Run("current sync committee", func(t *testing.T) {
		results := []string{
			"0xacff3e632bf8ff27b783ac48086a544d1e920512add91817790d355e09846cd0",
			"0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71",
			"0x260573b8c3368abde2e68d9282e184ff623c2067fdd2f6dd0b17ad6f6dac06ea",
			"0x0493908ba862a93eb36ae54c462417a15457080ee7e9ca7dba38b299bca05e25",
			"0x77e194167398c2332f6acc3b6b0311ac9a01d17ac1a19120c1a206a6e40f435f",
		}
		cscp, err := bellatrix.CurrentSyncCommitteeProof(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, len(cscp))
		for i, bytes := range cscp {
			require.Equal(t, results[i], hexutil.Encode(bytes))
		}
	})
	t.Run("next sync committee", func(t *testing.T) {
		n_results := []string{
			"0x22216a4a17e55cc41ce454600e5deb8aad32f15580a938b1914f93a9652c0e2c",
			"0xf5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b",
			"0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71",
			"0x2bd2408b7bd7bc2d3dca7f5feb032195af06ceab3848d3b4a88bf9fadd85112e",
			"0x77e194167398c2332f6acc3b6b0311ac9a01d17ac1a19120c1a206a6e40f435f",
		}
		nscp, err := bellatrix.NextSyncCommitteeProof(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, len(nscp))
		for i, bytes := range nscp {
			require.Equal(t, n_results[i], hexutil.Encode(bytes))
		}
	})
	t.Run("finalized root", func(t *testing.T) {
		finalizedRoot := bellatrix.FinalizedCheckpoint().Root
		proof, err := bellatrix.FinalizedRootProof(ctx)
		require.NoError(t, err)
		gIndex := statenative.FinalizedRootGeneralizedIndex()
		valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, gIndex, proof)
		require.Equal(t, true, valid)
	})
	t.Run("recomputes root on dirty fields", func(t *testing.T) {
		currentRoot, err := bellatrix.HashTreeRoot(ctx)
		require.NoError(t, err)
		cpt := bellatrix.FinalizedCheckpoint()
		require.NoError(t, err)

		// Edit the checkpoint.
		cpt.Epoch = 100
		require.NoError(t, bellatrix.SetFinalizedCheckpoint(cpt))

		// Produce a proof for the finalized root.
		proof, err := bellatrix.FinalizedRootProof(ctx)
		require.NoError(t, err)

		// We expect the previous step to have triggered
		// a recomputation of dirty fields in the beacon state, resulting
		// in a new hash tree root as the finalized checkpoint had previously
		// changed and should have been marked as a dirty state field.
		// The proof validity should be false for the old root, but true for the new.
		finalizedRoot := bellatrix.FinalizedCheckpoint().Root
		gIndex := statenative.FinalizedRootGeneralizedIndex()
		valid := trie.VerifyMerkleProof(currentRoot[:], finalizedRoot, gIndex, proof)
		require.Equal(t, false, valid)

		newRoot, err := bellatrix.HashTreeRoot(ctx)
		require.NoError(t, err)

		valid = trie.VerifyMerkleProof(newRoot[:], finalizedRoot, gIndex, proof)
		require.Equal(t, true, valid)
	})
}
