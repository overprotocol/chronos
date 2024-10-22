package eth

import (
	"bytes"
	"math/bits"
)

func FloorLog2(x uint64) int {
	return bits.Len64(x - 1)
}

func isEmptyWithLength(bb [][]byte, length uint64) bool {
	if len(bb) == 0 {
		return true
	}
	l := FloorLog2(length)
	if len(bb) != l {
		return false
	}
	for _, b := range bb {
		if !bytes.Equal(b, []byte{}) {
			return false
		}
	}
	return true
}
