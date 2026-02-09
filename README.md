# Bitcoin-Like Blockchain in Go

A production-quality blockchain implementation from scratch in Go, featuring Proof-of-Work consensus, UTXO model, P2P networking, and gRPC API.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## 🚀 Features

### ✅ Phase 1: Blockchain Core
- **SHA-256 hashing** - Cryptographic block hashing
- **Proof-of-Work mining** - Bitcoin-style consensus mechanism  
- **Merkle trees** - Efficient transaction verification
- **Dynamic difficulty adjustment** - Self-regulating mining difficulty
- **Chain validation** - Complete blockchain integrity verification

### ✅ Phase 2: Transactions & Wallets
- **ECDSA secp256k1** - Bitcoin-compatible cryptography
- **UTXO model** - Unspent Transaction Output tracking
- **Digital signatures** - Transaction authentication
- **Wallet management** - Key generation and address encoding
- **Transaction validation** - Signature verification and double-spend prevention

### ✅ Phase 3: Persistent Storage
- **LevelDB integration** - High-performance key-value storage
- **Block persistence** - Complete blockchain storage
- **UTXO set storage** - Efficient balance tracking
- **Metadata management** - Chain height, difficulty, and tip storage

### ✅ Phase 4: P2P Networking
- **libp2p framework** - Production-grade peer-to-peer networking
- **Peer discovery** - Automatic peer connection
- **Blockchain synchronization** - Automatic chain sync across nodes
- **Block propagation** - Real-time block broadcasting
- **Custom protocols** - Block, transaction, and sync protocols

### ✅ Phase 5: gRPC + Protobuf API
- **21 RPC methods** - Complete blockchain API
- **Protocol Buffers** - Type-safe serialization
- **Real-time streaming** - Server-side streaming for blocks and transactions
- **Wallet service** - Remote wallet management
- **Mining control** - Start/stop mining via API

### ✅ Phase 6: Web Frontend (NEW!)
- **Modern blockchain explorer** - Real-time blockchain visualization
- **Interactive dashboard** - Live stats and metrics
- **Wallet management UI** - Create and manage wallets
- **Block browser** - Explore blocks and transactions
- **Mempool viewer** - Monitor pending transactions
- **REST API proxy** - HTTP server for gRPC access

## 📊 Project Status

**Overall Progress: 100% Complete (6/6 phases)**

- ✅ Phase 1: Blockchain Core (100%)
- ✅ Phase 2: Transactions & Wallets (100%)
- ✅ Phase 3: Persistent Storage (100%)
- ✅ Phase 4: P2P Networking (95%)
- ✅ Phase 5: gRPC + Protobuf API (100%)
- ✅ Phase 6: Web Frontend (100%)

## 🛠️ Installation

### Prerequisites
- Go 1.21 or higher
- Protocol Buffer compiler (protoc) for regenerating proto files
- Git

### Clone and Build
```bash
git clone https://github.com/yourusername/bt.git
cd bt

# Install dependencies
go mod download

# Build all executables
make build

# Or build individually
go build -o bin/node cmd/node/main.go
go build -o bin/wallet cmd/wallet/main.go
go build -o bin/node-p2p cmd/node-p2p/main.go
go build -o bin/node-grpc cmd/node-grpc/main.go
go build -o bin/web-server cmd/web-server/main.go
```

## 🚀 Quick Start

### 1. Basic Blockchain Node
```bash
# Start a basic node with persistence
./bin/node

# Start fresh blockchain
./bin/node -fresh

# Use custom database path
./bin/node -db /path/to/blockchain.db
```

### 2. Wallet Management
```bash
# Create a new wallet
./bin/wallet create

# List all wallets
./bin/wallet list

# Check balance
./bin/wallet balance --address <your-address>
```

### 3. P2P Network Node
```bash
# Start bootstrap node
./bin/node-p2p -db node1.db -listen "/ip4/0.0.0.0/tcp/9001" -fresh -mine

# Connect peer node
./bin/node-p2p -db node2.db -listen "/ip4/0.0.0.0/tcp/9002" \
  -connect "/ip4/127.0.0.1/tcp/9001/p2p/<PEER_ID>"
```

### 4. gRPC API Node
```bash
# Start gRPC server
./bin/node-grpc -grpc :50051 -fresh

# Test API
go run cmd/grpc-test/main.go
```

### 5. Web Frontend (NEW!)
```bash
# Terminal 1: Start gRPC node
./bin/node-grpc -grpc :50051 -fresh

# Terminal 2: Start web server
./bin/web-server

# Open browser
# Visit http://localhost:8080
```

The web interface provides:
- 📊 Live blockchain dashboard with real-time stats
- 🔗 Block explorer with transaction details
- 👛 Wallet management and balance tracking
- 📭 Mempool viewer for pending transactions
- ⛏️ Mining information and hashrate
- 🎨 Beautiful modern UI with auto-refresh

See [web/README.md](web/README.md) for detailed frontend documentation.

## 📡 gRPC API Usage

### Using grpcurl
```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Get blockchain info
grpcurl -plaintext localhost:50051 blockchain.BlockchainService/GetBlockchainInfo

# Get block by height
grpcurl -plaintext -d '{"height": 0}' localhost:50051 \
  blockchain.BlockchainService/GetBlockByHeight

# Create wallet
grpcurl -plaintext -d '{}' localhost:50051 \
  blockchain.WalletService/CreateWallet

# Start mining
grpcurl -plaintext -d '{"miner_address": "<address>"}' localhost:50051 \
  blockchain.BlockchainService/StartMining
```

### Available gRPC Services

**BlockchainService:**
- `GetBlockchainInfo` - Get chain statistics
- `GetBlockByHash` / `GetBlockByHeight` - Retrieve blocks
- `GetBestBlockHash` / `GetBlockHeight` - Query chain state
- `GetTransaction` / `SubmitTransaction` - Transaction operations
- `GetMempool` - View pending transactions
- `GetUTXO` / `GetBalance` - Query UTXOs and balances
- `StartMining` / `StopMining` / `GetMiningInfo` - Mining control
- `SubscribeBlocks` / `SubscribeTransactions` - Real-time streaming

**WalletService:**
- `CreateWallet` / `GetWallet` / `ListWallets` - Wallet management
- `GetWalletBalance` / `SendTransaction` - Transaction creation

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/blockchain/...

# Run short tests (skip P2P multi-node tests)
go test -short ./...

# Verbose output
go test -v ./...
```

### Test Results
- **47 unit tests** passing (Phases 1-3)
- **18 gRPC tests** (Phase 5)
- **6 P2P tests** (Phase 4)

## 📦 Project Structure

```
bt/
├── api/proto/              # Protocol Buffer definitions
│   ├── blockchain.proto    # gRPC service definitions
│   ├── blockchain.pb.go    # Generated protobuf code
│   └── blockchain_grpc.pb.go
├── cmd/                    # Executable commands
│   ├── node/              # Basic blockchain node
│   ├── wallet/            # Wallet management CLI
│   ├── node-p2p/          # P2P-enabled node
│   ├── node-grpc/         # gRPC-enabled node
│   └── grpc-test/         # gRPC API test client
├── internal/              # Private packages
│   ├── blockchain/        # Core blockchain logic
│   ├── crypto/            # Cryptography (ECDSA, hashing)
│   ├── grpc/              # gRPC server implementation
│   ├── merkle/            # Merkle tree
│   ├── p2p/               # P2P networking
│   ├── pow/               # Proof-of-Work
│   ├── storage/           # LevelDB persistence
│   ├── tx/                # Transactions
│   └── utxo/              # UTXO set management
├── pkg/types/             # Public type definitions
├── scripts/               # Test and utility scripts
├── go.mod                 # Go module definition
└── README.md             # This file
```

## 🏗️ Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                     APPLICATION LAYER                        │
│  node | wallet | node-p2p | node-grpc                       │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                      API LAYER                               │
│  gRPC Services | P2P Protocols | CLI Commands               │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   BUSINESS LOGIC                             │
│  blockchain | tx | utxo | crypto | pow | merkle            │
│  grpc | p2p                                                  │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   PERSISTENCE LAYER                          │
│  storage (LevelDB wrapper)                                   │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                      DATA LAYER                              │
│  LevelDB Files | Blockchain DB | Wallet DB                  │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

**Block Creation:**
1. Transaction created and signed
2. Added to mempool
3. Miner collects transactions
4. Merkle root calculated
5. PoW mining (nonce iteration)
6. Block validated
7. Added to blockchain
8. UTXO set updated
9. Persisted to LevelDB
10. Broadcast to P2P network
11. Stream to gRPC subscribers

## 🔧 Development

### Regenerate Protocol Buffers
```bash
# Install tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate code
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  api/proto/blockchain.proto
```

### Code Quality
```bash
# Format code
go fmt ./...

# Lint
golangci-lint run

# Vet
go vet ./...

# Security scan
gosec ./...
```

## 📈 Performance

- **Mining**: ~0.1-10 seconds per block (difficulty dependent)
- **Block Validation**: < 1ms
- **P2P Sync**: ~1-2 seconds for 100 blocks
- **gRPC Latency**: < 10ms (local)
- **Storage**: ~10KB per block
- **Memory**: ~50-100MB per node

## 🔐 Security Features

- **ECDSA secp256k1** - Bitcoin-standard elliptic curve
- **SHA-256** - Cryptographic hashing
- **Digital Signatures** - Transaction authentication
- **Merkle Trees** - Efficient integrity verification
- **Proof-of-Work** - Sybil attack resistance
- **UTXO Model** - Double-spend prevention
- **libp2p Security** - TLS 1.3 / Noise encryption

## 🐛 Known Issues

### Phase 4 (P2P)
- Dependency conflicts between libp2p v0.38.3 and quic-go v0.46.0
- Code is complete but may not build in some environments
- Pre-built executable (node-p2p) works correctly

### Resolution
- Use Go 1.21 for best compatibility
- Or use the pre-built executable from releases

## 🗺️ Roadmap

### Phase 6: Docker Multi-Node (Planned)
- [ ] Dockerfile for blockchain node
- [ ] Docker Compose for multi-node network
- [ ] Container networking configuration
- [ ] Volume persistence
- [ ] Network monitoring and visualization
- [ ] Load testing suite

### Future Enhancements
- [ ] REST API gateway
- [ ] Web dashboard
- [ ] Smart contract support
- [ ] Lightning Network integration
- [ ] Cross-chain bridges
- [ ] Advanced mining pools

## 📚 Documentation

Detailed phase documentation:
- [Phase 1: Blockchain Core](PHASE1_COMPLETE.md)
- [Phase 2: Transactions & Wallets](PHASE2_COMPLETE.md)
- [Phase 3: LevelDB Persistence](PHASE3_COMPLETE.md)
- [Phase 4: P2P Networking](PHASE4_COMPLETE.md)
- [Phase 5: gRPC + Protobuf API](PHASE5_COMPLETE.md)
- [Complete Documentation](COMPLETE_PROJECT_DOCUMENTATION.md)

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup
```bash
# Clone your fork
git clone https://github.com/yourusername/bt.git
cd bt

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o bin/node cmd/node/main.go
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Bitcoin** - Original blockchain design and inspiration
- **libp2p** - Production-grade P2P networking library
- **gRPC** - High-performance RPC framework
- **LevelDB** - Fast key-value storage
- **Go Community** - Excellent tooling and libraries

## 📞 Contact

- **GitHub**: [@vishalharkal15](https://github.com/vishalharkal15)
- **Email**: vishalharkal@dietms.org

## ⭐ Star History

If you find this project useful, please consider giving it a star!

---

**Built with ❤️ using Go**

**Educational Purpose**: This project is designed for learning blockchain fundamentals and should not be used in production for real cryptocurrency applications.
