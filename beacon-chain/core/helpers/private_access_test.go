//go:build !fuzz

package helpers

import "github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"

func CommitteeCache() *cache.CommitteeCache {
	return committeeCache
}

func ProposerIndicesCache() *cache.ProposerIndicesCache {
	return proposerIndicesCache
}
