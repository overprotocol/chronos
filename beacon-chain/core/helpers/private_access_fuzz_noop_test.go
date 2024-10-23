//go:build fuzz

package helpers

import "github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"

func ProposerIndicesCache() *cache.FakeProposerIndicesCache {
	return proposerIndicesCache
}
