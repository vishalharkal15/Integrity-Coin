package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/bt/internal/blockchain"
	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/grpc"
)

func main() {
	// Command-line flags
	dbPath := flag.String("db", "./data/blockchain", "Path to blockchain database")
	fresh := flag.Bool("fresh", false, "Start with a fresh blockchain")
	grpcAddr := flag.String("grpc", ":50051", "gRPC server address")
	flag.Parse()

	// Delete old database if fresh start
	if *fresh {
		log.Println("Starting with fresh blockchain...")
		os.RemoveAll(*dbPath)
	}

	// Create blockchain
	wallet, err := crypto.NewWallet()
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}
	minerAddr := wallet.GetAddress()
	bc, err := blockchain.NewBlockchain(minerAddr, *dbPath)
	if err != nil {
		log.Fatalf("Failed to create blockchain: %v", err)
	}
	defer bc.Close()

	log.Printf("Blockchain initialized with height: %d", bc.Height())
	log.Printf("Current difficulty: %d", bc.DifficultyTarget)

	// Create and start gRPC server (without P2P for now)
	log.Printf("Starting gRPC server on %s", *grpcAddr)
	server := grpc.NewServer(bc, nil)

	// Start gRPC server in goroutine
	go func() {
		if err := server.Start(*grpcAddr); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(500 * time.Millisecond)
	log.Println("✅ gRPC server started successfully")
	
	log.Println("\n=== Node Information ===")
	log.Printf("Blockchain Height: %d", bc.Height())
	log.Printf("gRPC Address: %s", *grpcAddr)
	log.Println("========================\n")

	// Example: Create a wallet on startup
	log.Printf("Created wallet with address: %s", wallet.GetAddress())

	// Print usage instructions
	printUsageInstructions(*grpcAddr)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down gracefully...")
	server.Stop()
	log.Println("Node stopped")
}

func printUsageInstructions(grpcAddr string) {
	fmt.Println("\n=== Usage Instructions ===")
	fmt.Println("\nYou can interact with the node using gRPC clients.")
	fmt.Println("\nExample using grpcurl:")
	fmt.Printf("  # Get blockchain info\n")
	fmt.Printf("  grpcurl -plaintext %s blockchain.BlockchainService/GetBlockchainInfo\n\n", grpcAddr)
	fmt.Printf("  # Get block by height\n")
	fmt.Printf("  grpcurl -plaintext -d '{\"height\": 0}' %s blockchain.BlockchainService/GetBlockByHeight\n\n", grpcAddr)
	fmt.Printf("  # Create a wallet\n")
	fmt.Printf("  grpcurl -plaintext -d '{}' %s blockchain.WalletService/CreateWallet\n\n", grpcAddr)
	fmt.Printf("  # Get balance\n")
	fmt.Printf("  grpcurl -plaintext -d '{\"address\": \"<address>\"}' %s blockchain.BlockchainService/GetBalance\n\n", grpcAddr)
	fmt.Printf("  # Start mining\n")
	fmt.Printf("  grpcurl -plaintext -d '{\"miner_address\": \"<address>\"}' %s blockchain.BlockchainService/StartMining\n\n", grpcAddr)
	fmt.Println("\nInstall grpcurl: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest")
	fmt.Println("\nPress Ctrl+C to stop the node")
	fmt.Println("===========================\n")
}
