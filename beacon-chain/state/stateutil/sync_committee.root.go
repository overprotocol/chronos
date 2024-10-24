package stateutil

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/crypto/hash/htr"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
)

func merkleizePubkey(pubkey []byte) ([32]byte, error) {
	if len(pubkey) == 0 {
		return [32]byte{}, errors.New("zero length pubkey provided")
	}
	chunks, err := ssz.PackByChunk([][]byte{pubkey})
	if err != nil {
		return [32]byte{}, err
	}
	outputChunk := htr.VectorizedSha256(chunks)

	return outputChunk[0], nil
}
