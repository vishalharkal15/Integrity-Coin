package pow

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/pkg/types"
)

func createTestBlock(difficulty uint32) *types.Block {
	return &types.Block{
		Header: types.BlockHeader{
			Version:          1,
			PrevBlockHash:    make([]byte, 32),
			MerkleRoot:       make([]byte, 32),
			Timestamp:        time.Now(),
			DifficultyTarget: difficulty,
			Nonce:            0,
		},
		Transactions: [][]byte{[]byte("test transaction")},
	}
}

func TestNewProofOfWork(t *testing.T) {
	block := createTestBlock(16)
	pow := NewProofOfWork(block)

	if pow.Block != block {
		t.Error("PoW block reference incorrect")
	}

	if pow.Target == nil {
		t.Error("PoW target not initialized")
	}

	// Verify target calculation
	expectedTarget := big.NewInt(1)
	expectedTarget.Lsh(expectedTarget, uint(256-16))

	if pow.Target.Cmp(expectedTarget) != 0 {
		t.Error("PoW target calculation incorrect")
	}
}

func TestProofOfWork_Mine_EasyDifficulty(t *testing.T) {
	block := createTestBlock(8) // Very easy difficulty for testing
	pow := NewProofOfWork(block)

	nonce, hash := pow.Mine()

	// Check that a valid nonce was found
	if nonce == 0 && len(hash) == 0 {
		t.Error("Mining failed to find nonce")
	}

	// Verify the hash meets difficulty requirement
	var hashInt big.Int
	hashInt.SetBytes(hash)

	if hashInt.Cmp(pow.Target) >= 0 {
		t.Error("Mined hash doesn't meet difficulty requirement")
	}

	// Verify the block was updated
	if block.Header.Nonce != nonce {
		t.Error("Block nonce not updated after mining")
	}
}

func TestProofOfWork_Validate(t *testing.T) {
	block := createTestBlock(12)
	pow := NewProofOfWork(block)

	// Mine the block
	nonce, hash := pow.Mine()
	block.Header.Nonce = nonce
	block.Hash = hash

	// Validate should pass
	if !pow.Validate() {
		t.Error("Valid PoW failed validation")
	}

	// Change nonce, should fail validation
	block.Header.Nonce = nonce + 1
	if pow.Validate() {
		t.Error("Invalid PoW passed validation")
	}
}

func TestIsValidHash(t *testing.T) {
	tests := []struct {
		name             string
		hash             []byte
		difficultyTarget uint32
		expectedValid    bool
	}{
		{
			name:             "Easy hash passes",
			hash:             make([]byte, 32), // All zeros
			difficultyTarget: 8,
			expectedValid:    true,
		},
		{
			name:             "Hard hash fails",
			hash:             []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			difficultyTarget: 32,
			expectedValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidHash(tt.hash, tt.difficultyTarget)
			if result != tt.expectedValid {
				t.Errorf("IsValidHash() = %v, want %v", result, tt.expectedValid)
			}
		})
	}
}

func TestCompareHashes(t *testing.T) {
	hash1 := []byte{0x00, 0x00, 0x00, 0x01}
	hash2 := []byte{0x00, 0x00, 0x00, 0x02}
	hash3 := []byte{0x00, 0x00, 0x00, 0x01}

	if CompareHashes(hash1, hash2) >= 0 {
		t.Error("hash1 should be less than hash2")
	}

	if CompareHashes(hash2, hash1) <= 0 {
		t.Error("hash2 should be greater than hash1")
	}

	if CompareHashes(hash1, hash3) != 0 {
		t.Error("hash1 should equal hash3")
	}
}

func TestProofOfWork_DifferentDifficulties(t *testing.T) {
	difficulties := []uint32{8, 12, 16}

	for _, difficulty := range difficulties {
		t.Run("", func(t *testing.T) {
			block := createTestBlock(difficulty)
			pow := NewProofOfWork(block)

			nonce, hash := pow.Mine()

			if !IsValidHash(hash, difficulty) {
				t.Errorf("Mined hash doesn't meet difficulty %d", difficulty)
			}

			block.Header.Nonce = nonce
			if !pow.Validate() {
				t.Errorf("Valid PoW failed validation at difficulty %d", difficulty)
			}
		})
	}
}

func TestProofOfWork_HashConsistency(t *testing.T) {
	block := createTestBlock(16)
	
	// Hash should be consistent for same block header
	hash1 := crypto.HashBlockHeader(&block.Header)
	hash2 := crypto.HashBlockHeader(&block.Header)

	if !bytes.Equal(hash1, hash2) {
		t.Error("Block header hashing is not consistent")
	}
}

func TestProofOfWork_NonceIncrement(t *testing.T) {
	block := createTestBlock(16)
	pow := NewProofOfWork(block)

	nonce, _ := pow.Mine()

	// Nonce should have been incremented during mining
	if nonce == 0 {
		// This could happen but is extremely unlikely with difficulty 16
		t.Log("Warning: nonce is 0, might be extremely lucky")
	}

	if nonce > MaxNonce {
		t.Errorf("Nonce %d exceeds MaxNonce %d", nonce, MaxNonce)
	}
}

// Benchmark mining with different difficulties
func BenchmarkMine_Difficulty8(b *testing.B) {
	for i := 0; i < b.N; i++ {
		block := createTestBlock(8)
		pow := NewProofOfWork(block)
		pow.Mine()
	}
}

func BenchmarkMine_Difficulty12(b *testing.B) {
	for i := 0; i < b.N; i++ {
		block := createTestBlock(12)
		pow := NewProofOfWork(block)
		pow.Mine()
	}
}

func BenchmarkValidate(b *testing.B) {
	block := createTestBlock(16)
	pow := NewProofOfWork(block)
	nonce, hash := pow.Mine()
	block.Header.Nonce = nonce
	block.Hash = hash

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pow.Validate()
	}
}
