package merkle

import (
	"bytes"
	"testing"

	"github.com/yourusername/bt/internal/crypto"
)

func TestBuildMerkleRoot_Empty(t *testing.T) {
	root := BuildMerkleRoot([][]byte{})
	expected := make([]byte, 32)

	if !bytes.Equal(root, expected) {
		t.Error("Empty merkle root should be 32 zero bytes")
	}
}

func TestBuildMerkleRoot_SingleTx(t *testing.T) {
	tx := []byte("single transaction")
	txHash := crypto.HashBytes(tx)

	root := BuildMerkleRoot([][]byte{txHash})

	if !bytes.Equal(root, txHash) {
		t.Error("Single tx merkle root should equal the tx hash itself")
	}
}

func TestBuildMerkleRoot_TwoTx(t *testing.T) {
	tx1 := []byte("transaction 1")
	tx2 := []byte("transaction 2")

	txHash1 := crypto.HashBytes(tx1)
	txHash2 := crypto.HashBytes(tx2)

	root := BuildMerkleRoot([][]byte{txHash1, txHash2})

	// Manually compute expected root
	combined := append(txHash1, txHash2...)
	expected := crypto.DoubleHashBytes(combined)

	if !bytes.Equal(root, expected) {
		t.Errorf("Two tx merkle root incorrect\nGot:  %x\nWant: %x", root, expected)
	}
}

func TestBuildMerkleRoot_OddNumber(t *testing.T) {
	// Test with 3 transactions (odd number)
	tx1 := []byte("tx1")
	tx2 := []byte("tx2")
	tx3 := []byte("tx3")

	txHash1 := crypto.HashBytes(tx1)
	txHash2 := crypto.HashBytes(tx2)
	txHash3 := crypto.HashBytes(tx3)

	root := BuildMerkleRoot([][]byte{txHash1, txHash2, txHash3})

	// Should duplicate the last hash
	// Level 1: [hash12, hash33]
	combined12 := append(txHash1, txHash2...)
	hash12 := crypto.DoubleHashBytes(combined12)

	combined33 := append(txHash3, txHash3...)
	hash33 := crypto.DoubleHashBytes(combined33)

	// Level 2: root
	combinedRoot := append(hash12, hash33...)
	expected := crypto.DoubleHashBytes(combinedRoot)

	if !bytes.Equal(root, expected) {
		t.Error("Odd number merkle root calculation incorrect")
	}
}

func TestBuildMerkleRoot_FourTx(t *testing.T) {
	txHashes := make([][]byte, 4)
	for i := 0; i < 4; i++ {
		txHashes[i] = crypto.HashBytes([]byte{byte(i)})
	}

	root := BuildMerkleRoot(txHashes)

	// Verify root is 32 bytes
	if len(root) != 32 {
		t.Errorf("Merkle root length = %d, want 32", len(root))
	}

	// Verify determinism
	root2 := BuildMerkleRoot(txHashes)
	if !bytes.Equal(root, root2) {
		t.Error("Merkle root is not deterministic")
	}
}

func TestBuildMerkleRoot_ChangeSensitivity(t *testing.T) {
	tx1 := []byte("tx1")
	tx2 := []byte("tx2")

	txHash1 := crypto.HashBytes(tx1)
	txHash2 := crypto.HashBytes(tx2)

	root1 := BuildMerkleRoot([][]byte{txHash1, txHash2})

	// Change one transaction
	tx2Modified := []byte("tx2_modified")
	txHash2Modified := crypto.HashBytes(tx2Modified)

	root2 := BuildMerkleRoot([][]byte{txHash1, txHash2Modified})

	if bytes.Equal(root1, root2) {
		t.Error("Merkle root should change when transaction changes")
	}
}

func TestBuildMerkleRoot_OrderMatters(t *testing.T) {
	tx1 := []byte("tx1")
	tx2 := []byte("tx2")

	txHash1 := crypto.HashBytes(tx1)
	txHash2 := crypto.HashBytes(tx2)

	root1 := BuildMerkleRoot([][]byte{txHash1, txHash2})
	root2 := BuildMerkleRoot([][]byte{txHash2, txHash1})

	if bytes.Equal(root1, root2) {
		t.Error("Merkle root should differ when transaction order changes")
	}
}

func TestBuildMerkleTree(t *testing.T) {
	txHashes := make([][]byte, 4)
	for i := 0; i < 4; i++ {
		txHashes[i] = crypto.HashBytes([]byte{byte(i)})
	}

	tree := BuildMerkleTree(txHashes)

	// Tree should have 3 levels: [4, 2, 1]
	if len(tree) != 3 {
		t.Errorf("Tree levels = %d, want 3", len(tree))
	}

	// Level 0 should have 4 nodes
	if len(tree[0]) != 4 {
		t.Errorf("Level 0 nodes = %d, want 4", len(tree[0]))
	}

	// Level 1 should have 2 nodes
	if len(tree[1]) != 2 {
		t.Errorf("Level 1 nodes = %d, want 2", len(tree[1]))
	}

	// Level 2 (root) should have 1 node
	if len(tree[2]) != 1 {
		t.Errorf("Level 2 nodes = %d, want 1", len(tree[2]))
	}

	// Root from tree should match BuildMerkleRoot
	root := BuildMerkleRoot(txHashes)
	if !bytes.Equal(tree[2][0], root) {
		t.Error("Tree root doesn't match BuildMerkleRoot result")
	}
}

func TestBuildMerkleTree_LargeTxSet(t *testing.T) {
	// Test with many transactions
	txHashes := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		txHashes[i] = crypto.HashBytes([]byte{byte(i), byte(i >> 8)})
	}

	root := BuildMerkleRoot(txHashes)

	if len(root) != 32 {
		t.Errorf("Root length = %d, want 32", len(root))
	}

	// Verify determinism
	root2 := BuildMerkleRoot(txHashes)
	if !bytes.Equal(root, root2) {
		t.Error("Large merkle root is not deterministic")
	}
}

func BenchmarkBuildMerkleRoot_4Tx(b *testing.B) {
	txHashes := make([][]byte, 4)
	for i := 0; i < 4; i++ {
		txHashes[i] = crypto.HashBytes([]byte{byte(i)})
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		BuildMerkleRoot(txHashes)
	}
}

func BenchmarkBuildMerkleRoot_100Tx(b *testing.B) {
	txHashes := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		txHashes[i] = crypto.HashBytes([]byte{byte(i), byte(i >> 8)})
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		BuildMerkleRoot(txHashes)
	}
}

func BenchmarkBuildMerkleTree_100Tx(b *testing.B) {
	txHashes := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		txHashes[i] = crypto.HashBytes([]byte{byte(i), byte(i >> 8)})
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		BuildMerkleTree(txHashes)
	}
}
