package pow

import (
	"bytes"
	"fmt"
	"math"
	"math/big"

	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/pkg/types"
)

const (
	// MaxNonce is the maximum value for nonce
	MaxNonce = math.MaxUint32

	// TargetBits is the default difficulty (will be adjusted dynamically)
	// Lower bits = harder difficulty
	// 16 bits = very easy (for testing)
	// 24 bits = Bitcoin's original difficulty
	DefaultTargetBits = 16
)

// ProofOfWork represents a proof-of-work algorithm
type ProofOfWork struct {
	Block  *types.Block
	Target *big.Int
}

// NewProofOfWork creates a new PoW instance for a block
func NewProofOfWork(block *types.Block) *ProofOfWork {
	target := big.NewInt(1)
	// Shift left by (256 - targetBits) to get the target value
	// The block hash must be less than this target
	targetBits := block.Header.DifficultyTarget
	target.Lsh(target, uint(256-targetBits))

	return &ProofOfWork{
		Block:  block,
		Target: target,
	}
}

// Mine performs the proof-of-work mining
// Returns the nonce and hash that satisfy the difficulty requirement
func (pow *ProofOfWork) Mine() (uint32, []byte) {
	var hashInt big.Int
	var hash []byte
	nonce := uint32(0)

	fmt.Printf("Mining block with difficulty target: %d bits\n", pow.Block.Header.DifficultyTarget)

	for nonce < MaxNonce {
		// Set the nonce in the header
		pow.Block.Header.Nonce = nonce

		// Calculate the hash
		hash = crypto.HashBlockHeader(&pow.Block.Header)

		// Convert hash to big.Int for comparison
		hashInt.SetBytes(hash)

		// Check if hash is less than target
		if hashInt.Cmp(pow.Target) == -1 {
			fmt.Printf("✓ Block mined! Nonce: %d, Hash: %x\n", nonce, hash)
			break
		}

		nonce++
	}

	return nonce, hash
}

// Validate checks if the block's proof-of-work is valid
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	hash := crypto.HashBlockHeader(&pow.Block.Header)
	hashInt.SetBytes(hash)

	return hashInt.Cmp(pow.Target) == -1
}

// IsValidHash checks if a hash meets the difficulty requirement
func IsValidHash(hash []byte, difficultyTarget uint32) bool {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-difficultyTarget))

	var hashInt big.Int
	hashInt.SetBytes(hash)

	return hashInt.Cmp(target) == -1
}

// CompareHashes checks if hash1 < hash2
func CompareHashes(hash1, hash2 []byte) int {
	return bytes.Compare(hash1, hash2)
}
