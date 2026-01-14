package crypto

import (
	"testing"
)

func TestNewWallet(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	if wallet.PrivateKey == nil {
		t.Error("Private key is nil")
	}

	if len(wallet.PublicKey) == 0 {
		t.Error("Public key is empty")
	}

	// Public key should be 65 bytes (uncompressed)
	if len(wallet.PublicKey) != 65 {
		t.Errorf("Public key length = %d, want 65", len(wallet.PublicKey))
	}
}

func TestGetAddress(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	address := wallet.GetAddress()

	if address == "" {
		t.Error("Address is empty")
	}

	// Address should be Base58 encoded
	if len(address) < 26 || len(address) > 35 {
		t.Errorf("Address length = %d, expected 26-35", len(address))
	}
}

func TestAddressEncodeDecode(t *testing.T) {
	wallet, _ := NewWallet()
	address := wallet.GetAddress()

	// Decode the address
	decoded, err := DecodeAddress(address)
	if err != nil {
		t.Fatalf("Failed to decode address: %v", err)
	}

	// Encode it again
	reEncoded := EncodeAddress(decoded)

	if address != reEncoded {
		t.Error("Address encoding/decoding is not reversible")
	}
}

func TestSignAndVerify(t *testing.T) {
	wallet, _ := NewWallet()
	message := []byte("test message")

	// Sign the message
	signature, err := wallet.Sign(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	if len(signature) == 0 {
		t.Error("Signature is empty")
	}

	// Verify the signature
	if !VerifySignature(wallet.PublicKey, message, signature) {
		t.Error("Signature verification failed")
	}
}

func TestVerifyInvalidSignature(t *testing.T) {
	wallet, _ := NewWallet()
	message := []byte("test message")
	wrongMessage := []byte("wrong message")

	signature, _ := wallet.Sign(message)

	// Verify with wrong message
	if VerifySignature(wallet.PublicKey, wrongMessage, signature) {
		t.Error("Invalid signature passed verification")
	}

	// Verify with wrong public key
	otherWallet, _ := NewWallet()
	if VerifySignature(otherWallet.PublicKey, message, signature) {
		t.Error("Signature verified with wrong public key")
	}

	// Verify with tampered signature
	tamperedSig := make([]byte, len(signature))
	copy(tamperedSig, signature)
	if len(tamperedSig) > 0 {
		tamperedSig[0] ^= 0xFF
	}
	if VerifySignature(wallet.PublicKey, message, tamperedSig) {
		t.Error("Tampered signature passed verification")
	}
}

func TestPublicKeyHash(t *testing.T) {
	wallet, _ := NewWallet()
	
	hash := PublicKeyHash(wallet.PublicKey)

	// RIPEMD-160 produces 20-byte hash
	if len(hash) != 20 {
		t.Errorf("Public key hash length = %d, want 20", len(hash))
	}

	// Same public key should produce same hash
	hash2 := PublicKeyHash(wallet.PublicKey)
	if string(hash) != string(hash2) {
		t.Error("Public key hash is not deterministic")
	}
}

func TestPrivateKeyHexConversion(t *testing.T) {
	wallet, _ := NewWallet()

	hexKey := PrivateKeyToHex(wallet.PrivateKey)
	if hexKey == "" {
		t.Error("Hex key is empty")
	}

	// Convert back
	privateKey, err := HexToPrivateKey(hexKey)
	if err != nil {
		t.Fatalf("Failed to convert hex to private key: %v", err)
	}

	// Should produce same public key
	pubKey1 := PublicKeyBytes(wallet.PrivateKey)
	pubKey2 := PublicKeyBytes(privateKey)

	if string(pubKey1) != string(pubKey2) {
		t.Error("Private key conversion is not reversible")
	}
}

func TestUniqueAddresses(t *testing.T) {
	wallet1, _ := NewWallet()
	wallet2, _ := NewWallet()

	addr1 := wallet1.GetAddress()
	addr2 := wallet2.GetAddress()

	if addr1 == addr2 {
		t.Error("Different wallets produced same address")
	}
}

func TestChecksum(t *testing.T) {
	data := []byte("test data")
	checksum1 := Checksum(data)

	if len(checksum1) != ChecksumLength {
		t.Errorf("Checksum length = %d, want %d", len(checksum1), ChecksumLength)
	}

	// Same data should produce same checksum
	checksum2 := Checksum(data)
	if string(checksum1) != string(checksum2) {
		t.Error("Checksum is not deterministic")
	}

	// Different data should produce different checksum
	data2 := []byte("different data")
	checksum3 := Checksum(data2)
	if string(checksum1) == string(checksum3) {
		t.Error("Different data produced same checksum")
	}
}

func BenchmarkNewWallet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewWallet()
	}
}

func BenchmarkSign(b *testing.B) {
	wallet, _ := NewWallet()
	message := []byte("benchmark message")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wallet.Sign(message)
	}
}

func BenchmarkVerifySignature(b *testing.B) {
	wallet, _ := NewWallet()
	message := []byte("benchmark message")
	signature, _ := wallet.Sign(message)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		VerifySignature(wallet.PublicKey, message, signature)
	}
}
