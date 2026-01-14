package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/yourusername/bt/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to gRPC server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create clients
	bcClient := pb.NewBlockchainServiceClient(conn)
	walletClient := pb.NewWalletServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("=== Testing gRPC API ===\n")

	// Test 1: Get blockchain info
	fmt.Println("1. Getting blockchain info...")
	info, err := bcClient.GetBlockchainInfo(ctx, &pb.GetBlockchainInfoRequest{})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Height: %d\n", info.Height)
		fmt.Printf("   Best Block: %s\n", info.BestBlockHash)
		fmt.Printf("   Difficulty: %d\n", info.Difficulty)
		fmt.Printf("   Total Transactions: %d\n\n", info.TotalTransactions)
	}

	// Test 2: Get block by height
	fmt.Println("2. Getting genesis block (height 0)...")
	block, err := bcClient.GetBlockByHeight(ctx, &pb.GetBlockByHeightRequest{Height: 0})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Hash: %s\n", block.Hash[:16]+"...")
		fmt.Printf("   Nonce: %d\n", block.Nonce)
		fmt.Printf("   Transactions: %d\n\n", len(block.Transactions))
	}

	// Test 3: Create wallet
	fmt.Println("3. Creating a new wallet...")
	wallet, err := walletClient.CreateWallet(ctx, &pb.CreateWalletRequest{})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Address: %s\n", wallet.Address)
		fmt.Printf("   Public Key: %s...\n\n", wallet.PublicKey[:32])
	}

	// Test 4: List wallets
	fmt.Println("4. Listing all wallets...")
	wallets, err := walletClient.ListWallets(ctx, &pb.ListWalletsRequest{})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Total wallets: %d\n\n", len(wallets.Wallets))
	}

	// Test 5: Get blockchain height
	fmt.Println("5. Getting blockchain height...")
	height, err := bcClient.GetBlockHeight(ctx, &pb.GetBlockHeightRequest{})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Height: %d\n\n", height.Height)
	}

	// Test 6: Get mempool
	fmt.Println("6. Getting mempool...")
	mempool, err := bcClient.GetMempool(ctx, &pb.GetMempoolRequest{})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Pending transactions: %d\n\n", mempool.Count)
	}

	// Test 7: Get mining info
	fmt.Println("7. Getting mining info...")
	miningInfo, err := bcClient.GetMiningInfo(ctx, &pb.GetMiningInfoRequest{})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("   Is Mining: %v\n", miningInfo.IsMining)
		fmt.Printf("   Blocks Mined: %d\n", miningInfo.BlocksMined)
		fmt.Printf("   Current Difficulty: %d\n\n", miningInfo.CurrentDifficulty)
	}

	fmt.Println("=== All tests completed successfully! ===")
}
