package types

import (
	"bytes"
	"encoding/binary"
	"time"
)

// BlockHeader contains the block metadata
type BlockHeader struct {
	Version          uint32    // Block version
	PrevBlockHash    []byte    // Previous block hash (32 bytes)
	MerkleRoot       []byte    // Merkle root of transactions (32 bytes)
	Timestamp        time.Time // Block creation time
	DifficultyTarget uint32    // Compact difficulty target
	Nonce            uint32    // Nonce for PoW
}

// Serialize converts BlockHeader to bytes for hashing
func (h *BlockHeader) Serialize() []byte {
	var buf bytes.Buffer

	// Version (4 bytes)
	binary.Write(&buf, binary.LittleEndian, h.Version)

	// PrevBlockHash (32 bytes)
	buf.Write(h.PrevBlockHash)

	// MerkleRoot (32 bytes)
	buf.Write(h.MerkleRoot)

	// Timestamp (8 bytes - Unix timestamp)
	binary.Write(&buf, binary.LittleEndian, h.Timestamp.Unix())

	// DifficultyTarget (4 bytes)
	binary.Write(&buf, binary.LittleEndian, h.DifficultyTarget)

	// Nonce (4 bytes)
	binary.Write(&buf, binary.LittleEndian, h.Nonce)

	return buf.Bytes()
}

// Block represents a complete block with header and transactions
// Note: Transactions is interface{} to avoid circular imports
// It will be []*tx.Transaction in practice
type Block struct {
	Header       BlockHeader
	Transactions interface{} // []*tx.Transaction
	Hash         []byte      // Block hash (cached)
}
