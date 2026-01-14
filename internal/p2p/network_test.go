package p2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yourusername/bt/internal/blockchain"
	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/pkg/types"
)

func TestNewNetwork(t *testing.T) {
	// Create blockchain
	wallet, _ := crypto.NewWallet()
	bc, err := blockchain.NewBlockchain(wallet.GetAddress(), ":memory:")
	if err != nil {
		bc, _ = blockchain.NewBlockchain(wallet.GetAddress(), "./test-p2p-new.db")
		defer bc.Close()
	}

	ctx := context.Background()

	// Create network
	network, err := NewNetwork(ctx, bc, "/ip4/127.0.0.1/tcp/0")
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}
	defer network.Stop()

	if network == nil {
		t.Fatal("Network is nil")
	}

	if network.host == nil {
		t.Fatal("Host is nil")
	}

	if network.blockchain != bc {
		t.Error("Blockchain not set correctly")
	}
}

func TestNetworkStart(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), "./test-p2p-start.db")
	defer bc.Close()

	ctx := context.Background()
	network, err := NewNetwork(ctx, bc, "/ip4/127.0.0.1/tcp/0")
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}
	defer network.Stop()

	if err := network.Start(); err != nil {
		t.Errorf("Failed to start network: %v", err)
	}

	if len(network.host.Addrs()) == 0 {
		t.Error("No listen addresses")
	}

	// Node should have an ID
	if network.host.ID() == "" {
		t.Error("Node has no ID")
	}
}

func TestPeerConnection(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping peer connection test in short mode")
	}

	// Create two nodes
	wallet1, _ := crypto.NewWallet()
	wallet2, _ := crypto.NewWallet()

	bc1, _ := blockchain.NewBlockchain(wallet1.GetAddress(), "./test-p2p-peer1.db")
	defer bc1.Close()

	bc2, _ := blockchain.NewBlockchain(wallet2.GetAddress(), "./test-p2p-peer2.db")
	defer bc2.Close()

	ctx := context.Background()

	// Node 1
	network1, err := NewNetwork(ctx, bc1, "/ip4/127.0.0.1/tcp/9101")
	if err != nil {
		t.Fatalf("Failed to create network1: %v", err)
	}
	defer network1.Stop()
	network1.Start()

	// Node 2
	network2, err := NewNetwork(ctx, bc2, "/ip4/127.0.0.1/tcp/9102")
	if err != nil {
		t.Fatalf("Failed to create network2: %v", err)
	}
	defer network2.Stop()
	network2.Start()

	// Connect node 2 to node 1
	node1Addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/9101/p2p/%s", network1.host.ID().String())
	
	if err := network2.ConnectToPeer(node1Addr); err != nil {
		t.Fatalf("Failed to connect peers: %v", err)
	}

	// Wait for connection
	time.Sleep(1 * time.Second)

	// Check peer count
	if network2.GetPeerCount() == 0 {
		t.Error("Node 2 has no peers after connection")
	}

	peers := network2.GetPeers()
	if len(peers) == 0 {
		t.Error("GetPeers returned empty list")
	}
}

func TestMessageBroadcast(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping broadcast test in short mode")
	}

	wallet1, _ := crypto.NewWallet()
	wallet2, _ := crypto.NewWallet()

	bc1, _ := blockchain.NewBlockchain(wallet1.GetAddress(), "./test-p2p-broadcast1.db")
	defer bc1.Close()

	bc2, _ := blockchain.NewBlockchain(wallet2.GetAddress(), "./test-p2p-broadcast2.db")
	defer bc2.Close()

	ctx := context.Background()

	network1, _ := NewNetwork(ctx, bc1, "/ip4/127.0.0.1/tcp/9201")
	defer network1.Stop()
	network1.Start()

	network2, _ := NewNetwork(ctx, bc2, "/ip4/127.0.0.1/tcp/9202")
	defer network2.Stop()
	network2.Start()

	// Connect
	node1Addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/9201/p2p/%s", network1.host.ID().String())
	network2.ConnectToPeer(node1Addr)
	time.Sleep(1 * time.Second)

	// Set up receiver
	received := false
	network2.SetBlockHandler(func(block *types.Block) {
		received = true
	})

	// Broadcast from node 1
	block := bc1.GetLatestBlock()
	network1.BroadcastBlock(block)

	// Wait for message
	time.Sleep(2 * time.Second)

	if !received {
		t.Error("Block was not received by peer")
	}
}

func TestGetPeerCount(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), "./test-p2p-count.db")
	defer bc.Close()

	ctx := context.Background()
	network, _ := NewNetwork(ctx, bc, "/ip4/127.0.0.1/tcp/0")
	defer network.Stop()

	count := network.GetPeerCount()
	if count != 0 {
		t.Errorf("Expected 0 peers, got %d", count)
	}
}

func TestGetPeers(t *testing.T) {
	wallet, _ := crypto.NewWallet()
	bc, _ := blockchain.NewBlockchain(wallet.GetAddress(), "./test-p2p-getpeers.db")
	defer bc.Close()

	ctx := context.Background()
	network, _ := NewNetwork(ctx, bc, "/ip4/127.0.0.1/tcp/0")
	defer network.Stop()

	peers := network.GetPeers()
	if peers == nil {
		t.Error("GetPeers returned nil")
	}

	if len(peers) != 0 {
		t.Errorf("Expected 0 peers, got %d", len(peers))
	}
}
