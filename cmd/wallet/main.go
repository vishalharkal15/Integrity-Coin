package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/storage"
)

func main() {
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)

	balanceAddress := balanceCmd.String("address", "", "Address to check balance")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create":
		createCmd.Parse(os.Args[2:])
		createWallet()

	case "balance":
		balanceCmd.Parse(os.Args[2:])
		if *balanceAddress == "" {
			fmt.Println("Error: --address is required")
			balanceCmd.PrintDefaults()
			os.Exit(1)
		}
		checkBalance(*balanceAddress)

	case "list":
		listCmd.Parse(os.Args[2:])
		listWallets()

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Bitcoin-like Cryptocurrency Wallet")
	fmt.Println("\nUsage:")
	fmt.Println("  wallet create                    Create a new wallet")
	fmt.Println("  wallet balance --address <addr>  Check balance of an address")
	fmt.Println("  wallet list                      List all wallets")
}

func createWallet() {
	wallet, err := crypto.NewWallet()
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	address := wallet.GetAddress()
	privateKey := crypto.PrivateKeyToHex(wallet.PrivateKey)

	// Save wallet to storage
	walletStore, err := storage.NewWalletStorage(storage.GetWalletPath())
	if err != nil {
		log.Fatalf("Failed to open wallet storage: %v", err)
	}
	defer walletStore.Close()

	if err := walletStore.SaveWallet(address, wallet.PrivateKey.D.Bytes(), wallet.PublicKey); err != nil {
		log.Fatalf("Failed to save wallet: %v", err)
	}

	fmt.Println("\n✓ New Wallet Created & Saved")
	fmt.Println("==========================================")
	fmt.Printf("Address:     %s\n", address)
	fmt.Printf("Private Key: %s\n", privateKey)
	fmt.Println("==========================================")
	fmt.Println("\n⚠️  IMPORTANT: Save your private key securely!")
	fmt.Println("Anyone with your private key can access your funds.")
	fmt.Printf("\n💾 Wallet saved to: %s\n", storage.GetWalletPath())
}

func checkBalance(address string) {
	// Validate address format
	_, err := crypto.DecodeAddress(address)
	if err != nil {
		log.Fatalf("Invalid address: %v", err)
	}

	fmt.Printf("\nAddress: %s\n", address)
	fmt.Println("Balance check requires blockchain connection (use node to check)")
	fmt.Println("For now, this validates the address format only.")
	fmt.Println("✓ Address is valid")
}

func listWallets() {
	walletStore, err := storage.NewWalletStorage(storage.GetWalletPath())
	if err != nil {
		log.Fatalf("Failed to open wallet storage: %v", err)
	}
	defer walletStore.Close()

	addresses, err := walletStore.GetAllAddresses()
	if err != nil {
		log.Fatalf("Failed to get wallets: %v", err)
	}

	if len(addresses) == 0 {
		fmt.Println("\n📭 No wallets found. Create one with: wallet create")
		return
	}

	fmt.Printf("\n💼 Saved Wallets (%d):\n", len(addresses))
	fmt.Println("==========================================")
	for i, addr := range addresses {
		fmt.Printf("%d. %s\n", i+1, addr)
	}
	fmt.Println("==========================================")
}
