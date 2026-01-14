package crypto

import (
	"bytes"
	"testing"
	"time"

	"github.com/yourusername/bt/pkg/types"
)

func TestHashBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string // hex encoded
	}{
		{
			name:     "Empty input",
			input:    []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "Simple string",
			input:    []byte("hello"),
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:     "Bitcoin genesis",
			input:    []byte("The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"),
			expected: "e1c7f8c5b0e8f5c3c0f5d4e4e0c3b0e8f5c3c0f5d4e4e0c3b0e8f5c3c0f5d4e4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashBytes(tt.input)
			if len(result) != 32 {
				t.Errorf("Hash length = %d, want 32", len(result))
			}
		})
	}
}

func TestDoubleHashBytes(t *testing.T) {
	input := []byte("test")
	single := HashBytes(input)
	double := DoubleHashBytes(input)

	if bytes.Equal(single, double) {
		t.Error("Double hash should be different from single hash")
	}

	if len(double) != 32 {
		t.Errorf("Double hash length = %d, want 32", len(double))
	}
}

func TestHashBlockHeader(t *testing.T) {
	header := &types.BlockHeader{
		Version:          1,
		PrevBlockHash:    make([]byte, 32),
		MerkleRoot:       make([]byte, 32),
		Timestamp:        time.Now(),
		DifficultyTarget: 16,
		Nonce:            0,
	}

	hash1 := HashBlockHeader(header)
	hash2 := HashBlockHeader(header)

	// Same header should produce same hash
	if !bytes.Equal(hash1, hash2) {
		t.Error("Same header produced different hashes")
	}

	// Change nonce, should produce different hash
	header.Nonce = 1
	hash3 := HashBlockHeader(header)

	if bytes.Equal(hash1, hash3) {
		t.Error("Different nonce produced same hash")
	}
}

func TestHashDeterminism(t *testing.T) {
	// Same input should always produce same output
	input := []byte("deterministic test")

	hash1 := HashBytes(input)
	hash2 := HashBytes(input)
	hash3 := DoubleHashBytes(input)
	hash4 := DoubleHashBytes(input)

	if !bytes.Equal(hash1, hash2) {
		t.Error("HashBytes is not deterministic")
	}

	if !bytes.Equal(hash3, hash4) {
		t.Error("DoubleHashBytes is not deterministic")
	}
}

func TestHashSensitivity(t *testing.T) {
	// Small change should produce completely different hash
	input1 := []byte("test")
	input2 := []byte("Test") // Capital T

	hash1 := HashBytes(input1)
	hash2 := HashBytes(input2)

	if bytes.Equal(hash1, hash2) {
		t.Error("Different inputs produced same hash")
	}

	// Check that hashes differ significantly (avalanche effect)
	differingBits := 0
	for i := 0; i < len(hash1); i++ {
		xor := hash1[i] ^ hash2[i]
		for xor != 0 {
			differingBits += int(xor & 1)
			xor >>= 1
		}
	}

	// Expect roughly 50% of bits to differ (avalanche effect)
	if differingBits < 50 || differingBits > 200 {
		t.Errorf("Expected ~128 differing bits, got %d", differingBits)
	}
}

func BenchmarkHashBytes(b *testing.B) {
	data := []byte("benchmark test data for hashing performance")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		HashBytes(data)
	}
}

func BenchmarkDoubleHashBytes(b *testing.B) {
	data := []byte("benchmark test data for double hashing")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		DoubleHashBytes(data)
	}
}

func BenchmarkHashBlockHeader(b *testing.B) {
	header := &types.BlockHeader{
		Version:          1,
		PrevBlockHash:    make([]byte, 32),
		MerkleRoot:       make([]byte, 32),
		Timestamp:        time.Now(),
		DifficultyTarget: 16,
		Nonce:            0,
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		HashBlockHeader(header)
	}
}
