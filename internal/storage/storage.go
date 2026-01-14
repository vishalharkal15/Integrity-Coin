package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/yourusername/bt/internal/tx"
	"github.com/yourusername/bt/pkg/types"
)

const (
	// Database prefixes
	blockPrefix     = "block_"
	heightPrefix    = "height_"
	utxoPrefix      = "utxo_"
	tipKey          = "chain_tip"
	heightKey       = "chain_height"
	difficultyKey   = "difficulty"
)

// Storage represents the LevelDB storage layer
type Storage struct {
	db *leveldb.DB
}

// NewStorage creates a new storage instance
func NewStorage(path string) (*Storage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	return &Storage{db: db}, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveBlock saves a block to the database
func (s *Storage) SaveBlock(block *types.Block) error {
	// Serialize block
	serialized, err := serializeBlock(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %v", err)
	}

	// Save block by hash
	key := []byte(blockPrefix + string(block.Hash))
	if err := s.db.Put(key, serialized, nil); err != nil {
		return fmt.Errorf("failed to save block: %v", err)
	}

	return nil
}

// GetBlock retrieves a block by hash
func (s *Storage) GetBlock(hash []byte) (*types.Block, error) {
	key := []byte(blockPrefix + string(hash))
	data, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("block not found: %v", err)
	}

	block, err := deserializeBlock(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize block: %v", err)
	}

	return block, nil
}

// SaveChainTip saves the current chain tip (latest block hash)
func (s *Storage) SaveChainTip(hash []byte) error {
	return s.db.Put([]byte(tipKey), hash, nil)
}

// GetChainTip retrieves the current chain tip
func (s *Storage) GetChainTip() ([]byte, error) {
	return s.db.Get([]byte(tipKey), nil)
}

// SaveChainHeight saves the current blockchain height
func (s *Storage) SaveChainHeight(height int) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(height); err != nil {
		return err
	}
	return s.db.Put([]byte(heightKey), buf.Bytes(), nil)
}

// GetChainHeight retrieves the current blockchain height
func (s *Storage) GetChainHeight() (int, error) {
	data, err := s.db.Get([]byte(heightKey), nil)
	if err != nil {
		return 0, err
	}

	var height int
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&height); err != nil {
		return 0, err
	}

	return height, nil
}

// SaveDifficulty saves the current difficulty target
func (s *Storage) SaveDifficulty(difficulty uint32) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(difficulty); err != nil {
		return err
	}
	return s.db.Put([]byte(difficultyKey), buf.Bytes(), nil)
}

// GetDifficulty retrieves the current difficulty target
func (s *Storage) GetDifficulty() (uint32, error) {
	data, err := s.db.Get([]byte(difficultyKey), nil)
	if err != nil {
		return 0, err
	}

	var difficulty uint32
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&difficulty); err != nil {
		return 0, err
	}

	return difficulty, nil
}

// SaveUTXO saves a UTXO to the database
func (s *Storage) SaveUTXO(txID []byte, index int, output tx.TxOutput) error {
	key := []byte(fmt.Sprintf("%s%x_%d", utxoPrefix, txID, index))
	
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode UTXO: %v", err)
	}

	return s.db.Put(key, buf.Bytes(), nil)
}

// DeleteUTXO removes a UTXO from the database
func (s *Storage) DeleteUTXO(txID []byte, index int) error {
	key := []byte(fmt.Sprintf("%s%x_%d", utxoPrefix, txID, index))
	return s.db.Delete(key, nil)
}

// GetUTXO retrieves a UTXO from the database
func (s *Storage) GetUTXO(txID []byte, index int) (*tx.TxOutput, error) {
	key := []byte(fmt.Sprintf("%s%x_%d", utxoPrefix, txID, index))
	data, err := s.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	var output tx.TxOutput
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&output); err != nil {
		return nil, err
	}

	return &output, nil
}

// GetAllUTXOs retrieves all UTXOs from the database
func (s *Storage) GetAllUTXOs() (map[string][]tx.TxOutput, error) {
	utxos := make(map[string][]tx.TxOutput)

	iter := s.db.NewIterator(util.BytesPrefix([]byte(utxoPrefix)), nil)
	defer iter.Release()

	for iter.Next() {
		key := string(iter.Key())
		// Parse key to extract txID
		// Format: "utxo_<txID>_<index>"
		
		var output tx.TxOutput
		decoder := gob.NewDecoder(bytes.NewReader(iter.Value()))
		if err := decoder.Decode(&output); err != nil {
			continue
		}

		// Extract txID from key (simplified, would need proper parsing)
		utxos[key] = append(utxos[key], output)
	}

	return utxos, iter.Error()
}

// BlockExists checks if a block exists in the database
func (s *Storage) BlockExists(hash []byte) bool {
	key := []byte(blockPrefix + string(hash))
	exists, _ := s.db.Has(key, nil)
	return exists
}

// DeleteBlock removes a block from the database
func (s *Storage) DeleteBlock(hash []byte) error {
	key := []byte(blockPrefix + string(hash))
	return s.db.Delete(key, nil)
}

// GetAllBlocks retrieves all blocks from the database
func (s *Storage) GetAllBlocks() ([]*types.Block, error) {
	var blocks []*types.Block

	iter := s.db.NewIterator(util.BytesPrefix([]byte(blockPrefix)), nil)
	defer iter.Release()

	for iter.Next() {
		block, err := deserializeBlock(iter.Value())
		if err != nil {
			continue
		}
		blocks = append(blocks, block)
	}

	return blocks, iter.Error()
}

// serializeBlock serializes a block to bytes
func serializeBlock(block *types.Block) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	// Create a serializable structure
	data := struct {
		Header       types.BlockHeader
		Transactions []*tx.Transaction
		Hash         []byte
	}{
		Header: block.Header,
		Hash:   block.Hash,
	}

	// Cast transactions
	if txs, ok := block.Transactions.([]*tx.Transaction); ok {
		data.Transactions = txs
	}

	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// deserializeBlock deserializes bytes to a block
func deserializeBlock(data []byte) (*types.Block, error) {
	var blockData struct {
		Header       types.BlockHeader
		Transactions []*tx.Transaction
		Hash         []byte
	}

	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&blockData); err != nil {
		return nil, err
	}

	block := &types.Block{
		Header:       blockData.Header,
		Transactions: blockData.Transactions,
		Hash:         blockData.Hash,
	}

	return block, nil
}

// Clear removes all data from the database
func (s *Storage) Clear() error {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)
	for iter.Next() {
		batch.Delete(iter.Key())
	}

	return s.db.Write(batch, nil)
}
