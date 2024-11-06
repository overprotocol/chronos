package cache

// MerkleTree defines methods for constructing and manipulating a merkle tree.
type MerkleTree interface {
	HashTreeRoot() ([32]byte, error)
	NumOfItems() int
	Insert(item []byte, index int) error
	MerkleProof(index int) ([][]byte, error)
}
