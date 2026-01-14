package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	walletPrefix = "wallet_"
	addressKey   = "addresses"
)

// WalletStorage manages wallet persistence
type WalletStorage struct {
	db *leveldb.DB
}

// WalletData represents serializable wallet information
type WalletData struct {
	Address    string
	PrivateKey []byte
	PublicKey  []byte
}

// NewWalletStorage creates a new wallet storage instance
func NewWalletStorage(path string) (*WalletStorage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open wallet database: %v", err)
	}

	return &WalletStorage{db: db}, nil
}

// Close closes the wallet database
func (ws *WalletStorage) Close() error {
	return ws.db.Close()
}

// SaveWallet saves a wallet to the database
func (ws *WalletStorage) SaveWallet(address string, privateKey, publicKey []byte) error {
	walletData := WalletData{
		Address:    address,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(walletData); err != nil {
		return fmt.Errorf("failed to encode wallet: %v", err)
	}

	key := []byte(walletPrefix + address)
	if err := ws.db.Put(key, buf.Bytes(), nil); err != nil {
		return fmt.Errorf("failed to save wallet: %v", err)
	}

	// Add address to the list
	return ws.addAddress(address)
}

// GetWallet retrieves a wallet by address
func (ws *WalletStorage) GetWallet(address string) (*WalletData, error) {
	key := []byte(walletPrefix + address)
	data, err := ws.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %v", err)
	}

	var walletData WalletData
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&walletData); err != nil {
		return nil, fmt.Errorf("failed to decode wallet: %v", err)
	}

	return &walletData, nil
}

// GetAllAddresses returns all wallet addresses
func (ws *WalletStorage) GetAllAddresses() ([]string, error) {
	data, err := ws.db.Get([]byte(addressKey), nil)
	if err == leveldb.ErrNotFound {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var addresses []string
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&addresses); err != nil {
		return nil, err
	}

	return addresses, nil
}

// addAddress adds an address to the list
func (ws *WalletStorage) addAddress(address string) error {
	addresses, err := ws.GetAllAddresses()
	if err != nil {
		return err
	}

	// Check if address already exists
	for _, addr := range addresses {
		if addr == address {
			return nil
		}
	}

	addresses = append(addresses, address)

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(addresses); err != nil {
		return err
	}

	return ws.db.Put([]byte(addressKey), buf.Bytes(), nil)
}

// DeleteWallet removes a wallet from the database
func (ws *WalletStorage) DeleteWallet(address string) error {
	key := []byte(walletPrefix + address)
	return ws.db.Delete(key, nil)
}

// WalletExists checks if a wallet exists
func (ws *WalletStorage) WalletExists(address string) bool {
	key := []byte(walletPrefix + address)
	exists, _ := ws.db.Has(key, nil)
	return exists
}

// Clear removes all wallets
func (ws *WalletStorage) Clear() error {
	addresses, err := ws.GetAllAddresses()
	if err != nil {
		return err
	}

	for _, address := range addresses {
		if err := ws.DeleteWallet(address); err != nil {
			return err
		}
	}

	return ws.db.Delete([]byte(addressKey), nil)
}

// GetWalletPath returns the default wallet storage path
func GetWalletPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./wallets.db"
	}
	return homeDir + "/.btc_wallets"
}
