package tx

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/yourusername/bt/internal/crypto"
)

// Transaction represents a cryptocurrency transaction
type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

// TxInput represents a transaction input (reference to previous output)
type TxInput struct {
	TxID      []byte // Previous transaction ID
	OutIndex  int    // Index of the output in previous transaction
	Signature []byte // Signature proving ownership
	PubKey    []byte // Public key of the sender
}

// TxOutput represents a transaction output (new UTXO)
type TxOutput struct {
	Value      int64  // Amount in satoshis
	PubKeyHash []byte // Hash of recipient's public key
}

// NewTransaction creates a new transaction
func NewTransaction(inputs []TxInput, outputs []TxOutput) *Transaction {
	tx := &Transaction{
		Inputs:  inputs,
		Outputs: outputs,
	}

	tx.ID = tx.Hash()
	return tx
}

// NewCoinbaseTx creates a coinbase transaction (mining reward)
func NewCoinbaseTx(to string, data string, reward int64) (*Transaction, error) {
	if data == "" {
		data = fmt.Sprintf("Reward to %s", to)
	}

	// Coinbase has no inputs (mined from nothing)
	txin := TxInput{
		TxID:      nil,
		OutIndex:  -1,
		Signature: nil,
		PubKey:    []byte(data),
	}

	// Decode recipient address to get pub key hash
	pubKeyHash, err := crypto.DecodeAddress(to)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %v", err)
	}

	txout := TxOutput{
		Value:      reward,
		PubKeyHash: pubKeyHash,
	}

	tx := &Transaction{
		Inputs:  []TxInput{txin},
		Outputs: []TxOutput{txout},
	}

	tx.ID = tx.Hash()

	return tx, nil
}

// IsCoinbase checks if the transaction is a coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && tx.Inputs[0].TxID == nil && tx.Inputs[0].OutIndex == -1
}

// Hash calculates the hash of the transaction
func (tx *Transaction) Hash() []byte {
	txCopy := *tx
	txCopy.ID = nil

	serialized, err := txCopy.Serialize()
	if err != nil {
		return nil
	}

	hash := crypto.DoubleHashBytes(serialized)
	return hash
}

// Serialize serializes the transaction to bytes
func (tx *Transaction) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %v", err)
	}

	return buffer.Bytes(), nil
}

// DeserializeTransaction deserializes bytes to a transaction
func DeserializeTransaction(data []byte) (*Transaction, error) {
	var tx Transaction
	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&tx)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	return &tx, nil
}

// Sign signs the transaction with the provided private key
// Only signs non-coinbase inputs
func (tx *Transaction) Sign(wallet *crypto.Wallet, prevTxs map[string]*Transaction) error {
	if tx.IsCoinbase() {
		return nil // Coinbase transactions don't need signing
	}

	// Verify all referenced transactions exist
	for _, input := range tx.Inputs {
		if prevTxs[string(input.TxID)] == nil {
			return fmt.Errorf("previous transaction not found")
		}
	}

	// Create a trimmed copy for signing (without signatures and pubkeys)
	txCopy := tx.TrimmedCopy()

	// Sign each input
	for i, input := range txCopy.Inputs {
		prevTx := prevTxs[string(input.TxID)]
		txCopy.Inputs[i].Signature = nil
		txCopy.Inputs[i].PubKey = prevTx.Outputs[input.OutIndex].PubKeyHash

		// Create data to sign (transaction hash)
		dataToSign := txCopy.Hash()

		// Sign the data
		signature, err := wallet.Sign(dataToSign)
		if err != nil {
			return fmt.Errorf("failed to sign input %d: %v", i, err)
		}

		// Store signature in original transaction
		tx.Inputs[i].Signature = signature

		// Clear for next iteration
		txCopy.Inputs[i].PubKey = nil
	}

	return nil
}

// Verify verifies signatures in the transaction
func (tx *Transaction) Verify(prevTxs map[string]*Transaction) bool {
	if tx.IsCoinbase() {
		return true // Coinbase transactions don't need verification
	}

	// Verify all referenced transactions exist
	for _, input := range tx.Inputs {
		if prevTxs[string(input.TxID)] == nil {
			return false
		}
	}

	// Create a trimmed copy for verification
	txCopy := tx.TrimmedCopy()

	// Verify each input
	for i, input := range tx.Inputs {
		prevTx := prevTxs[string(input.TxID)]
		txCopy.Inputs[i].Signature = nil
		txCopy.Inputs[i].PubKey = prevTx.Outputs[input.OutIndex].PubKeyHash

		// Get the data that was signed
		dataToVerify := txCopy.Hash()

		// Verify signature
		if !crypto.VerifySignature(input.PubKey, dataToVerify, input.Signature) {
			return false
		}

		// Clear for next iteration
		txCopy.Inputs[i].PubKey = nil
	}

	return true
}

// TrimmedCopy creates a copy of the transaction without signatures and pubkeys
func (tx *Transaction) TrimmedCopy() *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, input := range tx.Inputs {
		inputs = append(inputs, TxInput{
			TxID:      input.TxID,
			OutIndex:  input.OutIndex,
			Signature: nil,
			PubKey:    nil,
		})
	}

	for _, output := range tx.Outputs {
		outputs = append(outputs, TxOutput{
			Value:      output.Value,
			PubKeyHash: output.PubKeyHash,
		})
	}

	return &Transaction{
		ID:      tx.ID,
		Inputs:  inputs,
		Outputs: outputs,
	}
}

// UsesKey checks if the input uses a specific public key hash
func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := crypto.PublicKeyHash(in.PubKey)
	return bytes.Equal(lockingHash, pubKeyHash)
}

// IsLockedWithKey checks if the output is locked with a specific public key hash
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

// Lock locks the output with a public key hash (for new outputs)
func (out *TxOutput) Lock(address string) error {
	pubKeyHash, err := crypto.DecodeAddress(address)
	if err != nil {
		return err
	}
	out.PubKeyHash = pubKeyHash
	return nil
}
