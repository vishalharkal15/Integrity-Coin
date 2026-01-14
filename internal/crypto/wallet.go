package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/ripemd160"
)

const (
	// Version for address generation
	AddressVersion = 0x00

	// ChecksumLength is the length of address checksum
	ChecksumLength = 4
)

// Wallet represents a cryptocurrency wallet with a key pair
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

// NewWallet creates a new wallet with a generated key pair
func NewWallet() (*Wallet, error) {
	privateKey, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	publicKey := PublicKeyBytes(privateKey)

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// GenerateKeyPair generates a new ECDSA secp256k1 key pair
func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	// Use btcsuite's secp256k1 curve (same as Bitcoin)
	curve := btcec.S256()

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	return privateKey, nil
}

// PublicKeyBytes converts an ECDSA public key to bytes (uncompressed format)
func PublicKeyBytes(privateKey *ecdsa.PrivateKey) []byte {
	pubKey := privateKey.PublicKey
	// Uncompressed format: 0x04 + X (32 bytes) + Y (32 bytes)
	return elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
}

// GetAddress generates a Bitcoin-like address from a public key
// Address = Base58(version + RIPEMD160(SHA256(pubKey)) + checksum)
func (w *Wallet) GetAddress() string {
	return GetAddressFromPubKey(w.PublicKey)
}

// GetAddressFromPubKey generates an address from a public key
func GetAddressFromPubKey(pubKey []byte) string {
	// 1. SHA-256 hash of public key
	sha256Hash := sha256.Sum256(pubKey)

	// 2. RIPEMD-160 hash of SHA-256 hash
	ripemd160Hasher := ripemd160.New()
	ripemd160Hasher.Write(sha256Hash[:])
	publicKeyHash := ripemd160Hasher.Sum(nil)

	return EncodeAddress(publicKeyHash)
}

// EncodeAddress encodes a public key hash into a Bitcoin-like address
func EncodeAddress(pubKeyHash []byte) string {
	// Add version byte
	versionedPayload := append([]byte{AddressVersion}, pubKeyHash...)

	// Calculate checksum (first 4 bytes of double SHA-256)
	checksum := Checksum(versionedPayload)

	// Concatenate version + payload + checksum
	fullPayload := append(versionedPayload, checksum...)

	// Base58 encode
	return base58.Encode(fullPayload)
}

// DecodeAddress decodes a Bitcoin-like address to public key hash
func DecodeAddress(address string) ([]byte, error) {
	decoded, err := base58.Decode(address)
	if err != nil {
		return nil, fmt.Errorf("failed to decode address: %v", err)
	}

	if len(decoded) < ChecksumLength+1 {
		return nil, fmt.Errorf("invalid address length")
	}

	// Split payload and checksum
	payload := decoded[:len(decoded)-ChecksumLength]
	checksumProvided := decoded[len(decoded)-ChecksumLength:]

	// Verify checksum
	checksumCalculated := Checksum(payload)
	if string(checksumCalculated) != string(checksumProvided) {
		return nil, fmt.Errorf("invalid address checksum")
	}

	// Remove version byte
	pubKeyHash := payload[1:]

	return pubKeyHash, nil
}

// Checksum generates a 4-byte checksum for address encoding
func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:ChecksumLength]
}

// Sign signs a message with the wallet's private key
func (w *Wallet) Sign(message []byte) ([]byte, error) {
	hash := sha256.Sum256(message)

	r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %v", err)
	}

	// Combine r and s into signature (r || s)
	signature := append(r.Bytes(), s.Bytes()...)

	return signature, nil
}

// VerifySignature verifies a signature against a message and public key
func VerifySignature(pubKey, message, signature []byte) bool {
	hash := sha256.Sum256(message)

	// Parse public key
	curve := btcec.S256()
	x, y := elliptic.Unmarshal(curve, pubKey)
	if x == nil {
		return false
	}

	publicKey := ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	// Split signature into r and s
	if len(signature) < 32 {
		return false
	}

	r := big.Int{}
	s := big.Int{}
	r.SetBytes(signature[:len(signature)/2])
	s.SetBytes(signature[len(signature)/2:])

	// Verify
	return ecdsa.Verify(&publicKey, hash[:], &r, &s)
}

// PrivateKeyToHex converts a private key to hex string
func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) string {
	return hex.EncodeToString(privateKey.D.Bytes())
}

// HexToPrivateKey converts a hex string to private key
func HexToPrivateKey(hexKey string) (*ecdsa.PrivateKey, error) {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}

	curve := btcec.S256()
	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = curve
	privateKey.D = new(big.Int).SetBytes(bytes)
	privateKey.PublicKey.X, privateKey.PublicKey.Y = curve.ScalarBaseMult(bytes)

	return privateKey, nil
}

// PublicKeyHash returns the RIPEMD160(SHA256(pubKey))
func PublicKeyHash(pubKey []byte) []byte {
	sha256Hash := sha256.Sum256(pubKey)
	ripemd160Hasher := ripemd160.New()
	ripemd160Hasher.Write(sha256Hash[:])
	return ripemd160Hasher.Sum(nil)
}
