package grpc

import (
	"testing"
	"context"
	"time"

	pb "github.com/yourusername/bt/api/proto"
	"github.com/yourusername/bt/internal/blockchain"
	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/tx"
)

func TestNewServer(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	if server == nil {
		t.Fatal("Expected server to be created")
	}
	
	if server.bc != bc {
		t.Error("Blockchain not set correctly")
	}
}

func TestGetBlockchainInfo(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	req := &pb.GetBlockchainInfoRequest{}
	info, err := server.GetBlockchainInfo(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetBlockchainInfo failed: %v", err)
	}
	
	if info.Height < 0 {
		t.Error("Invalid blockchain height")
	}
	
	if info.Difficulty <= 0 {
		t.Error("Invalid difficulty")
	}
}

func TestGetBlockHeight(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	req := &pb.GetBlockHeightRequest{}
	resp, err := server.GetBlockHeight(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetBlockHeight failed: %v", err)
	}
	
	if resp.Height < 0 {
		t.Error("Invalid height")
	}
}

func TestGetBlockByHeight(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Get genesis block
	req := &pb.GetBlockByHeightRequest{Height: 0}
	block, err := server.GetBlockByHeight(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetBlockByHeight failed: %v", err)
	}
	
	if block.Height != 0 {
		t.Errorf("Expected height 0, got %d", block.Height)
	}
	
	if block.Hash == "" {
		t.Error("Block hash is empty")
	}
}

func TestGetBestBlockHash(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	req := &pb.GetBestBlockHashRequest{}
	resp, err := server.GetBestBlockHash(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetBestBlockHash failed: %v", err)
	}
	
	if resp.Hash == "" {
		t.Error("Best block hash is empty")
	}
}

func TestGetMempool(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	req := &pb.GetMempoolRequest{}
	resp, err := server.GetMempool(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetMempool failed: %v", err)
	}
	
	// Initially should be empty
	if resp.Count != 0 {
		t.Errorf("Expected empty mempool, got %d transactions", resp.Count)
	}
}

func TestSubmitTransaction(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Create a test transaction
	pbTx := &pb.Transaction{
		Id:        "test-tx-id",
		Timestamp: nil,
		Inputs: []*pb.TxInput{
			{
				TxId:      "prev-tx-id",
				Vout:      0,
				Signature: "test-sig",
				PublicKey: "test-pubkey",
			},
		},
		Outputs: []*pb.TxOutput{
			{
				Value:         100,
				PublicKeyHash: "test-hash",
			},
		},
	}
	
	req := &pb.SubmitTransactionRequest{Transaction: pbTx}
	resp, err := server.SubmitTransaction(context.Background(), req)
	
	if err != nil {
		t.Fatalf("SubmitTransaction failed: %v", err)
	}
	
	if !resp.Accepted {
		t.Error("Transaction not accepted")
	}
	
	if resp.TxId == "" {
		t.Error("Transaction ID is empty")
	}
	
	// Check if transaction is in mempool
	mempoolReq := &pb.GetMempoolRequest{}
	mempoolResp, _ := server.GetMempool(context.Background(), mempoolReq)
	
	if mempoolResp.Count != 1 {
		t.Errorf("Expected 1 transaction in mempool, got %d", mempoolResp.Count)
	}
}

func TestCreateWallet(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	req := &pb.CreateWalletRequest{Name: "test-wallet"}
	wallet, err := server.CreateWallet(context.Background(), req)
	
	if err != nil {
		t.Fatalf("CreateWallet failed: %v", err)
	}
	
	if wallet.Address == "" {
		t.Error("Wallet address is empty")
	}
	
	if wallet.PublicKey == "" {
		t.Error("Wallet public key is empty")
	}
}

func TestListWallets(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Create a few wallets
	server.CreateWallet(context.Background(), &pb.CreateWalletRequest{Name: "wallet1"})
	server.CreateWallet(context.Background(), &pb.CreateWalletRequest{Name: "wallet2"})
	
	req := &pb.ListWalletsRequest{}
	resp, err := server.ListWallets(context.Background(), req)
	
	if err != nil {
		t.Fatalf("ListWallets failed: %v", err)
	}
	
	if len(resp.Wallets) != 2 {
		t.Errorf("Expected 2 wallets, got %d", len(resp.Wallets))
	}
}

func TestGetBalance(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Create a wallet
	walletResp, _ := server.CreateWallet(context.Background(), &pb.CreateWalletRequest{Name: "test"})
	
	req := &pb.GetBalanceRequest{Address: walletResp.Address}
	resp, err := server.GetBalance(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}
	
	// New wallet should have 0 balance
	if resp.Balance != 0 {
		t.Errorf("Expected balance 0, got %d", resp.Balance)
	}
}

func TestGetUTXO(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Create a wallet
	walletResp, _ := server.CreateWallet(context.Background(), &pb.CreateWalletRequest{Name: "test"})
	
	req := &pb.GetUTXORequest{Address: walletResp.Address}
	resp, err := server.GetUTXO(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetUTXO failed: %v", err)
	}
	
	// New wallet should have no UTXOs
	if len(resp.Utxos) != 0 {
		t.Errorf("Expected 0 UTXOs, got %d", len(resp.Utxos))
	}
}

func TestGetPeerInfo(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	req := &pb.GetPeerInfoRequest{}
	resp, err := server.GetPeerInfo(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetPeerInfo failed: %v", err)
	}
	
	// No network, should have 0 peers
	if resp.PeerCount != 0 {
		t.Errorf("Expected 0 peers, got %d", resp.PeerCount)
	}
}

func TestGetMiningInfo(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	req := &pb.GetMiningInfoRequest{}
	info, err := server.GetMiningInfo(context.Background(), req)
	
	if err != nil {
		t.Fatalf("GetMiningInfo failed: %v", err)
	}
	
	if info.IsMining {
		t.Error("Mining should not be active initially")
	}
	
	if info.CurrentDifficulty <= 0 {
		t.Error("Invalid difficulty")
	}
}

func TestStartStopMining(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping mining test in short mode")
	}
	
	bc := blockchain.NewBlockchain("")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Create a miner wallet
	wallet, _ := crypto.NewWallet()
	minerAddress := wallet.GetAddress()
	
	// Start mining
	startReq := &pb.StartMiningRequest{MinerAddress: minerAddress}
	startResp, err := server.StartMining(context.Background(), startReq)
	
	if err != nil {
		t.Fatalf("StartMining failed: %v", err)
	}
	
	if !startResp.Success {
		t.Error("Mining did not start successfully")
	}
	
	// Wait a bit for mining to start
	time.Sleep(100 * time.Millisecond)
	
	// Check mining info
	infoReq := &pb.GetMiningInfoRequest{}
	info, _ := server.GetMiningInfo(context.Background(), infoReq)
	
	if !info.IsMining {
		t.Error("Mining should be active")
	}
	
	// Stop mining
	stopReq := &pb.StopMiningRequest{}
	stopResp, err := server.StopMining(context.Background(), stopReq)
	
	if err != nil {
		t.Fatalf("StopMining failed: %v", err)
	}
	
	if !stopResp.Success {
		t.Error("Mining did not stop successfully")
	}
	
	// Verify mining stopped
	time.Sleep(100 * time.Millisecond)
	info, _ = server.GetMiningInfo(context.Background(), infoReq)
	
	if info.IsMining {
		t.Error("Mining should be stopped")
	}
}

func TestSendTransaction(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Create sender and receiver wallets
	senderResp, _ := server.CreateWallet(context.Background(), &pb.CreateWalletRequest{Name: "sender"})
	receiverResp, _ := server.CreateWallet(context.Background(), &pb.CreateWalletRequest{Name: "receiver"})
	
	// Try to send transaction (will fail due to insufficient funds)
	req := &pb.SendTransactionRequest{
		FromAddress: senderResp.Address,
		ToAddress:   receiverResp.Address,
		Amount:      100,
	}
	
	resp, err := server.SendTransaction(context.Background(), req)
	
	if err != nil {
		t.Fatalf("SendTransaction failed: %v", err)
	}
	
	// Should fail due to insufficient funds
	if resp.Success {
		t.Error("Transaction should fail due to insufficient funds")
	}
}

func TestBlockToProto(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Get genesis block
	block, _ := bc.GetBlockByHeight(0)
	
	pbBlock := server.blockToProto(block)
	
	if pbBlock.Hash == "" {
		t.Error("Block hash is empty")
	}
	
	if pbBlock.Height != int64(block.Height) {
		t.Error("Block height mismatch")
	}
	
	if pbBlock.PreviousHash != block.PrevHash {
		t.Error("Previous hash mismatch")
	}
}

func TestTxToProto(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	defer bc.Close()
	
	server := NewServer(bc, nil)
	
	// Create a test transaction
	transaction := &tx.Transaction{
		ID:        "test-tx",
		Timestamp: time.Now().Unix(),
		Inputs: []tx.TxInput{
			{
				TxID:      "prev-tx",
				Vout:      0,
				Signature: "sig",
				PublicKey: "pubkey",
			},
		},
		Outputs: []tx.TxOutput{
			{
				Value:         100,
				PublicKeyHash: "hash",
			},
		},
	}
	
	pbTx := server.txToProto(transaction)
	
	if pbTx.Id != transaction.ID {
		t.Error("Transaction ID mismatch")
	}
	
	if len(pbTx.Inputs) != len(transaction.Inputs) {
		t.Error("Input count mismatch")
	}
	
	if len(pbTx.Outputs) != len(transaction.Outputs) {
		t.Error("Output count mismatch")
	}
}
