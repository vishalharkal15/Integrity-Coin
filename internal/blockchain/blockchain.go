package blockchain

import (
	"bytes"
	"fmt"
	"time"

	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/merkle"
	"github.com/yourusername/bt/internal/pow"
	"github.com/yourusername/bt/internal/storage"
	"github.com/yourusername/bt/internal/tx"
	"github.com/yourusername/bt/internal/utxo"
	"github.com/yourusername/bt/pkg/types"
)

const (
	// GenesisData is the data stored in the genesis block
	GenesisData = "Genesis Block - Bitcoin-like Cryptocurrency"

	// BlockGenerationInterval is the target time between blocks (in seconds)
	BlockGenerationInterval = 10 // 10 seconds for testing

	// DifficultyAdjustmentInterval is how often difficulty adjusts (in blocks)
	DifficultyAdjustmentInterval = 10 // Adjust every 10 blocks
)

// Blockchain represents the entire blockchain
type Blockchain struct {
	Blocks           []*types.Block
	DifficultyTarget uint32
	UTXOSet          *utxo.UTXOSet
	PendingTxs       []*tx.Transaction
	Storage          *storage.Storage
}
// NewBlockchain creates a new blockchain with a genesis block
func NewBlockchain(genesisAddress string, dbPath string) (*Blockchain, error) {
	// Open storage
	store, err := storage.NewStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage: %v", err)
	}

	utxoSet := utxo.NewUTXOSet()

	// Try to load existing blockchain
	tip, err := store.GetChainTip()
	if err == nil && len(tip) > 0 {
		// Blockchain exists, load it
		fmt.Println("📂 Loading existing blockchain from disk...")
		return loadBlockchain(store)
	}

	// Create new blockchain with genesis
	fmt.Println("🆕 Creating new blockchain...")
	genesisBlock := createGenesisBlock(genesisAddress, utxoSet)
	
	bc := &Blockchain{
		Blocks:           []*types.Block{genesisBlock},
		DifficultyTarget: pow.DefaultTargetBits,
		UTXOSet:          utxoSet,
		PendingTxs:       []*tx.Transaction{},
		Storage:          store,
	}

	// Save genesis block
	if err := bc.saveBlockToDB(genesisBlock); err != nil {
		return nil, fmt.Errorf("failed to save genesis block: %v", err)
	}

	return bc, nil
}

// loadBlockchain loads an existing blockchain from storage
func loadBlockchain(store *storage.Storage) (*Blockchain, error) {
	// Get blockchain metadata
	height, err := store.GetChainHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to get chain height: %v", err)
	}

	difficulty, err := store.GetDifficulty()
	if err != nil {
		return nil, fmt.Errorf("failed to get difficulty: %v", err)
	}

	// Load blocks in order by following the chain backwards from tip
	var blocks []*types.Block
	
	// Start with the tip
	currentHash, err := store.GetChainTip()
	if err != nil {
		return nil, fmt.Errorf("failed to get chain tip: %v", err)
	}

	// Walk backwards from tip to genesis
	for {
		block, err := store.GetBlock(currentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to load block %x: %v", currentHash, err)
		}

		blocks = append([]*types.Block{block}, blocks...) // Prepend block

		// Stop if we reached genesis (all zeros prev hash)
		zeroHash := make([]byte, 32)
		if bytes.Equal(block.Header.PrevBlockHash, zeroHash) {
			break
		}

		currentHash = block.Header.PrevBlockHash
	}

	// Rebuild UTXO set from all blocks in order
	utxoSet := utxo.NewUTXOSet()
	for _, block := range blocks {
		if txs, ok := block.Transactions.([]*tx.Transaction); ok {
			for _, transaction := range txs {
				utxoSet.Update(transaction)
			}
		}
	}

	fmt.Printf("✓ Loaded blockchain: %d blocks, difficulty %d bits\n", height, difficulty)

	return &Blockchain{
		Blocks:           blocks,
		DifficultyTarget: difficulty,
		UTXOSet:          utxoSet,
		PendingTxs:       []*tx.Transaction{},
		Storage:          store,
	}, nil
}

// createGenesisBlock creates the first block in the chain
func createGenesisBlock(genesisAddress string, utxoSet *utxo.UTXOSet) *types.Block {
	timestamp := time.Now()
	prevHash := make([]byte, 32) // All zeros for genesis

	// Create genesis coinbase transaction (mining reward)
	genesisTx, err := tx.NewCoinbaseTx(genesisAddress, GenesisData, 50*1e8) // 50 coins
	if err != nil {
		panic(fmt.Sprintf("Failed to create genesis transaction: %v", err))
	}

	// Update UTXO set with genesis transaction
	utxoSet.AddUTXO(genesisTx.ID, genesisTx.Outputs)

	// Serialize transactions
	transactions := []*tx.Transaction{genesisTx}
	txHashes := make([][]byte, len(transactions))
	for i, transaction := range transactions {
		txHashes[i] = transaction.ID
	}
	merkleRoot := merkle.BuildMerkleRoot(txHashes)

	block := &types.Block{
		Header: types.BlockHeader{
			Version:          1,
			PrevBlockHash:    prevHash,
			MerkleRoot:       merkleRoot,
			Timestamp:        timestamp,
			DifficultyTarget: pow.DefaultTargetBits,
			Nonce:            0,
		},
		Transactions: transactions,
	}

	// Mine the genesis block
	proofOfWork := pow.NewProofOfWork(block)
	nonce, hash := proofOfWork.Mine()

	block.Header.Nonce = nonce
	block.Hash = hash

	return block
}

// AddBlock mines and adds a new block with transactions to the blockchain
func (bc *Blockchain) AddBlock(transactions []*tx.Transaction, minerAddress string) (*types.Block, error) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]

	// Add coinbase transaction (mining reward)
	coinbaseTx, err := tx.NewCoinbaseTx(minerAddress, fmt.Sprintf("Block %d reward", len(bc.Blocks)), 50*1e8)
	if err != nil {
		return nil, fmt.Errorf("failed to create coinbase: %v", err)
	}

	// Add coinbase as first transaction
	allTxs := append([]*tx.Transaction{coinbaseTx}, transactions...)

	// Validate all non-coinbase transactions
	for _, transaction := range transactions {
		if !bc.VerifyTransaction(transaction) {
			return nil, fmt.Errorf("invalid transaction: %x", transaction.ID)
		}
	}

	// Adjust difficulty if needed
	bc.adjustDifficulty()

	// Build merkle root from transaction IDs
	txHashes := make([][]byte, len(allTxs))
	for i, transaction := range allTxs {
		txHashes[i] = transaction.ID
	}
	merkleRoot := merkle.BuildMerkleRoot(txHashes)

	// Create new block
	newBlock := &types.Block{
		Header: types.BlockHeader{
			Version:          1,
			PrevBlockHash:    prevBlock.Hash,
			MerkleRoot:       merkleRoot,
			Timestamp:        time.Now(),
			DifficultyTarget: bc.DifficultyTarget,
			Nonce:            0,
		},
		Transactions: allTxs,
	}

	// Mine the block
	proofOfWork := pow.NewProofOfWork(newBlock)
	nonce, hash := proofOfWork.Mine()

	newBlock.Header.Nonce = nonce
	newBlock.Hash = hash

	// Validate before adding
	if err := bc.ValidateBlock(newBlock); err != nil {
		return nil, fmt.Errorf("block validation failed: %v", err)
	}

	// Update UTXO set with all transactions
	for _, transaction := range allTxs {
		if err := bc.UTXOSet.Update(transaction); err != nil {
			return nil, fmt.Errorf("failed to update UTXO set: %v", err)
		}
	}

	bc.Blocks = append(bc.Blocks, newBlock)
	
	// Save to database
	if err := bc.saveBlockToDB(newBlock); err != nil {
		return nil, fmt.Errorf("failed to save block: %v", err)
	}
	
	return newBlock, nil
}

// saveBlockToDB saves a block and updates chain metadata
func (bc *Blockchain) saveBlockToDB(block *types.Block) error {
	if bc.Storage == nil {
		return nil // Storage not enabled
	}

	// Save the block
	if err := bc.Storage.SaveBlock(block); err != nil {
		return err
	}

	// Update chain tip
	if err := bc.Storage.SaveChainTip(block.Hash); err != nil {
		return err
	}

	// Update height
	if err := bc.Storage.SaveChainHeight(len(bc.Blocks)); err != nil {
		return err
	}

	// Update difficulty
	if err := bc.Storage.SaveDifficulty(bc.DifficultyTarget); err != nil {
		return err
	}

	// Save UTXOs
	if txs, ok := block.Transactions.([]*tx.Transaction); ok {
		for _, transaction := range txs {
			for i, output := range transaction.Outputs {
				if err := bc.Storage.SaveUTXO(transaction.ID, i, output); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Close closes the blockchain storage
func (bc *Blockchain) Close() error {
	if bc.Storage != nil {
		return bc.Storage.Close()
	}
	return nil
}

// ValidateBlock validates a single block
func (bc *Blockchain) ValidateBlock(block *types.Block) error {
	// 1. Validate proof-of-work
	proofOfWork := pow.NewProofOfWork(block)
	if !proofOfWork.Validate() {
		return fmt.Errorf("invalid proof-of-work")
	}

	// 2. Verify block hash
	computedHash := crypto.HashBlockHeader(&block.Header)
	if !bytes.Equal(computedHash, block.Hash) {
		return fmt.Errorf("block hash mismatch")
	}

	// Cast transactions
	transactions, ok := block.Transactions.([]*tx.Transaction)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}

	// 3. Verify merkle root
	txHashes := make([][]byte, len(transactions))
	for i, transaction := range transactions {
		txHashes[i] = transaction.ID
	}
	computedMerkleRoot := merkle.BuildMerkleRoot(txHashes)
	if !bytes.Equal(computedMerkleRoot, block.Header.MerkleRoot) {
		return fmt.Errorf("merkle root mismatch")
	}

	// 4. Verify previous block hash (if not genesis)
	if len(bc.Blocks) > 0 {
		prevBlock := bc.Blocks[len(bc.Blocks)-1]
		if !bytes.Equal(block.Header.PrevBlockHash, prevBlock.Hash) {
			return fmt.Errorf("previous block hash mismatch")
		}
	}

	// 5. Verify timestamp is reasonable
	if block.Header.Timestamp.After(time.Now().Add(2 * time.Hour)) {
		return fmt.Errorf("block timestamp too far in future")
	}

	// 6. Verify first transaction is coinbase
	if len(transactions) == 0 {
		return fmt.Errorf("block has no transactions")
	}
	if !transactions[0].IsCoinbase() {
		return fmt.Errorf("first transaction is not coinbase")
	}

	// 7. Verify only one coinbase
	for i := 1; i < len(transactions); i++ {
		if transactions[i].IsCoinbase() {
			return fmt.Errorf("multiple coinbase transactions")
		}
	}

	return nil
}

// ValidateChain validates the entire blockchain
func (bc *Blockchain) ValidateChain() error {
	for i := 1; i < len(bc.Blocks); i++ {
		block := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		// Validate proof-of-work
		proofOfWork := pow.NewProofOfWork(block)
		if !proofOfWork.Validate() {
			return fmt.Errorf("invalid PoW at block %d", i)
		}

		// Validate previous hash linkage
		if !bytes.Equal(block.Header.PrevBlockHash, prevBlock.Hash) {
			return fmt.Errorf("broken chain at block %d", i)
		}

		// Cast transactions
		transactions, ok := block.Transactions.([]*tx.Transaction)
		if !ok {
			return fmt.Errorf("invalid transaction type at block %d", i)
		}

		// Validate merkle root
		txHashes := make([][]byte, len(transactions))
		for j, transaction := range transactions {
			txHashes[j] = transaction.ID
		}
		computedMerkleRoot := merkle.BuildMerkleRoot(txHashes)
		if !bytes.Equal(computedMerkleRoot, block.Header.MerkleRoot) {
			return fmt.Errorf("invalid merkle root at block %d", i)
		}
	}

	return nil
}

// adjustDifficulty adjusts the mining difficulty based on block generation time
func (bc *Blockchain) adjustDifficulty() {
	blockCount := len(bc.Blocks)

	// Only adjust at intervals
	if blockCount%DifficultyAdjustmentInterval != 0 {
		return
	}

	// Need at least the adjustment interval blocks
	if blockCount < DifficultyAdjustmentInterval {
		return
	}

	// Calculate time taken for last interval
	lastAdjustmentBlock := bc.Blocks[blockCount-DifficultyAdjustmentInterval]
	currentBlock := bc.Blocks[blockCount-1]

	timeTaken := currentBlock.Header.Timestamp.Sub(lastAdjustmentBlock.Header.Timestamp).Seconds()
	expectedTime := float64(DifficultyAdjustmentInterval * BlockGenerationInterval)

	// Adjust difficulty
	if timeTaken < expectedTime/2 {
		// Blocks generated too fast, increase difficulty
		if bc.DifficultyTarget < 32 {
			bc.DifficultyTarget++
			fmt.Printf("⚡ Difficulty increased to %d bits\n", bc.DifficultyTarget)
		}
	} else if timeTaken > expectedTime*2 {
		// Blocks generated too slow, decrease difficulty
		if bc.DifficultyTarget > 8 {
			bc.DifficultyTarget--
			fmt.Printf("⚡ Difficulty decreased to %d bits\n", bc.DifficultyTarget)
		}
	}
}

// GetLatestBlock returns the most recent block
func (bc *Blockchain) GetLatestBlock() *types.Block {
	return bc.Blocks[len(bc.Blocks)-1]
}

// GetBlock returns a block by index
func (bc *Blockchain) GetBlock(index int) (*types.Block, error) {
	if index < 0 || index >= len(bc.Blocks) {
		return nil, fmt.Errorf("block index out of range")
	}
	return bc.Blocks[index], nil
}

// GetBlockByHash finds a block by its hash
func (bc *Blockchain) GetBlockByHash(hash []byte) (*types.Block, error) {
	for _, block := range bc.Blocks {
		if bytes.Equal(block.Hash, hash) {
			return block, nil
		}
	}
	return nil, fmt.Errorf("block not found")
}

// Height returns the current blockchain height
func (bc *Blockchain) Height() int {
	return len(bc.Blocks)
}

// PrintChain prints the blockchain for debugging
func (bc *Blockchain) PrintChain() {
	fmt.Println("\n=== BLOCKCHAIN ===")
	for i, block := range bc.Blocks {
		fmt.Printf("\nBlock %d:\n", i)
		fmt.Printf("  Hash: %x\n", block.Hash)
		fmt.Printf("  Prev Hash: %x\n", block.Header.PrevBlockHash)
		fmt.Printf("  Merkle Root: %x\n", block.Header.MerkleRoot)
		fmt.Printf("  Timestamp: %s\n", block.Header.Timestamp.Format(time.RFC3339))
		fmt.Printf("  Difficulty: %d bits\n", block.Header.DifficultyTarget)
		fmt.Printf("  Nonce: %d\n", block.Header.Nonce)
		
		transactions, ok := block.Transactions.([]*tx.Transaction)
		if ok {
			fmt.Printf("  Transactions: %d\n", len(transactions))
			for j, transaction := range transactions {
				if transaction.IsCoinbase() {
					fmt.Printf("    [%d] Coinbase: %x\n", j, transaction.ID[:8])
				} else {
					fmt.Printf("    [%d] TX: %x\n", j, transaction.ID[:8])
				}
			}
		}
	}
	fmt.Println("==================")
}

// VerifyTransaction verifies a transaction's signatures
func (bc *Blockchain) VerifyTransaction(transaction *tx.Transaction) bool {
	if transaction.IsCoinbase() {
		return true
	}

	prevTxs := make(map[string]*tx.Transaction)

	for _, input := range transaction.Inputs {
		prevTx, err := bc.FindTransaction(input.TxID)
		if err != nil {
			return false
		}
		prevTxs[string(input.TxID)] = prevTx
	}

	return transaction.Verify(prevTxs)
}

// FindTransaction finds a transaction by ID
func (bc *Blockchain) FindTransaction(ID []byte) (*tx.Transaction, error) {
	for _, block := range bc.Blocks {
		transactions, ok := block.Transactions.([]*tx.Transaction)
		if !ok {
			continue
		}

		for _, transaction := range transactions {
			if bytes.Equal(transaction.ID, ID) {
				return transaction, nil
			}
		}
	}

	return nil, fmt.Errorf("transaction not found")
}

// CreateTransaction creates a new signed transaction
func (bc *Blockchain) CreateTransaction(from, to string, amount int64, wallet *crypto.Wallet) (*tx.Transaction, error) {
	// Find spendable outputs
	accumulated, validOutputs, err := bc.UTXOSet.FindSpendableOutputs(from, amount)
	if err != nil {
		return nil, err
	}

	// Build inputs
	var inputs []tx.TxInput
	for txID, outputs := range validOutputs {
		for _, outIdx := range outputs {
			input := tx.TxInput{
				TxID:      []byte(txID),
				OutIndex:  outIdx,
				Signature: nil,
				PubKey:    wallet.PublicKey,
			}
			inputs = append(inputs, input)
		}
	}

	// Build outputs
	var outputs []tx.TxOutput

	// Output to recipient
	recipientOutput := tx.TxOutput{
		Value: amount,
	}
	if err := recipientOutput.Lock(to); err != nil {
		return nil, fmt.Errorf("failed to lock output: %v", err)
	}
	outputs = append(outputs, recipientOutput)

	// Change output (if any)
	if accumulated > amount {
		changeOutput := tx.TxOutput{
			Value: accumulated - amount,
		}
		if err := changeOutput.Lock(from); err != nil {
			return nil, fmt.Errorf("failed to lock change output: %v", err)
		}
		outputs = append(outputs, changeOutput)
	}

	// Create transaction
	transaction := tx.NewTransaction(inputs, outputs)

	// Sign transaction
	prevTxs := make(map[string]*tx.Transaction)
	for _, input := range inputs {
		prevTx, err := bc.FindTransaction(input.TxID)
		if err != nil {
			return nil, fmt.Errorf("failed to find previous transaction: %v", err)
		}
		prevTxs[string(input.TxID)] = prevTx
	}

	if err := transaction.Sign(wallet, prevTxs); err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	return transaction, nil
}
