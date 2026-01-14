package crypto

import (
	"crypto/sha256"

	"github.com/yourusername/bt/pkg/types"
)

// HashBytes returns SHA-256 hash of the input data
func HashBytes(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// DoubleHashBytes returns double SHA-256 hash (Bitcoin-style)
func DoubleHashBytes(data []byte) []byte {
	firstHash := sha256.Sum256(data)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:]
}

// HashBlockHeader computes the hash of a block header
func HashBlockHeader(header *types.BlockHeader) []byte {
	serialized := header.Serialize()
	return DoubleHashBytes(serialized)
}
