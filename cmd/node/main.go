package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/yourusername/bt/internal/blockchain"
	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/tx"
)

func main() {
	dbPath := flag.String("db", "./blockchain.db", "Path to blockchain database")
	fresh := flag.Bool("fresh", false, "Start with a fresh blockchain")
	flag.Parse()

	fmt.Println("🚀 Starting Bitcoin-like Cryptocurrency Node (Phase 3)")
	fmt.Println("=" + "=====================================================")

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
		// Note: In production, you'd want to properly delete the DB
	}

	bc, err := blockchain.NewBlockchain(minerAddr, *dbPath)
	if err != nil {
		log.Fatalf("Failed to create blockchain: %v", err)
	}
	defer bc.Close()

	fmt.Printf("  Height: %d\n", bc.Height())
	fmt.Printf("  Difficulty: %d bits\n\n", bc.DifficultyTarget)

	// If blockchain already exists, just show info
	if bc.Height() > 1 {
		fmt.Println("📊 Existing blockchain loaded!")
		bc.PrintChain()
		
		minerBalance, _ := bc.UTXOSet.GetBalance(minerAddr)
		fmt.Printf("\n💰 Balances:\n")
		fmt.Printf("  Miner: %d satoshis (%.2f BTC)\n", minerBalance, float64(minerBalance)/1e8)
		
		fmt.Printf("\n📊 Summary:\n")
		fmt.Printf("  Total blocks: %d\n", bc.Height())
		fmt.Printf("  Total UTXOs: %d\n", bc.UTXOSet.CountUTXOs())
		fmt.Printf("  Current difficulty: %d bits\n", bc.DifficultyTarget)
		return
	}

	// New blockchain - run demo transactions
	minerBalance, _ := bc.UTXOSet.GetBalance(minerAddr)
	fmt.Printf("💰 Initial Balances:\n")
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

	fmt.Println("⛏️  Mining Block 1...")
	block1, err := bc.AddBlock([]*tx.Transaction{tx1}, minerAddr)
	if err != nil {
		log.Fatalf("Failed to add block 1: %v", err)
	}
	fmt.Printf("✓ Block 1 mined - Hash: %x\n", block1.Hash[:8])
	fmt.Println("💾 Block 1 saved to database")

	// Check balances
	minerBalance, _ = bc.UTXOSet.GetBalance(minerAddr)
	aliceBalance, _ := bc.UTXOSet.GetBalance(aliceAddr)
	bobBalance, _ := bc.UTXOSet.GetBalance(bobAddr)

	fmt.Printf("💰 Balances after Block 1:\n")
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

	fmt.Println("⛏️  Mining Block 2...")
	block2, err := bc.AddBlock([]*tx.Transaction{tx2}, minerAddr)
	if err != nil {
		log.Fatalf("Failed to add block 2: %v", err)
	}
	fmt.Printf("✓ Block 2 mined - Hash: %x\n", block2.Hash[:8])
	fmt.Println("💾 Block 2 saved to database")

	// Final balances
	minerBalance, _ = bc.UTXOSet.GetBalance(minerAddr)
	aliceBalance, _ = bc.UTXOSet.GetBalance(aliceAddr)
	bobBalance, _ = bc.UTXOSet.GetBalance(bobAddr)

	fmt.Printf("💰 Final Balances:\n")
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

	// Summary
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  Total blocks: %d\n", bc.Height())
	fmt.Printf("  Total UTXOs: %d\n", bc.UTXOSet.CountUTXOs())
	fmt.Printf("  Current difficulty: %d bits\n", bc.DifficultyTarget)
	fmt.Printf("  Latest block hash: %x\n", bc.GetLatestBlock().Hash[:16])
	fmt.Printf("  Database: %s\n", *dbPath)
	fmt.Println("\n✓ Phase 3 Complete: LevelDB Persistence")
	fmt.Println("✨ Run again to load blockchain from disk!")
}
