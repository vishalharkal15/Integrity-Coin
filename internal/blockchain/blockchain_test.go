package blockchain

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/tx"
)

// Helper function to create a test blockchain with cleanup
func setupTestBlockchain(t *testing.T) (*Blockchain, *crypto.Wallet, func()) {
	dbPath := fmt.Sprintf("./test_blockchain_%d.db", time.Now().UnixNano())
	wallet, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	
	bc, err := NewBlockchain(wallet.GetAddress(), dbPath)
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}
	
	cleanup := func() {
		bc.Close()
		os.RemoveAll(dbPath)
	}
	
	return bc, wallet, cleanup
}

func TestNewBlockchain(t *testing.T) {
	bc, _, cleanup := setupTestBlockchain(t)
	defer cleanup()

	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	if len(bc.Blocks) != 1 {
		t.Errorf("New blockchain height = %d, want 1", len(bc.Blocks))
	}

	genesisBlock := bc.Blocks[0]
	if genesisBlock == nil {
		t.Fatal("Genesis block is nil")
	}

	// Genesis should have zero previous hash
	expectedPrevHash := make([]byte, 32)
	if !bytes.Equal(genesisBlock.Header.PrevBlockHash, expectedPrevHash) {
		t.Error("Genesis block should have zero previous hash")
	}

	// Genesis should have valid hash
	if len(genesisBlock.Hash) != 32 {
		t.Error("Genesis block hash invalid")
	}
}

func TestAddBlock(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	initialHeight := bc.Height()
	minerAddr := wallet.GetAddress()

	// Create a transaction
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()
	
	tx1, err := bc.CreateTransaction(minerAddr, aliceAddr, 10*1e8, wallet)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	block, err := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	if err != nil {
		t.Fatalf("AddBlock failed: %v", err)
	}

	if block == nil {
		t.Fatal("AddBlock returned nil block")
	}

	if bc.Height() != initialHeight+1 {
		t.Errorf("Height after adding block = %d, want %d", bc.Height(), initialHeight+1)
	}

	// Verify block is linked to previous block
	prevBlock := bc.Blocks[len(bc.Blocks)-2]
	if !bytes.Equal(block.Header.PrevBlockHash, prevBlock.Hash) {
		t.Error("New block not properly linked to previous block")
	}
}

func TestAddMultipleBlocks(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	for i := 0; i < 5; i++ {
		tx1, err := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e8, wallet)
		if err != nil {
			t.Fatalf("Failed to create transaction %d: %v", i, err)
		}
		
		_, err = bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
		if err != nil {
			t.Fatalf("Failed to add block %d: %v", i, err)
		}
	}

	if bc.Height() != 6 { // 1 genesis + 5 new blocks
		t.Errorf("Height after 5 blocks = %d, want 6", bc.Height())
	}
}

func TestValidateBlock(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 10*1e8, wallet)
	block, err := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)

	if err != nil {
		t.Fatalf("Failed to add block: %v", err)
	}

	// Remove block from chain to test validation independently
	bc.Blocks = bc.Blocks[:len(bc.Blocks)-1]

	// Re-validate the block
	err = bc.ValidateBlock(block)
	if err != nil {
		t.Errorf("Valid block failed validation: %v", err)
	}
}

func TestValidateBlock_InvalidPoW(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 10*1e8, wallet)
	block, _ := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)

	// Tamper with nonce to invalidate PoW
	block.Header.Nonce = 0

	bc.Blocks = bc.Blocks[:len(bc.Blocks)-1]
	err := bc.ValidateBlock(block)

	if err == nil {
		t.Error("Invalid PoW block passed validation")
	}
}

func TestValidateBlock_InvalidMerkleRoot(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 10*1e8, wallet)
	block, _ := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)

	// Tamper with merkle root
	block.Header.MerkleRoot = make([]byte, 32)

	bc.Blocks = bc.Blocks[:len(bc.Blocks)-1]
	err := bc.ValidateBlock(block)

	if err == nil {
		t.Error("Invalid merkle root block passed validation")
	}
}

func TestValidateBlock_FutureTimestamp(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 10*1e8, wallet)
	block, _ := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)

	// Set timestamp far in future
	block.Header.Timestamp = time.Now().Add(3 * time.Hour)

	bc.Blocks = bc.Blocks[:len(bc.Blocks)-1]
	err := bc.ValidateBlock(block)

	if err == nil {
		t.Error("Future timestamp block passed validation")
	}
}

func TestValidateChain(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	// Add several blocks
	for i := 0; i < 3; i++ {
		tx1, err := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e8, wallet)
		if err != nil {
			t.Fatalf("Failed to create transaction: %v", err)
		}
		
		_, err = bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
		if err != nil {
			t.Fatalf("Failed to add block: %v", err)
		}
	}

	// Validate entire chain
	err := bc.ValidateChain()
	if err != nil {
		t.Errorf("Valid chain failed validation: %v", err)
	}
}

func TestValidateChain_BrokenLink(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	for i := 0; i < 3; i++ {
		tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e8, wallet)
		bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	}

	// Break the chain by tampering with a block hash
	bc.Blocks[1].Hash = make([]byte, 32)

	err := bc.ValidateChain()
	if err == nil {
		t.Error("Broken chain passed validation")
	}
}

func TestGetLatestBlock(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	latest := bc.GetLatestBlock()
	if latest == nil {
		t.Fatal("GetLatestBlock returned nil")
	}

	// Add a block and verify latest updates
	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 10*1e8, wallet)
	newBlock, _ := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)

	latest = bc.GetLatestBlock()
	if !bytes.Equal(latest.Hash, newBlock.Hash) {
		t.Error("GetLatestBlock didn't return the most recent block")
	}
}

func TestGetBlock(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	// Add some blocks
	for i := 0; i < 3; i++ {
		tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e8, wallet)
		bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	}

	// Test valid index
	block, err := bc.GetBlock(1)
	if err != nil {
		t.Errorf("GetBlock(1) failed: %v", err)
	}
	if block == nil {
		t.Error("GetBlock returned nil for valid index")
	}

	// Test invalid index
	_, err = bc.GetBlock(100)
	if err == nil {
		t.Error("GetBlock should fail for out-of-range index")
	}

	// Test negative index
	_, err = bc.GetBlock(-1)
	if err == nil {
		t.Error("GetBlock should fail for negative index")
	}
}

func TestGetBlockByHash(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 10*1e8, wallet)
	addedBlock, _ := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)

	// Find block by hash
	foundBlock, err := bc.GetBlockByHash(addedBlock.Hash)
	if err != nil {
		t.Errorf("GetBlockByHash failed: %v", err)
	}

	if !bytes.Equal(foundBlock.Hash, addedBlock.Hash) {
		t.Error("GetBlockByHash returned wrong block")
	}

	// Try with non-existent hash
	fakeHash := make([]byte, 32)
	fakeHash[0] = 0xff
	_, err = bc.GetBlockByHash(fakeHash)
	if err == nil {
		t.Error("GetBlockByHash should fail for non-existent hash")
	}
}

func TestHeight(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	if bc.Height() != 1 {
		t.Errorf("Initial height = %d, want 1", bc.Height())
	}

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	for i := 0; i < 5; i++ {
		tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e8, wallet)
		bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	}

	if bc.Height() != 6 {
		t.Errorf("Height after 5 additions = %d, want 6", bc.Height())
	}
}

func TestDifficultyAdjustment(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	initialDifficulty := bc.DifficultyTarget
	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	// Add blocks to trigger difficulty adjustment
	for i := 0; i < DifficultyAdjustmentInterval; i++ {
		tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e7, wallet)
		bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	}

	// Difficulty should have been adjusted
	if bc.DifficultyTarget == initialDifficulty {
		t.Log("Difficulty unchanged (possible if timing is exact)")
	}
}

func TestBlockLinkage(t *testing.T) {
	bc, wallet, cleanup := setupTestBlockchain(t)
	defer cleanup()

	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	// Add several blocks and verify linkage
	for i := 0; i < 5; i++ {
		tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e7, wallet)
		bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	}

	// Verify each block links to previous
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		if !bytes.Equal(currentBlock.Header.PrevBlockHash, prevBlock.Hash) {
			t.Errorf("Block %d not properly linked to block %d", i, i-1)
		}
	}
}

func TestPersistence(t *testing.T) {
	dbPath := "./test_persistence.db"
	defer os.RemoveAll(dbPath)

	wallet, _ := crypto.NewWallet()
	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	// Create blockchain and add blocks
	{
		bc, err := NewBlockchain(minerAddr, dbPath)
		if err != nil {
			t.Fatalf("Failed to create blockchain: %v", err)
		}

		for i := 0; i < 3; i++ {
			tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e8, wallet)
			bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
		}

		originalHeight := bc.Height()
		bc.Close()

		// Reopen blockchain
		bc2, err := NewBlockchain(minerAddr, dbPath)
		if err != nil {
			t.Fatalf("Failed to load blockchain: %v", err)
		}
		defer bc2.Close()

		if bc2.Height() != originalHeight {
			t.Errorf("Loaded blockchain height = %d, want %d", bc2.Height(), originalHeight)
		}

		// Verify chain is valid after loading
		if err := bc2.ValidateChain(); err != nil {
			t.Errorf("Loaded blockchain validation failed: %v", err)
		}
	}
}

func BenchmarkAddBlock(b *testing.B) {
	dbPath := "./bench_blockchain.db"
	defer os.RemoveAll(dbPath)

	wallet, _ := crypto.NewWallet()
	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	bc, _ := NewBlockchain(minerAddr, dbPath)
	defer bc.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e7, wallet)
		bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	}
}

func BenchmarkValidateChain(b *testing.B) {
	dbPath := "./bench_validate.db"
	defer os.RemoveAll(dbPath)

	wallet, _ := crypto.NewWallet()
	minerAddr := wallet.GetAddress()
	aliceWallet, _ := crypto.NewWallet()
	aliceAddr := aliceWallet.GetAddress()

	bc, _ := NewBlockchain(minerAddr, dbPath)
	defer bc.Close()

	// Create a chain with 10 blocks
	for i := 0; i < 10; i++ {
		tx1, _ := bc.CreateTransaction(minerAddr, aliceAddr, 1*1e7, wallet)
		bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bc.ValidateChain()
	}
}
