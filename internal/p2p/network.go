package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"

	"github.com/yourusername/bt/internal/blockchain"
	"github.com/yourusername/bt/internal/tx"
	"github.com/yourusername/bt/pkg/types"
)

const (
	// Protocol IDs
	BlockProtocol = "/btc/block/1.0.0"
	TxProtocol    = "/btc/tx/1.0.0"
	SyncProtocol  = "/btc/sync/1.0.0"
	PingProtocol  = "/btc/ping/1.0.0"
)

// MessageType represents the type of P2P message
type MessageType string

const (
	MsgTypeNewBlock     MessageType = "new_block"
	MsgTypeNewTx        MessageType = "new_tx"
	MsgTypeGetBlocks    MessageType = "get_blocks"
	MsgTypeBlocks       MessageType = "blocks"
	MsgTypeGetHeight    MessageType = "get_height"
	MsgTypeHeight       MessageType = "height"
	MsgTypePing         MessageType = "ping"
	MsgTypePong         MessageType = "pong"
)

// Message represents a P2P network message
type Message struct {
	Type      MessageType `json:"type"`
	Data      []byte      `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	From      string      `json:"from"`
}

// Network manages P2P networking for the blockchain
type Network struct {
	host       host.Host
	blockchain *blockchain.Blockchain
	ctx        context.Context
	cancel     context.CancelFunc

	// Peer management
	peers     map[peer.ID]bool
	peerMutex sync.RWMutex

	// Message handlers
	blockHandler func(*types.Block)
	txHandler    func(*tx.Transaction)

	// Synchronization
	syncMutex sync.Mutex
	syncing   bool
}

// NewNetwork creates a new P2P network
func NewNetwork(ctx context.Context, bc *blockchain.Blockchain, listenAddr string) (*Network, error) {
	// Parse listen address
	addr, err := multiaddr.NewMultiaddr(listenAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid listen address: %v", err)
	}

	// Create libp2p host
	h, err := libp2p.New(
		libp2p.ListenAddrs(addr),
		libp2p.NATPortMap(), // Enable NAT traversal
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %v", err)
	}

	netCtx, cancel := context.WithCancel(ctx)

	n := &Network{
		host:       h,
		blockchain: bc,
		ctx:        netCtx,
		cancel:     cancel,
		peers:      make(map[peer.ID]bool),
	}

	// Set up stream handlers
	h.SetStreamHandler(protocol.ID(BlockProtocol), n.handleBlockStream)
	h.SetStreamHandler(protocol.ID(TxProtocol), n.handleTxStream)
	h.SetStreamHandler(protocol.ID(SyncProtocol), n.handleSyncStream)
	h.SetStreamHandler(protocol.ID(PingProtocol), n.handlePingStream)

	return n, nil
}

// Start begins network operations
func (n *Network) Start() error {
	fmt.Printf("🌐 P2P Network started\n")
	fmt.Printf("   Node ID: %s\n", n.host.ID().String())
	fmt.Printf("   Addresses:\n")
	for _, addr := range n.host.Addrs() {
		fmt.Printf("      %s/p2p/%s\n", addr, n.host.ID().String())
	}
	return nil
}

// Stop gracefully shuts down the network
func (n *Network) Stop() error {
	n.cancel()
	return n.host.Close()
}

// ConnectToPeer connects to a peer using its multiaddr
func (n *Network) ConnectToPeer(peerAddr string) error {
	addr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %v", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %v", err)
	}

	if err := n.host.Connect(n.ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}

	n.peerMutex.Lock()
	n.peers[peerInfo.ID] = true
	n.peerMutex.Unlock()

	fmt.Printf("✓ Connected to peer: %s\n", peerInfo.ID.String())

	// Start synchronization with the new peer
	go n.syncWithPeer(peerInfo.ID)

	return nil
}

// BroadcastBlock broadcasts a new block to all peers
func (n *Network) BroadcastBlock(block *types.Block) {
	data, err := json.Marshal(block)
	if err != nil {
		fmt.Printf("Failed to marshal block: %v\n", err)
		return
	}

	msg := Message{
		Type:      MsgTypeNewBlock,
		Data:      data,
		Timestamp: time.Now(),
		From:      n.host.ID().String(),
	}

	n.broadcast(BlockProtocol, msg)
}

// BroadcastTransaction broadcasts a new transaction to all peers
func (n *Network) BroadcastTransaction(transaction *tx.Transaction) {
	data, err := json.Marshal(transaction)
	if err != nil {
		fmt.Printf("Failed to marshal transaction: %v\n", err)
		return
	}

	msg := Message{
		Type:      MsgTypeNewTx,
		Data:      data,
		Timestamp: time.Now(),
		From:      n.host.ID().String(),
	}

	n.broadcast(TxProtocol, msg)
}

// broadcast sends a message to all connected peers
func (n *Network) broadcast(proto string, msg Message) {
	n.peerMutex.RLock()
	peers := make([]peer.ID, 0, len(n.peers))
	for p := range n.peers {
		peers = append(peers, p)
	}
	n.peerMutex.RUnlock()

	for _, peerID := range peers {
		go n.sendMessage(peerID, proto, msg)
	}
}

// sendMessage sends a message to a specific peer
func (n *Network) sendMessage(peerID peer.ID, proto string, msg Message) error {
	stream, err := n.host.NewStream(n.ctx, peerID, protocol.ID(proto))
	if err != nil {
		return fmt.Errorf("failed to open stream: %v", err)
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// handleBlockStream handles incoming block messages
func (n *Network) handleBlockStream(stream network.Stream) {
	defer stream.Close()

	var msg Message
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		fmt.Printf("Failed to decode block message: %v\n", err)
		return
	}

	if msg.Type == MsgTypeNewBlock {
		var block types.Block
		if err := json.Unmarshal(msg.Data, &block); err != nil {
			fmt.Printf("Failed to unmarshal block: %v\n", err)
			return
		}

		// Process the received block
		n.processReceivedBlock(&block)
	}
}

// handleTxStream handles incoming transaction messages
func (n *Network) handleTxStream(stream network.Stream) {
	defer stream.Close()

	var msg Message
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		fmt.Printf("Failed to decode tx message: %v\n", err)
		return
	}

	if msg.Type == MsgTypeNewTx {
		var transaction tx.Transaction
		if err := json.Unmarshal(msg.Data, &transaction); err != nil {
			fmt.Printf("Failed to unmarshal transaction: %v\n", err)
			return
		}

		// Process the received transaction
		n.processReceivedTransaction(&transaction)
	}
}

// handleSyncStream handles blockchain synchronization requests
func (n *Network) handleSyncStream(stream network.Stream) {
	defer stream.Close()

	var msg Message
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		fmt.Printf("Failed to decode sync message: %v\n", err)
		return
	}

	encoder := json.NewEncoder(stream)

	switch msg.Type {
	case MsgTypeGetHeight:
		// Send our blockchain height
		response := Message{
			Type:      MsgTypeHeight,
			Data:      []byte(fmt.Sprintf("%d", n.blockchain.Height())),
			Timestamp: time.Now(),
			From:      n.host.ID().String(),
		}
		encoder.Encode(response)

	case MsgTypeGetBlocks:
		// Send requested blocks
		var startHeight int
		fmt.Sscanf(string(msg.Data), "%d", &startHeight)

		blocks := make([]*types.Block, 0)
		for i := startHeight; i < n.blockchain.Height(); i++ {
			block, err := n.blockchain.GetBlock(i)
			if err == nil {
				blocks = append(blocks, block)
			}
		}

		data, _ := json.Marshal(blocks)
		response := Message{
			Type:      MsgTypeBlocks,
			Data:      data,
			Timestamp: time.Now(),
			From:      n.host.ID().String(),
		}
		encoder.Encode(response)
	}
}

// handlePingStream handles ping/pong messages for peer liveness
func (n *Network) handlePingStream(stream network.Stream) {
	defer stream.Close()

	var msg Message
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		return
	}

	if msg.Type == MsgTypePing {
		encoder := json.NewEncoder(stream)
		response := Message{
			Type:      MsgTypePong,
			Timestamp: time.Now(),
			From:      n.host.ID().String(),
		}
		encoder.Encode(response)
	}
}

// syncWithPeer synchronizes blockchain with a peer
func (n *Network) syncWithPeer(peerID peer.ID) {
	n.syncMutex.Lock()
	if n.syncing {
		n.syncMutex.Unlock()
		return
	}
	n.syncing = true
	n.syncMutex.Unlock()

	defer func() {
		n.syncMutex.Lock()
		n.syncing = false
		n.syncMutex.Unlock()
	}()

	// Get peer's blockchain height
	stream, err := n.host.NewStream(n.ctx, peerID, protocol.ID(SyncProtocol))
	if err != nil {
		fmt.Printf("Failed to open sync stream: %v\n", err)
		return
	}
	defer stream.Close()

	// Request height
	msg := Message{
		Type:      MsgTypeGetHeight,
		Timestamp: time.Now(),
		From:      n.host.ID().String(),
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		fmt.Printf("Failed to request height: %v\n", err)
		return
	}

	// Read response
	var response Message
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&response); err != nil {
		fmt.Printf("Failed to read height response: %v\n", err)
		return
	}

	var peerHeight int
	fmt.Sscanf(string(response.Data), "%d", &peerHeight)

	ourHeight := n.blockchain.Height()

	if peerHeight > ourHeight {
		fmt.Printf("📥 Syncing blockchain (our: %d, peer: %d)\n", ourHeight, peerHeight)
		n.downloadBlocks(peerID, ourHeight)
	}
}

// downloadBlocks downloads missing blocks from a peer
func (n *Network) downloadBlocks(peerID peer.ID, startHeight int) {
	stream, err := n.host.NewStream(n.ctx, peerID, protocol.ID(SyncProtocol))
	if err != nil {
		fmt.Printf("Failed to open sync stream: %v\n", err)
		return
	}
	defer stream.Close()

	// Request blocks
	msg := Message{
		Type:      MsgTypeGetBlocks,
		Data:      []byte(fmt.Sprintf("%d", startHeight)),
		Timestamp: time.Now(),
		From:      n.host.ID().String(),
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		fmt.Printf("Failed to request blocks: %v\n", err)
		return
	}

	// Read response
	var response Message
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&response); err != nil {
		fmt.Printf("Failed to read blocks response: %v\n", err)
		return
	}

	if response.Type == MsgTypeBlocks {
		var blocks []*types.Block
		if err := json.Unmarshal(response.Data, &blocks); err != nil {
			fmt.Printf("Failed to unmarshal blocks: %v\n", err)
			return
		}

		// Add blocks to our blockchain
		for _, block := range blocks {
			n.processReceivedBlock(block)
		}

		fmt.Printf("✓ Synced %d blocks\n", len(blocks))
	}
}

// processReceivedBlock processes a block received from the network
func (n *Network) processReceivedBlock(block *types.Block) {
	// Check if we already have this block
	if _, err := n.blockchain.GetBlockByHash(block.Hash); err == nil {
		return // Already have it
	}

	// Validate the block
	if err := n.blockchain.ValidateBlock(block); err != nil {
		fmt.Printf("Received invalid block: %v\n", err)
		return
	}

	// Add to blockchain
	n.blockchain.Blocks = append(n.blockchain.Blocks, block)

	// Update UTXO set
	if txs, ok := block.Transactions.([]*tx.Transaction); ok {
		for _, transaction := range txs {
			n.blockchain.UTXOSet.Update(transaction)
		}
	}

	// Save to storage
	if n.blockchain.Storage != nil {
		if err := n.blockchain.Storage.SaveBlock(block); err != nil {
			fmt.Printf("Failed to save received block: %v\n", err)
		}
	}

	fmt.Printf("✓ Received and added block %x from network\n", block.Hash[:8])

	// Call custom handler if set
	if n.blockHandler != nil {
		n.blockHandler(block)
	}
}

// processReceivedTransaction processes a transaction received from the network
func (n *Network) processReceivedTransaction(transaction *tx.Transaction) {
	// Note: Full verification requires previous transactions
	// For now, we'll add to pending and let the blockchain validate
	// TODO: Implement transaction pool with full verification

	// Add to pending transactions
	n.blockchain.PendingTxs = append(n.blockchain.PendingTxs, transaction)

	fmt.Printf("✓ Received transaction %x from network\n", transaction.ID[:8])

	// Call custom handler if set
	if n.txHandler != nil {
		n.txHandler(transaction)
	}
}

// SetBlockHandler sets a custom handler for received blocks
func (n *Network) SetBlockHandler(handler func(*types.Block)) {
	n.blockHandler = handler
}

// SetTxHandler sets a custom handler for received transactions
func (n *Network) SetTxHandler(handler func(*tx.Transaction)) {
	n.txHandler = handler
}

// GetPeerCount returns the number of connected peers
func (n *Network) GetPeerCount() int {
	n.peerMutex.RLock()
	defer n.peerMutex.RUnlock()
	return len(n.peers)
}

// GetPeers returns a list of connected peer IDs
func (n *Network) GetPeers() []string {
	n.peerMutex.RLock()
	defer n.peerMutex.RUnlock()

	peers := make([]string, 0, len(n.peers))
	for p := range n.peers {
		peers = append(peers, p.String())
	}
	return peers
}
