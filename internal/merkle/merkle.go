package merkle

import (
	"github.com/yourusername/bt/internal/crypto"
)

// BuildMerkleRoot constructs a merkle root from transaction hashes
// If there's an odd number of hashes, the last one is duplicated
func BuildMerkleRoot(txHashes [][]byte) []byte {
	if len(txHashes) == 0 {
		return make([]byte, 32) // Empty merkle root
	}

	// Make a copy to avoid modifying the original slice
	tree := make([][]byte, len(txHashes))
	copy(tree, txHashes)

	// Build the tree bottom-up
	for len(tree) > 1 {
		// If odd number of nodes, duplicate the last one
		if len(tree)%2 != 0 {
			tree = append(tree, tree[len(tree)-1])
		}

		var newLevel [][]byte
		for i := 0; i < len(tree); i += 2 {
			// Concatenate two hashes and hash them
			combined := append(tree[i], tree[i+1]...)
			parentHash := crypto.DoubleHashBytes(combined)
			newLevel = append(newLevel, parentHash)
		}

		tree = newLevel
	}

	return tree[0]
}

// BuildMerkleTree builds the complete merkle tree and returns all levels
func BuildMerkleTree(txHashes [][]byte) [][][]byte {
	if len(txHashes) == 0 {
		return nil
	}

	var tree [][][]byte
	currentLevel := make([][]byte, len(txHashes))
	copy(currentLevel, txHashes)
	tree = append(tree, currentLevel)

	for len(currentLevel) > 1 {
		if len(currentLevel)%2 != 0 {
			currentLevel = append(currentLevel, currentLevel[len(currentLevel)-1])
		}

		var nextLevel [][]byte
		for i := 0; i < len(currentLevel); i += 2 {
			combined := append(currentLevel[i], currentLevel[i+1]...)
			parentHash := crypto.DoubleHashBytes(combined)
			nextLevel = append(nextLevel, parentHash)
		}

		currentLevel = nextLevel
		tree = append(tree, currentLevel)
	}

	return tree
}
