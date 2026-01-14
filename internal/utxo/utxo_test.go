package utxo

import (
	"testing"

	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/tx"
)

func TestNewUTXOSet(t *testing.T) {
	utxoSet := NewUTXOSet()

	if utxoSet == nil {
		t.Fatal("NewUTXOSet returned nil")
	}

	if utxoSet.UTXOs == nil {
		t.Error("UTXOs map is nil")
	}

	if len(utxoSet.UTXOs) != 0 {
		t.Error("New UTXO set should be empty")
	}
}

func TestAddUTXO(t *testing.T) {
	utxoSet := NewUTXOSet()
	txID := []byte("test-tx")
	outputs := []tx.TxOutput{
		{Value: 100, PubKeyHash: []byte("hash1")},
		{Value: 200, PubKeyHash: []byte("hash2")},
	}

	utxoSet.AddUTXO(txID, outputs)

	if len(utxoSet.UTXOs) != 1 {
		t.Errorf("Expected 1 UTXO entry, got %d", len(utxoSet.UTXOs))
	}

	stored := utxoSet.UTXOs[string(txID)]
	if len(stored) != 2 {
		t.Errorf("Expected 2 outputs, got %d", len(stored))
	}
}

func TestFindUTXO(t *testing.T) {
	utxoSet := NewUTXOSet()
	txID := []byte("test-tx")
	outputs := []tx.TxOutput{
		{Value: 100, PubKeyHash: []byte("hash1")},
		{Value: 200, PubKeyHash: []byte("hash2")},
	}

	utxoSet.AddUTXO(txID, outputs)

	// Find existing UTXO
	output, err := utxoSet.FindUTXO(txID, 0)
	if err != nil {
		t.Fatalf("Failed to find UTXO: %v", err)
	}

	if output.Value != 100 {
		t.Errorf("Expected value 100, got %d", output.Value)
	}

	// Try to find non-existent UTXO
	_, err = utxoSet.FindUTXO([]byte("nonexistent"), 0)
	if err == nil {
		t.Error("Expected error for non-existent UTXO")
	}

	// Try invalid index
	_, err = utxoSet.FindUTXO(txID, 10)
	if err == nil {
		t.Error("Expected error for invalid index")
	}
}

func TestRemoveUTXO(t *testing.T) {
	utxoSet := NewUTXOSet()
	txID := []byte("test-tx")
	outputs := []tx.TxOutput{
		{Value: 100, PubKeyHash: []byte("hash1")},
		{Value: 200, PubKeyHash: []byte("hash2")},
	}

	utxoSet.AddUTXO(txID, outputs)

	// Remove first output
	err := utxoSet.RemoveUTXO(txID, 0)
	if err != nil {
		t.Fatalf("Failed to remove UTXO: %v", err)
	}

	// Should still have one output
	remaining := utxoSet.UTXOs[string(txID)]
	if len(remaining) != 1 {
		t.Errorf("Expected 1 remaining output, got %d", len(remaining))
	}

	// Remove last output
	err = utxoSet.RemoveUTXO(txID, 0)
	if err != nil {
		t.Fatalf("Failed to remove last UTXO: %v", err)
	}

	// Transaction should be removed from map
	if _, exists := utxoSet.UTXOs[string(txID)]; exists {
		t.Error("Transaction entry should be removed when all outputs spent")
	}
}

func TestGetBalance(t *testing.T) {
	utxoSet := NewUTXOSet()
	wallet, _ := crypto.NewWallet()
	address := wallet.GetAddress()
	pubKeyHash, _ := crypto.DecodeAddress(address)

	// Add some UTXOs for the wallet
	outputs := []tx.TxOutput{
		{Value: 100, PubKeyHash: pubKeyHash},
		{Value: 200, PubKeyHash: pubKeyHash},
	}

	utxoSet.AddUTXO([]byte("tx1"), outputs)

	// Add UTXO for another address
	otherOutputs := []tx.TxOutput{
		{Value: 500, PubKeyHash: []byte("other")},
	}
	utxoSet.AddUTXO([]byte("tx2"), otherOutputs)

	balance, err := utxoSet.GetBalance(address)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	if balance != 300 {
		t.Errorf("Expected balance 300, got %d", balance)
	}
}

func TestFindSpendableOutputs(t *testing.T) {
	utxoSet := NewUTXOSet()
	wallet, _ := crypto.NewWallet()
	address := wallet.GetAddress()
	pubKeyHash, _ := crypto.DecodeAddress(address)

	// Add UTXOs
	utxoSet.AddUTXO([]byte("tx1"), []tx.TxOutput{
		{Value: 100, PubKeyHash: pubKeyHash},
	})
	utxoSet.AddUTXO([]byte("tx2"), []tx.TxOutput{
		{Value: 200, PubKeyHash: pubKeyHash},
	})
	utxoSet.AddUTXO([]byte("tx3"), []tx.TxOutput{
		{Value: 300, PubKeyHash: pubKeyHash},
	})

	// Find spendable outputs for amount 250
	accumulated, outputs, err := utxoSet.FindSpendableOutputs(address, 250)
	if err != nil {
		t.Fatalf("Failed to find spendable outputs: %v", err)
	}

	if accumulated < 250 {
		t.Errorf("Accumulated %d < required 250", accumulated)
	}

	if len(outputs) == 0 {
		t.Error("No outputs found")
	}

	// Try to spend more than available
	_, _, err = utxoSet.FindSpendableOutputs(address, 10000)
	if err == nil {
		t.Error("Expected error for insufficient funds")
	}
}

func TestUpdate(t *testing.T) {
	utxoSet := NewUTXOSet()
	wallet1, _ := crypto.NewWallet()
	wallet2, _ := crypto.NewWallet()

	// Create initial UTXO
	prevTx, _ := tx.NewCoinbaseTx(wallet1.GetAddress(), "Initial", 100*1e8)
	utxoSet.AddUTXO(prevTx.ID, prevTx.Outputs)

	// Create a transaction spending the UTXO
	pubKeyHash2, _ := crypto.DecodeAddress(wallet2.GetAddress())
	newTx := tx.NewTransaction(
		[]tx.TxInput{{TxID: prevTx.ID, OutIndex: 0}},
		[]tx.TxOutput{{Value: 50 * 1e8, PubKeyHash: pubKeyHash2}},
	)

	// Update UTXO set
	err := utxoSet.Update(newTx)
	if err != nil {
		t.Fatalf("Failed to update UTXO set: %v", err)
	}

	// Old UTXO should be removed
	_, err = utxoSet.FindUTXO(prevTx.ID, 0)
	if err == nil {
		t.Error("Spent UTXO still exists")
	}

	// New UTXO should exist
	_, err = utxoSet.FindUTXO(newTx.ID, 0)
	if err != nil {
		t.Error("New UTXO not found")
	}
}

func TestSerializeDeserialize(t *testing.T) {
	utxoSet := NewUTXOSet()

	outputs := []tx.TxOutput{
		{Value: 100, PubKeyHash: []byte("hash1")},
		{Value: 200, PubKeyHash: []byte("hash2")},
	}
	utxoSet.AddUTXO([]byte("tx1"), outputs)

	// Serialize
	serialized, err := utxoSet.Serialize()
	if err != nil {
		t.Fatalf("Serialization failed: %v", err)
	}

	// Deserialize
	deserialized, err := DeserializeUTXOSet(serialized)
	if err != nil {
		t.Fatalf("Deserialization failed: %v", err)
	}

	if len(deserialized.UTXOs) != len(utxoSet.UTXOs) {
		t.Error("Deserialized UTXO set size mismatch")
	}
}

func TestCountUTXOs(t *testing.T) {
	utxoSet := NewUTXOSet()

	utxoSet.AddUTXO([]byte("tx1"), []tx.TxOutput{
		{Value: 100, PubKeyHash: []byte("hash")},
		{Value: 200, PubKeyHash: []byte("hash")},
	})

	utxoSet.AddUTXO([]byte("tx2"), []tx.TxOutput{
		{Value: 300, PubKeyHash: []byte("hash")},
	})

	count := utxoSet.CountUTXOs()
	if count != 3 {
		t.Errorf("Expected 3 UTXOs, got %d", count)
	}
}

func TestGetAllUTXOs(t *testing.T) {
	utxoSet := NewUTXOSet()
	wallet, _ := crypto.NewWallet()
	address := wallet.GetAddress()
	pubKeyHash, _ := crypto.DecodeAddress(address)

	utxoSet.AddUTXO([]byte("tx1"), []tx.TxOutput{
		{Value: 100, PubKeyHash: pubKeyHash},
		{Value: 200, PubKeyHash: pubKeyHash},
	})

	utxoSet.AddUTXO([]byte("tx2"), []tx.TxOutput{
		{Value: 300, PubKeyHash: []byte("other")},
	})

	utxos, err := utxoSet.GetAllUTXOs(address)
	if err != nil {
		t.Fatalf("Failed to get all UTXOs: %v", err)
	}

	if len(utxos) != 2 {
		t.Errorf("Expected 2 UTXOs, got %d", len(utxos))
	}
}

func TestClear(t *testing.T) {
	utxoSet := NewUTXOSet()

	utxoSet.AddUTXO([]byte("tx1"), []tx.TxOutput{{Value: 100}})
	utxoSet.AddUTXO([]byte("tx2"), []tx.TxOutput{{Value: 200}})

	utxoSet.Clear()

	if len(utxoSet.UTXOs) != 0 {
		t.Error("UTXO set not cleared")
	}
}

func BenchmarkGetBalance(b *testing.B) {
	utxoSet := NewUTXOSet()
	wallet, _ := crypto.NewWallet()
	address := wallet.GetAddress()
	pubKeyHash, _ := crypto.DecodeAddress(address)

	// Add many UTXOs
	for i := 0; i < 100; i++ {
		outputs := []tx.TxOutput{
			{Value: int64(i * 100), PubKeyHash: pubKeyHash},
		}
		utxoSet.AddUTXO([]byte{byte(i)}, outputs)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utxoSet.GetBalance(address)
	}
}
