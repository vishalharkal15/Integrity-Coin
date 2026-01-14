package utxo

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/tx"
)

// UTXOSet represents the unspent transaction output set
type UTXOSet struct {
	UTXOs map[string][]tx.TxOutput // Key: txID, Value: outputs
}

// NewUTXOSet creates a new UTXO set
func NewUTXOSet() *UTXOSet {
	return &UTXOSet{
		UTXOs: make(map[string][]tx.TxOutput),
	}
}

// AddUTXO adds a UTXO to the set
func (u *UTXOSet) AddUTXO(txID []byte, outputs []tx.TxOutput) {
	u.UTXOs[string(txID)] = outputs
}

// RemoveUTXO removes a UTXO from the set
func (u *UTXOSet) RemoveUTXO(txID []byte, index int) error {
	key := string(txID)
	outputs, exists := u.UTXOs[key]

	if !exists {
		return fmt.Errorf("UTXO not found")
	}

	if index < 0 || index >= len(outputs) {
		return fmt.Errorf("invalid output index")
	}

	// Remove the output at the specified index
	outputs = append(outputs[:index], outputs[index+1:]...)

	if len(outputs) == 0 {
		delete(u.UTXOs, key)
	} else {
		u.UTXOs[key] = outputs
	}

	return nil
}

// FindUTXO finds a specific UTXO
func (u *UTXOSet) FindUTXO(txID []byte, index int) (*tx.TxOutput, error) {
	outputs, exists := u.UTXOs[string(txID)]

	if !exists {
		return nil, fmt.Errorf("transaction not found")
	}

	if index < 0 || index >= len(outputs) {
		return nil, fmt.Errorf("invalid output index")
	}

	return &outputs[index], nil
}

// FindSpendableOutputs finds spendable outputs for an address
func (u *UTXOSet) FindSpendableOutputs(address string, amount int64) (int64, map[string][]int, error) {
	unspentOutputs := make(map[string][]int)
	accumulated := int64(0)

	pubKeyHash, err := crypto.DecodeAddress(address)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid address: %v", err)
	}

	for txID, outputs := range u.UTXOs {
		for outIdx, out := range outputs {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

				if accumulated >= amount {
					break
				}
			}
		}

		if accumulated >= amount {
			break
		}
	}

	if accumulated < amount {
		return 0, nil, fmt.Errorf("insufficient funds: have %d, need %d", accumulated, amount)
	}

	return accumulated, unspentOutputs, nil
}

// GetBalance calculates the balance for an address
func (u *UTXOSet) GetBalance(address string) (int64, error) {
	balance := int64(0)

	pubKeyHash, err := crypto.DecodeAddress(address)
	if err != nil {
		return 0, fmt.Errorf("invalid address: %v", err)
	}

	for _, outputs := range u.UTXOs {
		for _, out := range outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				balance += out.Value
			}
		}
	}

	return balance, nil
}

// Update updates the UTXO set with a new transaction
func (u *UTXOSet) Update(transaction *tx.Transaction) error {
	// Remove spent outputs (inputs)
	if !transaction.IsCoinbase() {
		for _, input := range transaction.Inputs {
			err := u.RemoveUTXO(input.TxID, input.OutIndex)
			if err != nil {
				return fmt.Errorf("failed to remove UTXO: %v", err)
			}
		}
	}

	// Add new outputs
	u.AddUTXO(transaction.ID, transaction.Outputs)

	return nil
}

// Serialize serializes the UTXO set
func (u *UTXOSet) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(u.UTXOs)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize UTXO set: %v", err)
	}

	return buffer.Bytes(), nil
}

// DeserializeUTXOSet deserializes bytes to UTXO set
func DeserializeUTXOSet(data []byte) (*UTXOSet, error) {
	var utxos map[string][]tx.TxOutput
	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&utxos)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize UTXO set: %v", err)
	}

	return &UTXOSet{UTXOs: utxos}, nil
}

// CountUTXOs returns the total number of UTXOs
func (u *UTXOSet) CountUTXOs() int {
	count := 0
	for _, outputs := range u.UTXOs {
		count += len(outputs)
	}
	return count
}

// GetAllUTXOs returns all UTXOs for an address
func (u *UTXOSet) GetAllUTXOs(address string) ([]tx.TxOutput, error) {
	var utxos []tx.TxOutput

	pubKeyHash, err := crypto.DecodeAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %v", err)
	}

	for _, outputs := range u.UTXOs {
		for _, out := range outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				utxos = append(utxos, out)
			}
		}
	}

	return utxos, nil
}

// Clear clears all UTXOs
func (u *UTXOSet) Clear() {
	u.UTXOs = make(map[string][]tx.TxOutput)
}
