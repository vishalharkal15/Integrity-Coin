# Integrity Coin - Web Frontend

A modern web-based blockchain explorer for Integrity Coin cryptocurrency.

## Features

- 📊 **Real-time Dashboard** - Live blockchain statistics and metrics
- 🔗 **Block Explorer** - Browse recent blocks with detailed information
- 👛 **Wallet Management** - Create and manage wallets with balance tracking
- 📭 **Mempool Viewer** - Monitor pending transactions
- ⛏️ **Mining Information** - Track mining difficulty and network hashrate
- 🎨 **Modern UI** - Beautiful gradient design with smooth animations
- 🔄 **Auto-refresh** - Updates every 10 seconds automatically

## Architecture

The frontend consists of three main components:

### 1. Static Frontend (`web/static/`)
- `index.html` - Main HTML structure with tabs and sections
- `css/style.css` - Modern CSS with gradient backgrounds and animations
- `js/api.js` - API client for backend communication
- `js/app.js` - Main application logic and UI updates

### 2. Backend Proxy (`cmd/web-server/`)
- Go HTTP server that serves static files
- REST API proxy to gRPC blockchain node
- CORS-enabled endpoints for frontend access

### 3. gRPC Node (`cmd/node-grpc/`)
- Full blockchain node with gRPC API
- 21 RPC methods for blockchain operations
- Required to be running for frontend to work

## Quick Start

### 1. Start the gRPC Node

```bash
# Start with fresh blockchain
./bin/node-grpc -grpc :50051 -fresh

# Or use existing blockchain
./bin/node-grpc -grpc :50051
```

### 2. Start the Web Server

```bash
# In a new terminal
./bin/web-server
```

### 3. Open in Browser

Visit: **http://localhost:8080**

## Building

```bash
# Build all components
make build

# Or build individually
make node-grpc
make web-server
```

## API Endpoints

The web server exposes these REST endpoints:

### Blockchain
- `GET /api/health` - Health check
- `GET /api/blockchain/info` - Blockchain information
- `GET /api/blockchain/blocks?count=10` - Recent blocks
- `GET /api/blockchain/block/:height` - Specific block
- `GET /api/blockchain/height` - Current height

### Wallets
- `GET /api/wallet/list` - List all wallets
- `POST /api/wallet/create` - Create new wallet

### Mempool & Mining
- `GET /api/mempool` - Get mempool transactions
- `GET /api/mining/info` - Get mining information

## Technology Stack

**Frontend:**
- Pure HTML5, CSS3, JavaScript (no frameworks)
- Modern CSS Grid and Flexbox layouts
- Fetch API for HTTP requests
- Google Fonts (Inter)

**Backend:**
- Go 1.23+
- gRPC client
- net/http standard library
- Protocol Buffers

**Dependencies:**
- gRPC blockchain node (port 50051)
- Modern web browser with JavaScript enabled

## Features in Detail

### Dashboard Stats
- Block height with live updates
- Current difficulty
- Mempool transaction count  
- Estimated network hashrate

### Block Explorer
- Recent blocks list (configurable count)
- Block hash, height, timestamp
- Previous block hash
- Nonce value
- Transaction count per block

### Wallet Manager
- Create new wallets with one click
- View all wallet addresses
- Check balance for each wallet
- Copy-friendly address display

### Mempool Monitor
- View pending transactions
- Transaction ID display
- Input/output counts
- Total transaction value

### Mining Dashboard
- Current difficulty
- Network hashrate estimate
- Mining status (active/inactive)
- Block reward information

## Configuration

Edit `cmd/web-server/main.go` to change:

```go
const (
    grpcAddress = "localhost:50051"  // gRPC node address
    httpPort    = ":8080"             // Web server port
)
```

## Development

### File Structure

```
web/
├── static/
│   ├── index.html          # Main HTML page
│   ├── css/
│   │   └── style.css       # Stylesheet with animations
│   └── js/
│       ├── api.js          # API client wrapper
│       └── app.js          # Application logic
│
cmd/
└── web-server/
    └── main.go             # HTTP server + gRPC proxy
```

### Adding New Features

1. Add gRPC method call in `cmd/web-server/main.go`
2. Create REST endpoint handler
3. Update `js/api.js` with new API method
4. Add UI components in `index.html`
5. Style with CSS in `style.css`
6. Implement logic in `app.js`

## Troubleshooting

**"Connection failed" error:**
- Ensure gRPC node is running on port 50051
- Check `./bin/node-grpc -grpc :50051` is active

**"Cannot connect to node" message:**
- Verify web server is running: `./bin/web-server`
- Check no firewall blocking port 8080

**Blank dashboard:**
- Open browser console (F12) for errors
- Verify gRPC node has blockchain data

**API errors:**
- Check gRPC node logs for errors
- Verify protobuf definitions match

## Performance

- Frontend size: ~50KB total (HTML + CSS + JS)
- Auto-refresh interval: 10 seconds
- Typical API response: <100ms
- Browser memory: ~20-30MB

## Browser Compatibility

- Chrome 90+ ✅
- Firefox 88+ ✅
- Safari 14+ ✅
- Edge 90+ ✅

## Security Notes

- Frontend connects to localhost by default
- CORS enabled for development
- Private keys never transmitted
- For production: Add authentication, HTTPS, rate limiting

## License

MIT License - See main project LICENSE file

## Contributing

1. Fork the repository
2. Create feature branch
3. Make changes and test
4. Submit pull request

For issues and suggestions, open an issue on GitHub.
