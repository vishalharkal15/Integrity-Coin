package grpc

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/yourusername/bt/api/proto"
	"github.com/yourusername/bt/internal/blockchain"
	"github.com/yourusername/bt/internal/crypto"
	"github.com/yourusername/bt/internal/tx"
	"github.com/yourusername/bt/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the gRPC services
type Server struct {
	pb.UnimplementedBlockchainServiceServer
	pb.UnimplementedWalletServiceServer
	
	bc              *blockchain.Blockchain
	wallets         map[string]*crypto.Wallet
	mempool         []*tx.Transaction
	mempoolMu       sync.RWMutex
	
	// Mining control
	isMining        bool
	miningMu        sync.RWMutex
	stopMining      chan struct{}
	minerAddress    string
	blocksMined     int64
	
	// Streaming subscriptions
	blockSubs       []chan *types.Block
	txSubs          []chan *tx.Transaction
	subsMu          sync.RWMutex
	
	grpcServer      *grpc.Server
}

// NewServer creates a new gRPC server
func NewServer(bc *blockchain.Blockchain, network interface{}) *Server {
	return &Server{
		bc:       bc,
		wallets:  make(map[string]*crypto.Wallet),
		mempool:  make([]*tx.Transaction, 0),
		blockSubs: make([]chan *types.Block, 0),
		txSubs:   make([]chan *tx.Transaction, 0),
	}
}

// Start starts the gRPC server
func (s *Server) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	
	s.grpcServer = grpc.NewServer()
	pb.RegisterBlockchainServiceServer(s.grpcServer, s)
	pb.RegisterWalletServiceServer(s.grpcServer, s)
	
	log.Printf("gRPC server listening on %s", address)
	return s.grpcServer.Serve(lis)
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.StopMiningInternal()
		s.grpcServer.GracefulStop()
	}
}

// GetBlockByHash retrieves a block by its hash
func (s *Server) GetBlockByHash(ctx context.Context, req *pb.GetBlockByHashRequest) (*pb.Block, error) {
	// Search through blocks for matching hash
	height := s.bc.Height()
	for i := 0; i <= height; i++ {
		block, err := s.bc.GetBlock(i)
		if err != nil {
			continue
		}
		hashStr := fmt.Sprintf("%x", block.Hash)
		if hashStr == req.Hash {
			return s.blockToProto(block), nil
		}
	}
	
	return nil, fmt.Errorf("block not found")
}

// GetBlockByHeight retrieves a block by its height
func (s *Server) GetBlockByHeight(ctx context.Context, req *pb.GetBlockByHeightRequest) (*pb.Block, error) {
	block, err := s.bc.GetBlock(int(req.Height))
	if err != nil {
		return nil, fmt.Errorf("block not found: %v", err)
	}
	
	return s.blockToProto(block), nil
}

// GetBlockchainInfo returns blockchain information
func (s *Server) GetBlockchainInfo(ctx context.Context, req *pb.GetBlockchainInfoRequest) (*pb.BlockchainInfo, error) {
	height := s.bc.Height()
	bestBlock, _ := s.bc.GetBlock(height)
	
	var bestHash string
	if bestBlock != nil {
		bestHash = fmt.Sprintf("%x", bestBlock.Hash)
	}
	
	return &pb.BlockchainInfo{
		Height:           int64(height),
		BestBlockHash:    bestHash,
		Difficulty:       int64(s.bc.DifficultyTarget),
		TotalTransactions: s.getTotalTransactions(),
		PeerCount:        0, // P2P not integrated yet
		IsSyncing:        false,
	}, nil
}

// GetBestBlockHash returns the hash of the best (latest) block
func (s *Server) GetBestBlockHash(ctx context.Context, req *pb.GetBestBlockHashRequest) (*pb.GetBestBlockHashResponse, error) {
	height := s.bc.Height()
	block, err := s.bc.GetBlock(height)
	if err != nil {
		return nil, fmt.Errorf("failed to get best block: %v", err)
	}
	
	return &pb.GetBestBlockHashResponse{
		Hash: fmt.Sprintf("%x", block.Hash),
	}, nil
}

// GetBlockHeight returns the current blockchain height
func (s *Server) GetBlockHeight(ctx context.Context, req *pb.GetBlockHeightRequest) (*pb.GetBlockHeightResponse, error) {
	return &pb.GetBlockHeightResponse{
		Height: int64(s.bc.Height()),
	}, nil
}

// GetTransaction retrieves a transaction by ID
func (s *Server) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.Transaction, error) {
	// Search in mempool first
	s.mempoolMu.RLock()
	for _, tx := range s.mempool {
		txIDStr := fmt.Sprintf("%x", tx.ID)
		if txIDStr == req.TxId {
			s.mempoolMu.RUnlock()
			return s.txToProto(tx), nil
		}
	}
	s.mempoolMu.RUnlock()
	
	// Search in blockchain
	height := s.bc.Height()
	for i := 0; i <= height; i++ {
		block, err := s.bc.GetBlock(i)
		if err != nil {
			continue
		}
		
		transactions := block.Transactions.([]*tx.Transaction)
		for _, transaction := range transactions {
			txIDStr := fmt.Sprintf("%x", transaction.ID)
			if txIDStr == req.TxId {
				return s.txToProto(transaction), nil
			}
		}
	}
	
	return nil, fmt.Errorf("transaction not found")
}

// SubmitTransaction submits a new transaction to the mempool
func (s *Server) SubmitTransaction(ctx context.Context, req *pb.SubmitTransactionRequest) (*pb.SubmitTransactionResponse, error) {
	if req.Transaction == nil {
		return &pb.SubmitTransactionResponse{
			Accepted: false,
			Message:  "transaction is nil",
		}, nil
	}
	
	// Convert proto transaction to internal type
	transaction := s.protoToTx(req.Transaction)
	
	// Add to mempool
	s.mempoolMu.Lock()
	s.mempool = append(s.mempool, transaction)
	s.mempoolMu.Unlock()
	
	// Notify subscribers
	s.notifyTxSubscribers(transaction)
	
	return &pb.SubmitTransactionResponse{
		TxId:     fmt.Sprintf("%x", transaction.ID),
		Accepted: true,
		Message:  "Transaction accepted into mempool",
	}, nil
}

// GetMempool returns all transactions in the mempool
func (s *Server) GetMempool(ctx context.Context, req *pb.GetMempoolRequest) (*pb.GetMempoolResponse, error) {
	s.mempoolMu.RLock()
	defer s.mempoolMu.RUnlock()
	
	txs := make([]*pb.Transaction, len(s.mempool))
	for i, tx := range s.mempool {
		txs[i] = s.txToProto(tx)
	}
	
	return &pb.GetMempoolResponse{
		Transactions: txs,
		Count:        int32(len(txs)),
	}, nil
}

// GetUTXO returns UTXOs for an address
func (s *Server) GetUTXO(ctx context.Context, req *pb.GetUTXORequest) (*pb.GetUTXOResponse, error) {
	utxos, _ := s.bc.UTXOSet.GetAllUTXOs(req.Address)
	
	pbUtxos := make([]*pb.UTXO, 0)
	totalValue := int64(0)
	
	for _, utxo := range utxos {
		pbUtxos = append(pbUtxos, &pb.UTXO{
			TxId: fmt.Sprintf("%x", utxo.PubKeyHash),
			Vout: 0,
			Output: &pb.TxOutput{
				Value:         int64(utxo.Value),
				PublicKeyHash: fmt.Sprintf("%x", utxo.PubKeyHash),
			},
		})
		totalValue += int64(utxo.Value)
	}
	
	return &pb.GetUTXOResponse{
		Utxos:      pbUtxos,
		TotalValue: totalValue,
	}, nil
}

// GetBalance returns the balance for an address
func (s *Server) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	balance, _ := s.bc.UTXOSet.GetBalance(req.Address)
	utxos, _ := s.bc.UTXOSet.GetAllUTXOs(req.Address)
	
	return &pb.GetBalanceResponse{
		Balance:   int64(balance),
		UtxoCount: int32(len(utxos)),
	}, nil
}

// GetPeerInfo returns information about connected peers (stub for now)
func (s *Server) GetPeerInfo(ctx context.Context, req *pb.GetPeerInfoRequest) (*pb.GetPeerInfoResponse, error) {
	return &pb.GetPeerInfoResponse{
		Peers:     []*pb.PeerInfo{},
		PeerCount: 0,
	}, nil
}

// ConnectPeer connects to a new peer (stub for now)
func (s *Server) ConnectPeer(ctx context.Context, req *pb.ConnectPeerRequest) (*pb.ConnectPeerResponse, error) {
	return &pb.ConnectPeerResponse{
		Success: false,
		Message: "P2P not integrated in this version",
	}, nil
}

// StartMining starts the mining process
func (s *Server) StartMining(ctx context.Context, req *pb.StartMiningRequest) (*pb.StartMiningResponse, error) {
	s.miningMu.Lock()
	if s.isMining {
		s.miningMu.Unlock()
		return &pb.StartMiningResponse{
			Success: false,
			Message: "Mining already in progress",
		}, nil
	}
	
	s.isMining = true
	s.minerAddress = req.MinerAddress
	s.stopMining = make(chan struct{})
	s.miningMu.Unlock()
	
	go s.mineBlocks()
	
	return &pb.StartMiningResponse{
		Success: true,
		Message: "Mining started",
	}, nil
}

// StopMining stops the mining process
func (s *Server) StopMining(ctx context.Context, req *pb.StopMiningRequest) (*pb.StopMiningResponse, error) {
	s.StopMiningInternal()
	
	return &pb.StopMiningResponse{
		Success: true,
		Message: "Mining stopped",
	}, nil
}

// GetMiningInfo returns mining status information
func (s *Server) GetMiningInfo(ctx context.Context, req *pb.GetMiningInfoRequest) (*pb.MiningInfo, error) {
	s.miningMu.RLock()
	defer s.miningMu.RUnlock()
	
	return &pb.MiningInfo{
		IsMining:          s.isMining,
		BlocksMined:       s.blocksMined,
		CurrentDifficulty: int64(s.bc.DifficultyTarget),
		HashRate:          0, // TODO: calculate hash rate
	}, nil
}

// SubscribeBlocks subscribes to new blocks
func (s *Server) SubscribeBlocks(req *pb.SubscribeBlocksRequest, stream pb.BlockchainService_SubscribeBlocksServer) error {
	ch := make(chan *types.Block, 10)
	
	s.subsMu.Lock()
	s.blockSubs = append(s.blockSubs, ch)
	s.subsMu.Unlock()
	
	defer func() {
		s.subsMu.Lock()
		for i, sub := range s.blockSubs {
			if sub == ch {
				s.blockSubs = append(s.blockSubs[:i], s.blockSubs[i+1:]...)
				break
			}
		}
		s.subsMu.Unlock()
		close(ch)
	}()
	
	for {
		select {
		case block := <-ch:
			if err := stream.Send(s.blockToProto(block)); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// SubscribeTransactions subscribes to new transactions
func (s *Server) SubscribeTransactions(req *pb.SubscribeTransactionsRequest, stream pb.BlockchainService_SubscribeTransactionsServer) error {
	ch := make(chan *tx.Transaction, 10)
	
	s.subsMu.Lock()
	s.txSubs = append(s.txSubs, ch)
	s.subsMu.Unlock()
	
	defer func() {
		s.subsMu.Lock()
		for i, sub := range s.txSubs {
			if sub == ch {
				s.txSubs = append(s.txSubs[:i], s.txSubs[i+1:]...)
				break
			}
		}
		s.subsMu.Unlock()
		close(ch)
	}()
	
	for {
		select {
		case transaction := <-ch:
			if err := stream.Send(s.txToProto(transaction)); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// Wallet service methods

// CreateWallet creates a new wallet
func (s *Server) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.Wallet, error) {
	wallet, err := crypto.NewWallet()
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %v", err)
	}
	address := wallet.GetAddress()
	
	s.wallets[address] = wallet
	
	return &pb.Wallet{
		Address:   address,
		PublicKey: fmt.Sprintf("%x", wallet.PublicKey),
	}, nil
}

// GetWallet retrieves a wallet by address
func (s *Server) GetWallet(ctx context.Context, req *pb.GetWalletRequest) (*pb.Wallet, error) {
	wallet, exists := s.wallets[req.Address]
	if !exists {
		return nil, fmt.Errorf("wallet not found")
	}
	
	return &pb.Wallet{
		Address:   wallet.GetAddress(),
		PublicKey: fmt.Sprintf("%x", wallet.PublicKey),
	}, nil
}

// ListWallets lists all wallets
func (s *Server) ListWallets(ctx context.Context, req *pb.ListWalletsRequest) (*pb.ListWalletsResponse, error) {
	wallets := make([]*pb.Wallet, 0, len(s.wallets))
	
	for address, wallet := range s.wallets {
		wallets = append(wallets, &pb.Wallet{
			Address:   address,
			PublicKey: fmt.Sprintf("%x", wallet.PublicKey),
		})
	}
	
	return &pb.ListWalletsResponse{
		Wallets: wallets,
	}, nil
}

// GetWalletBalance returns the balance of a wallet
func (s *Server) GetWalletBalance(ctx context.Context, req *pb.GetWalletBalanceRequest) (*pb.GetWalletBalanceResponse, error) {
	balance, _ := s.bc.UTXOSet.GetBalance(req.Address)
	
	return &pb.GetWalletBalanceResponse{
		Balance: int64(balance),
	}, nil
}

// SendTransaction sends coins from one address to another
func (s *Server) SendTransaction(ctx context.Context, req *pb.SendTransactionRequest) (*pb.SendTransactionResponse, error) {
	wallet, exists := s.wallets[req.FromAddress]
	if !exists {
		return &pb.SendTransactionResponse{
			Success: false,
			Message: "Wallet not found",
		}, nil
	}
	
	// Find spendable outputs
	accumulated, spendableOutputs, err := s.bc.UTXOSet.FindSpendableOutputs(req.FromAddress, req.Amount)
	if err != nil {
		return &pb.SendTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Insufficient funds: %v", err),
		}, nil
	}
	
	// Build inputs from spendable outputs
	var inputs []tx.TxInput
	for txIDStr, outIndices := range spendableOutputs {
		for _, outIdx := range outIndices {
			inputs = append(inputs, tx.TxInput{
				TxID:      []byte(txIDStr),
				OutIndex:  outIdx,
				Signature: nil,
				PubKey:    wallet.PublicKey,
			})
		}
	}
	
	// Decode recipient address
	pubKeyHash, err := crypto.DecodeAddress(req.ToAddress)
	if err != nil {
		return &pb.SendTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid recipient address: %v", err),
		}, nil
	}
	
	// Build outputs
	var outputs []tx.TxOutput
	
	// Create output for recipient
	outputs = append(outputs, tx.TxOutput{
		Value:      req.Amount,
		PubKeyHash: pubKeyHash,
	})
	
	// Create change output if needed
	if accumulated > req.Amount {
		senderPubKeyHash, _ := crypto.DecodeAddress(req.FromAddress)
		outputs = append(outputs, tx.TxOutput{
			Value:      accumulated - req.Amount,
			PubKeyHash: senderPubKeyHash,
		})
	}
	
	// Create transaction
	transaction := tx.NewTransaction(inputs, outputs)
	
	// Sign the transaction (we pass empty prevTxs map for now)
	err = transaction.Sign(wallet, make(map[string]*tx.Transaction))
	if err != nil {
		return &pb.SendTransactionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to sign transaction: %v", err),
		}, nil
	}
	
	// Add to mempool
	s.mempoolMu.Lock()
	s.mempool = append(s.mempool, transaction)
	s.mempoolMu.Unlock()
	
	// Notify subscribers
	s.notifyTxSubscribers(transaction)
	
	return &pb.SendTransactionResponse{
		TxId:    fmt.Sprintf("%x", transaction.ID),
		Success: true,
		Message: "Transaction submitted successfully",
	}, nil
}

// Helper methods

func (s *Server) blockToProto(block *types.Block) *pb.Block {
	transactions := block.Transactions.([]*tx.Transaction)
	txs := make([]*pb.Transaction, len(transactions))
	for i, tx := range transactions {
		txs[i] = s.txToProto(tx)
	}
	
	// Calculate height by finding block in chain
	height := int64(0)
	for i := 0; i <= s.bc.Height(); i++ {
		b, _ := s.bc.GetBlock(i)
		if b != nil && bytes.Equal(b.Hash, block.Hash) {
			height = int64(i)
			break
		}
	}
	
	return &pb.Block{
		Hash:         fmt.Sprintf("%x", block.Hash),
		Height:       height,
		PreviousHash: fmt.Sprintf("%x", block.Header.PrevBlockHash),
		Timestamp:    timestamppb.New(block.Header.Timestamp),
		Nonce:        int64(block.Header.Nonce),
		Difficulty:   int32(block.Header.DifficultyTarget),
		Transactions: txs,
		MerkleRoot:   fmt.Sprintf("%x", block.Header.MerkleRoot),
	}
}

func (s *Server) txToProto(transaction *tx.Transaction) *pb.Transaction {
	inputs := make([]*pb.TxInput, len(transaction.Inputs))
	for i, in := range transaction.Inputs {
		inputs[i] = &pb.TxInput{
			TxId:      fmt.Sprintf("%x", in.TxID),
			Vout:      int32(in.OutIndex),
			Signature: fmt.Sprintf("%x", in.Signature),
			PublicKey: fmt.Sprintf("%x", in.PubKey),
		}
	}
	
	outputs := make([]*pb.TxOutput, len(transaction.Outputs))
	for i, out := range transaction.Outputs {
		outputs[i] = &pb.TxOutput{
			Value:         int64(out.Value),
			PublicKeyHash: fmt.Sprintf("%x", out.PubKeyHash),
		}
	}
	
	return &pb.Transaction{
		Id:        fmt.Sprintf("%x", transaction.ID),
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: timestamppb.Now(),
	}
}

func (s *Server) protoToTx(pbTx *pb.Transaction) *tx.Transaction {
	inputs := make([]tx.TxInput, len(pbTx.Inputs))
	for i, in := range pbTx.Inputs {
		txID := []byte{}
		if len(in.TxId) > 0 {
			fmt.Sscanf(in.TxId, "%x", &txID)
		}
		signature := []byte{}
		if len(in.Signature) > 0 {
			fmt.Sscanf(in.Signature, "%x", &signature)
		}
		pubKey := []byte{}
		if len(in.PublicKey) > 0 {
			fmt.Sscanf(in.PublicKey, "%x", &pubKey)
		}
		inputs[i] = tx.TxInput{
			TxID:      txID,
			OutIndex:  int(in.Vout),
			Signature: signature,
			PubKey:    pubKey,
		}
	}
	
	outputs := make([]tx.TxOutput, len(pbTx.Outputs))
	for i, out := range pbTx.Outputs {
		pubKeyHash := []byte{}
		if len(out.PublicKeyHash) > 0 {
			fmt.Sscanf(out.PublicKeyHash, "%x", &pubKeyHash)
		}
		outputs[i] = tx.TxOutput{
			Value:      int64(out.Value),
			PubKeyHash: pubKeyHash,
		}
	}
	
	id := []byte{}
	if len(pbTx.Id) > 0 {
		fmt.Sscanf(pbTx.Id, "%x", &id)
	}
	
	return &tx.Transaction{
		ID:      id,
		Inputs:  inputs,
		Outputs: outputs,
	}
}

func (s *Server) getTotalTransactions() int64 {
	count := int64(0)
	height := s.bc.Height()
	
	for i := 0; i <= height; i++ {
		block, err := s.bc.GetBlock(i)
		if err != nil {
			continue
		}
		transactions := block.Transactions.([]*tx.Transaction)
		count += int64(len(transactions))
	}
	
	return count
}

func (s *Server) mineBlocks() {
	log.Printf("Mining started for address: %s", s.minerAddress)
	
	for {
		select {
		case <-s.stopMining:
			log.Println("Mining stopped")
			return
		default:
			// Get transactions from mempool
			s.mempoolMu.Lock()
			txs := make([]*tx.Transaction, len(s.mempool))
			copy(txs, s.mempool)
			s.mempool = s.mempool[:0] // Clear mempool
			s.mempoolMu.Unlock()
			
			// Mine new block
			block, err := s.bc.AddBlock(txs, s.minerAddress)
			if err != nil {
				log.Printf("Mining error: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			
			if block != nil {
				s.miningMu.Lock()
				s.blocksMined++
				s.miningMu.Unlock()
				
				// Find block height
				height := s.bc.Height()
				log.Printf("Mined block #%d with hash: %x", height, block.Hash)
				
				// Notify subscribers
				s.notifyBlockSubscribers(block)
			}
			
			time.Sleep(5 * time.Second) // Mining interval
		}
	}
}

func (s *Server) StopMiningInternal() {
	s.miningMu.Lock()
	defer s.miningMu.Unlock()
	
	if s.isMining {
		close(s.stopMining)
		s.isMining = false
	}
}

func (s *Server) notifyBlockSubscribers(block *types.Block) {
	s.subsMu.RLock()
	defer s.subsMu.RUnlock()
	
	for _, ch := range s.blockSubs {
		select {
		case ch <- block:
		default:
			// Skip if channel is full
		}
	}
}

func (s *Server) notifyTxSubscribers(transaction *tx.Transaction) {
	s.subsMu.RLock()
	defer s.subsMu.RUnlock()
	
	for _, ch := range s.txSubs {
		select {
		case ch <- transaction:
		default:
			// Skip if channel is full
		}
	}
}
