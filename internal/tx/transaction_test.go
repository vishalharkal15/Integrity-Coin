package tx

import (
	"testing"

	"github.com/yourusername/bt/internal/crypto"
)

func TestNewTransaction(t *testing.T) {
	inputs := []TxInput{
		{
			TxID:     []byte("test"),
			OutIndex: 0,
		},
	}

	outputs := []TxOutput{
		{
			Value:      100,
			PubKeyHash: []byte("recipient"),
		},
	}

	tx := NewTransaction(inputs, outputs)

	if tx == nil {
		t.Fatal("NewTransaction returned nil")
	}

	if len(tx.ID) == 0 {
		t.Error("Transaction ID is empty")
	}

	if len(tx.Inputs) != 1 {
		t.Errorf("Expected 1 input, got %d", len(tx.Inputs))
	}

	if len(tx.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(tx.Outputs))
	}
}

func TestNewCoinbaseTx(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	address := wallet.GetAddress()

	tx, err := NewCoinbaseTx(address, "Test coinbase", 50*1e8)
	if err != nil {
		t.Fatalf("Failed to create coinbase tx: %v", err)
	}

	if !tx.IsCoinbase() {
		t.Error("Transaction is not coinbase")
	}

	if len(tx.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(tx.Outputs))
	}

	if tx.Outputs[0].Value != 50*1e8 {
		t.Errorf("Expected value 50*1e8, got %d", tx.Outputs[0].Value)
	}
}

func TestIsCoinbase(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	address := wallet.GetAddress()

	coinbaseTx, _ := NewCoinbaseTx(address, "Coinbase", 50*1e8)
	if !coinbaseTx.IsCoinbase() {
		t.Error("Coinbase transaction not identified correctly")
	}

	regularTx := NewTransaction(
		[]TxInput{{TxID: []byte("test"), OutIndex: 0}},
		[]TxOutput{{Value: 100, PubKeyHash: []byte("test")}},
	)
	if regularTx.IsCoinbase() {
		t.Error("Regular transaction identified as coinbase")
	}
}

func TestTransactionHash(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: []byte("test"), OutIndex: 0}},
		[]TxOutput{{Value: 100, PubKeyHash: []byte("test")}},
	)

	hash1 := tx.Hash()
	hash2 := tx.Hash()

	if string(hash1) != string(hash2) {
		t.Error("Transaction hash is not deterministic")
	}

	if len(hash1) != 32 {
		t.Errorf("Hash length = %d, want 32", len(hash1))
	}
}

func TestSerializeDeserialize(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: []byte("test"), OutIndex: 0}},
		[]TxOutput{{Value: 100, PubKeyHash: []byte("test")}},
	)

	serialized, err := tx.Serialize()
	if err != nil {
		t.Fatalf("Serialization failed: %v", err)
	}

	deserialized, err := DeserializeTransaction(serialized)
	if err != nil {
		t.Fatalf("Deserialization failed: %v", err)
	}

	if string(tx.ID) != string(deserialized.ID) {
		t.Error("Deserialized transaction ID mismatch")
	}

	if len(tx.Inputs) != len(deserialized.Inputs) {
		t.Error("Deserialized inputs count mismatch")
	}

	if len(tx.Outputs) != len(deserialized.Outputs) {
		t.Error("Deserialized outputs count mismatch")
	}
}

func TestSignAndVerify(t *testing.T) {
	// Create two wallets
	wallet1, _ := crypto.NewWallet()
	wallet2, _ := crypto.NewWallet()

	// Create a "previous" transaction (wallet1 received coins)
	prevTx, _ := NewCoinbaseTx(wallet1.GetAddress(), "Prev TX", 100*1e8)

	// Create a transaction sending coins from wallet1 to wallet2
	input := TxInput{
		TxID:      prevTx.ID,
		OutIndex:  0,
		Signature: nil,
		PubKey:    wallet1.PublicKey,
	}

	output := TxOutput{
		Value: 50 * 1e8,
	}
	output.Lock(wallet2.GetAddress())

	tx := NewTransaction([]TxInput{input}, []TxOutput{output})

	// Sign the transaction
	prevTxs := map[string]*Transaction{
		string(prevTx.ID): prevTx,
	}

	err := tx.Sign(wallet1, prevTxs)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify the transaction
	if !tx.Verify(prevTxs) {
		t.Error("Transaction verification failed")
	}
}

func TestVerifyInvalidSignature(t *testing.T) {
	wallet1, _ := crypto.NewWallet()
	wallet2, _ := crypto.NewWallet()
	wallet3, _ := crypto.NewWallet()

	prevTx, _ := NewCoinbaseTx(wallet1.GetAddress(), "Prev", 100*1e8)

	input := TxInput{
		TxID:      prevTx.ID,
		OutIndex:  0,
		Signature: nil,
		PubKey:    wallet1.PublicKey,
	}

	output := TxOutput{Value: 50 * 1e8}
	output.Lock(wallet2.GetAddress())

	tx := NewTransaction([]TxInput{input}, []TxOutput{output})

	prevTxs := map[string]*Transaction{
		string(prevTx.ID): prevTx,
	}

	tx.Sign(wallet1, prevTxs)

	// Try to verify with wrong wallet
	tx.Inputs[0].PubKey = wallet3.PublicKey
	if tx.Verify(prevTxs) {
		t.Error("Invalid transaction passed verification")
	}
}

func TestTrimmedCopy(t *testing.T) {
	wallet, _ := crypto.NewWallet()

	input := TxInput{
		TxID:      []byte("test"),
		OutIndex:  0,
		Signature: []byte("signature"),
		PubKey:    wallet.PublicKey,
	}

	output := TxOutput{
		Value:      100,
		PubKeyHash: []byte("hash"),
	}

	tx := NewTransaction([]TxInput{input}, []TxOutput{output})
	trimmed := tx.TrimmedCopy()

	if trimmed.Inputs[0].Signature != nil {
		t.Error("Trimmed copy contains signature")
	}

	if trimmed.Inputs[0].PubKey != nil {
		t.Error("Trimmed copy contains public key")
	}

	if trimmed.Inputs[0].OutIndex != input.OutIndex {
		t.Error("Trimmed copy modified OutIndex")
	}
}

func TestUsesKey(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	pubKeyHash := crypto.PublicKeyHash(wallet.PublicKey)

	input := TxInput{
		PubKey: wallet.PublicKey,
	}

	if !input.UsesKey(pubKeyHash) {
		t.Error("UsesKey returned false for correct key")
	}

	wrongHash := []byte("wrong hash")
	if input.UsesKey(wrongHash) {
		t.Error("UsesKey returned true for wrong key")
	}
}

func TestIsLockedWithKey(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	address := wallet.GetAddress()

	output := TxOutput{}
	output.Lock(address)

	pubKeyHash, _ := crypto.DecodeAddress(address)

	if !output.IsLockedWithKey(pubKeyHash) {
		t.Error("IsLockedWithKey returned false for correct key")
	}

	wrongHash := []byte("wrong hash")
	if output.IsLockedWithKey(wrongHash) {
		t.Error("IsLockedWithKey returned true for wrong key")
	}
}

func BenchmarkNewTransaction(b *testing.B) {
	inputs := []TxInput{{TxID: []byte("test"), OutIndex: 0}}
	outputs := []TxOutput{{Value: 100, PubKeyHash: []byte("test")}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewTransaction(inputs, outputs)
	}
}

func BenchmarkSerialize(b *testing.B) {
	tx := NewTransaction(
		[]TxInput{{TxID: []byte("test"), OutIndex: 0}},
		[]TxOutput{{Value: 100, PubKeyHash: []byte("test")}},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx.Serialize()
	}
}
