package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/bt/internal/blockchain"
	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/p2p"
	"github.com/yourusername/bt/internal/tx"
	"github.com/yourusername/bt/pkg/types"
)

func main() {
	// Command line flags
	dbPath := flag.String("db", "./blockchain.db", "Path to blockchain database")
	fresh := flag.Bool("fresh", false, "Start with a fresh blockchain")
	listen := flag.String("listen", "/ip4/0.0.0.0/tcp/9000", "P2P listen address")
	connect := flag.String("connect", "", "Connect to peer (e.g., /ip4/127.0.0.1/tcp/9000/p2p/...)")
	mine := flag.Bool("mine", false, "Enable mining mode")
	flag.Parse()

	fmt.Println("🚀 Starting Bitcoin-like Cryptocurrency Node (Phase 4 - P2P)")
	fmt.Println("================================================================")

	// Create wallets for testing
	fmt.Println("\n📝 Creating test wallets...")
	minerWallet, _ := crypto.NewWallet()
	aliceWallet, _ := crypto.NewWallet()
	bobWallet, _ := crypto.NewWallet()

	minerAddr := minerWallet.GetAddress()
	aliceAddr := aliceWallet.GetAddress()
	bobAddr := bobWallet.GetAddress()

	fmt.Printf("Miner:  %s\n", minerAddr)
	fmt.Printf("Alice:  %s\n", aliceAddr)
	fmt.Printf("Bob:    %s\n", bobAddr)

	// Create or load blockchain
	fmt.Printf("\n💾 Database: %s\n", *dbPath)
	if *fresh {
		fmt.Println("🗑️  Removing old database...")
		os.RemoveAll(*dbPath)
	}

	bc, err := blockchain.NewBlockchain(minerAddr, *dbPath)
	if err != nil {
		log.Fatalf("Failed to create blockchain: %v", err)
	}
	defer bc.Close()

	fmt.Printf("  Height: %d\n", bc.Height())
	fmt.Printf("  Difficulty: %d bits\n", bc.DifficultyTarget)

	// Create P2P network
	fmt.Printf("\n🌐 Starting P2P network...\n")
	fmt.Printf("  Listen address: %s\n", *listen)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	network, err := p2p.NewNetwork(ctx, bc, *listen)
	if err != nil {
		log.Fatalf("Failed to create P2P network: %v", err)
	}
	defer network.Stop()

	if err := network.Start(); err != nil {
		log.Fatalf("Failed to start P2P network: %v", err)
	}

	// Set up handlers for received blocks and transactions
	network.SetBlockHandler(func(block *types.Block) {
		fmt.Printf("📦 New block received from network: %x\n", block.Hash[:8])
	})

	network.SetTxHandler(func(transaction *tx.Transaction) {
		fmt.Printf("💸 New transaction received from network: %x\n", transaction.ID[:8])
	})

	// Connect to peer if specified
	if *connect != "" {
		fmt.Printf("\n🔗 Connecting to peer: %s\n", *connect)
		if err := network.ConnectToPeer(*connect); err != nil {
			log.Printf("Warning: Failed to connect to peer: %v", err)
		}
	}

	// If blockchain already exists and we're not mining, just show info
	if bc.Height() > 1 && !*mine {
		fmt.Println("\n📊 Existing blockchain loaded!")
		bc.PrintChain()

		minerBalance, _ := bc.UTXOSet.GetBalance(minerAddr)
		fmt.Printf("\n💰 Balances:\n")
		fmt.Printf("  Miner: %d satoshis (%.2f BTC)\n", minerBalance, float64(minerBalance)/1e8)

		fmt.Printf("\n📊 Summary:\n")
		fmt.Printf("  Total blocks: %d\n", bc.Height())
		fmt.Printf("  Total UTXOs: %d\n", bc.UTXOSet.CountUTXOs())
		fmt.Printf("  Current difficulty: %d bits\n", bc.DifficultyTarget)
		fmt.Printf("  Connected peers: %d\n", network.GetPeerCount())

		fmt.Println("\n⏸️  Node running in network mode (mining disabled)")
		fmt.Println("   Use --mine flag to enable mining")
		fmt.Println("   Press Ctrl+C to stop")

		// Wait for interrupt
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\n👋 Shutting down...")
		return
	}

	// New blockchain or mining mode - run demo transactions
	if bc.Height() == 1 {
		minerBalance, _ := bc.UTXOSet.GetBalance(minerAddr)
		fmt.Printf("\n💰 Initial Balances:\n")
		fmt.Printf("  Miner: %d satoshis (50 BTC)\n", minerBalance)
		fmt.Printf("  Alice: 0 satoshis\n")
		fmt.Printf("  Bob:   0 satoshis\n\n")

		// Block 1: Miner sends 30 BTC to Alice
		fmt.Println("📤 Creating transaction: Miner -> Alice (30 BTC)")
		tx1, err := bc.CreateTransaction(minerAddr, aliceAddr, 30*1e8, minerWallet)
		if err != nil {
			log.Fatalf("Failed to create transaction: %v", err)
		}
		fmt.Printf("✓ Transaction created: %x\n", tx1.ID[:8])

		// Broadcast transaction to network
		network.BroadcastTransaction(tx1)
		fmt.Println("📡 Transaction broadcasted to network")

		fmt.Println("⛏️  Mining Block 1...")
		block1, err := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
		if err != nil {
			log.Fatalf("Failed to add block 1: %v", err)
		}
		fmt.Printf("✓ Block 1 mined - Hash: %x\n", block1.Hash[:8])
		fmt.Println("💾 Block 1 saved to database")

		// Broadcast block to network
		network.BroadcastBlock(block1)
		fmt.Println("📡 Block broadcasted to network")

		// Check balances
		minerBalance, _ = bc.UTXOSet.GetBalance(minerAddr)
		aliceBalance, _ := bc.UTXOSet.GetBalance(aliceAddr)
		bobBalance, _ := bc.UTXOSet.GetBalance(bobAddr)

		fmt.Printf("\n💰 Balances after Block 1:\n")
		fmt.Printf("  Miner: %d satoshis (%.2f BTC)\n", minerBalance, float64(minerBalance)/1e8)
		fmt.Printf("  Alice: %d satoshis (%.2f BTC)\n", aliceBalance, float64(aliceBalance)/1e8)
		fmt.Printf("  Bob:   %d satoshis (%.2f BTC)\n\n", bobBalance, float64(bobBalance)/1e8)

		// Block 2: Alice sends 10 BTC to Bob
		fmt.Println("📤 Creating transaction: Alice -> Bob (10 BTC)")
		tx2, err := bc.CreateTransaction(aliceAddr, bobAddr, 10*1e8, aliceWallet)
		if err != nil {
			log.Fatalf("Failed to create transaction: %v", err)
		}
		fmt.Printf("✓ Transaction created: %x\n", tx2.ID[:8])

		// Broadcast transaction to network
		network.BroadcastTransaction(tx2)
		fmt.Println("📡 Transaction broadcasted to network")

		fmt.Println("⛏️  Mining Block 2...")
		block2, err := bc.AddBlock([]*tx.Transaction{tx2}, minerAddr)
		if err != nil {
			log.Fatalf("Failed to add block 2: %v", err)
		}
		fmt.Printf("✓ Block 2 mined - Hash: %x\n", block2.Hash[:8])
		fmt.Println("💾 Block 2 saved to database")

		// Broadcast block to network
		network.BroadcastBlock(block2)
		fmt.Println("📡 Block broadcasted to network")

		// Final balances
		minerBalance, _ = bc.UTXOSet.GetBalance(minerAddr)
		aliceBalance, _ = bc.UTXOSet.GetBalance(aliceAddr)
		bobBalance, _ = bc.UTXOSet.GetBalance(bobAddr)

		fmt.Printf("\n💰 Final Balances:\n")
		fmt.Printf("  Miner: %d satoshis (%.2f BTC)\n", minerBalance, float64(minerBalance)/1e8)
		fmt.Printf("  Alice: %d satoshis (%.2f BTC)\n", aliceBalance, float64(aliceBalance)/1e8)
		fmt.Printf("  Bob:   %d satoshis (%.2f BTC)\n\n", bobBalance, float64(bobBalance)/1e8)

		// Validate the blockchain
		fmt.Println("🔍 Validating blockchain...")
		if err := bc.ValidateChain(); err != nil {
			log.Fatalf("Blockchain validation failed: %v", err)
		}
		fmt.Println("✓ Blockchain is valid!")

		// Print blockchain
		bc.PrintChain()
	}

	// Summary
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  Total blocks: %d\n", bc.Height())
	fmt.Printf("  Total UTXOs: %d\n", bc.UTXOSet.CountUTXOs())
	fmt.Printf("  Current difficulty: %d bits\n", bc.DifficultyTarget)
	fmt.Printf("  Latest block hash: %x\n", bc.GetLatestBlock().Hash[:16])
	fmt.Printf("  Database: %s\n", *dbPath)
	fmt.Printf("  Connected peers: %d\n", network.GetPeerCount())

	if network.GetPeerCount() > 0 {
		fmt.Printf("\n👥 Connected Peers:\n")
		for i, peer := range network.GetPeers() {
			fmt.Printf("  %d. %s\n", i+1, peer)
		}
	}

	fmt.Println("\n✓ Phase 4 Complete: P2P Networking")
	fmt.Println("   Node is running and connected to the network")
	fmt.Println("   Press Ctrl+C to stop")

	// Keep running until interrupted
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 Shutting down gracefully...")
}
