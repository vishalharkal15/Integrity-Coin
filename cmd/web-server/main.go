package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	proto "github.com/yourusername/bt/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Configuration
const (
	grpcAddress = "localhost:50051"
	httpPort    = ":8080"
)

// Global gRPC clients
var (
	blockchainClient proto.BlockchainServiceClient
	walletClient     proto.WalletServiceClient
)

// Response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// CORS middleware
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// Helper to send JSON response
func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		},
	})
}

// Get blockchain info
func getBlockchainInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := blockchainClient.GetBlockchainInfo(ctx, &proto.GetBlockchainInfoRequest{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"height":    info.Height,
			"bestBlock": info.BestBlockHash,
			"difficulty": info.Difficulty,
		},
	})
}

// Get recent blocks
func getRecentBlocksHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get count parameter
	countStr := r.URL.Query().Get("count")
	count := 10
	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
			count = c
		}
	}

	// Get blockchain height
	heightResp, err := blockchainClient.GetBlockHeight(ctx, &proto.GetBlockHeightRequest{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	height := int(heightResp.Height)
	blocks := []map[string]interface{}{}

	// Fetch recent blocks
	start := height - count
	if start < 0 {
		start = 0
	}

	for i := height; i >= start && i >= 0; i-- {
		block, err := blockchainClient.GetBlockByHeight(ctx, &proto.GetBlockByHeightRequest{
			Height: int64(i),
		})
		if err != nil {
			continue
		}

		blocks = append(blocks, map[string]interface{}{
			"height":           i,
			"hash":             block.Hash,
			"prevHash":         block.PreviousHash,
			"timestamp":        block.Timestamp.AsTime().Unix(),
			"nonce":            block.Nonce,
			"transactionCount": len(block.Transactions),
		})
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"blocks": blocks,
		},
	})
}

// Get block by height
func getBlockHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Extract height from URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid block height",
		})
		return
	}

	heightStr := parts[len(parts)-1]
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid block height",
		})
		return
	}

	block, err := blockchainClient.GetBlockByHeight(ctx, &proto.GetBlockByHeightRequest{
		Height: int64(height),
	})
	if err != nil {
		sendJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"height":           height,
			"hash":             block.Hash,
			"prevHash":         block.PreviousHash,
			"timestamp":        block.Timestamp.AsTime().Unix(),
			"nonce":            block.Nonce,
			"transactionCount": len(block.Transactions),
			"transactions":     block.Transactions,
		},
	})
}

// Get block height
func getBlockHeightHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	height, err := blockchainClient.GetBlockHeight(ctx, &proto.GetBlockHeightRequest{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"height": height.Height,
		},
	})
}

// List wallets
func listWalletsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wallets, err := walletClient.ListWallets(ctx, &proto.ListWalletsRequest{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	walletList := []map[string]interface{}{}
	for _, wallet := range wallets.Wallets {
		balance, _ := walletClient.GetWalletBalance(ctx, &proto.GetWalletBalanceRequest{Address: wallet.Address})
		walletList = append(walletList, map[string]interface{}{
			"address": wallet.Address,
			"balance": balance.GetBalance(),
		})
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"wallets": walletList,
		},
	})
}

// Create wallet
func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wallet, err := walletClient.CreateWallet(ctx, &proto.CreateWalletRequest{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"address": wallet.Address,
		},
	})
}

// Get mempool
func getMempoolHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mempool, err := blockchainClient.GetMempool(ctx, &proto.GetMempoolRequest{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	transactions := []map[string]interface{}{}
	for _, tx := range mempool.Transactions {
		transactions = append(transactions, map[string]interface{}{
			"id":          tx.Id,
			"inputCount":  len(tx.Inputs),
			"outputCount": len(tx.Outputs),
		})
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"transactions": transactions,
		},
	})
}

// Get mining info
func getMiningInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := blockchainClient.GetMiningInfo(ctx, &proto.GetMiningInfoRequest{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"difficulty":  info.CurrentDifficulty,
			"isMining":    info.IsMining,
			"blockReward": 50, // Default block reward
		},
	})
}

// Initialize gRPC connection
func initGRPCClients() error {
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	blockchainClient = proto.NewBlockchainServiceClient(conn)
	walletClient = proto.NewWalletServiceClient(conn)

	log.Printf("Connected to gRPC server at %s", grpcAddress)
	return nil
}

func main() {
	// Initialize gRPC clients
	if err := initGRPCClients(); err != nil {
		log.Fatalf("Failed to initialize gRPC clients: %v", err)
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("./web/static"))
	mux.Handle("/", fs)

	// API routes
	mux.HandleFunc("/api/health", corsMiddleware(healthHandler))
	mux.HandleFunc("/api/blockchain/info", corsMiddleware(getBlockchainInfoHandler))
	mux.HandleFunc("/api/blockchain/blocks", corsMiddleware(getRecentBlocksHandler))
	mux.HandleFunc("/api/blockchain/block/", corsMiddleware(getBlockHandler))
	mux.HandleFunc("/api/blockchain/height", corsMiddleware(getBlockHeightHandler))
	mux.HandleFunc("/api/wallet/list", corsMiddleware(listWalletsHandler))
	mux.HandleFunc("/api/wallet/create", corsMiddleware(createWalletHandler))
	mux.HandleFunc("/api/mempool", corsMiddleware(getMempoolHandler))
	mux.HandleFunc("/api/mining/info", corsMiddleware(getMiningInfoHandler))

	// Start server
	log.Printf("Starting web server on %s", httpPort)
	log.Printf("Visit http://localhost%s to view the blockchain explorer", httpPort)
	if err := http.ListenAndServe(httpPort, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
