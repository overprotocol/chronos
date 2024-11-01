package blocks

import (
	"bytes"

	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// AreEth1DataEqual checks equality between two eth1 data objects.
func AreEth1DataEqual(a, b *ethpb.Eth1Data) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.DepositCount == b.DepositCount &&
		bytes.Equal(a.BlockHash, b.BlockHash) &&
		bytes.Equal(a.DepositRoot, b.DepositRoot)
}
