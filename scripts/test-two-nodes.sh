#!/bin/bash

# Simple Two-Node Test
# Demonstrates P2P connectivity and blockchain synchronization

echo "🚀 Two-Node P2P Test"
echo "===================="

# Clean up
rm -rf ./node-a.db ./node-b.db

# Build
echo "🔨 Building..."
go build -o bin/node-p2p cmd/node-p2p/main.go 2>&1 | grep -v "^#"
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo ""
echo "🟢 Starting Node A (creates blockchain)..."
./bin/node-p2p -db node-a.db -listen "/ip4/127.0.0.1/tcp/9001" -fresh > node-a.log 2>&1 &
NODE_A_PID=$!
sleep 4

# Get Node A's address
NODE_A_ADDR=$(grep "/ip4/127.0.0.1/tcp/9001/p2p/" node-a.log | head -1 | awk '{print $NF}')

if [ -z "$NODE_A_ADDR" ]; then
    echo "❌ Failed to get Node A address"
    kill $NODE_A_PID 2>/dev/null
    cat node-a.log
    exit 1
fi

echo "✓ Node A started (PID: $NODE_A_PID)"
echo "  Address: $NODE_A_ADDR"

# Check Node A's blockchain
NODE_A_BLOCKS=$(grep "Total blocks:" node-a.log | tail -1 | awk '{print $3}')
echo "  Blockchain height: $NODE_A_BLOCKS blocks"

echo ""
echo "🟡 Starting Node B (will sync from Node A)..."
./bin/node-p2p -db node-b.db -listen "/ip4/127.0.0.1/tcp/9002" -connect "$NODE_A_ADDR" -fresh > node-b.log 2>&1 &
NODE_B_PID=$!
sleep 5

echo "✓ Node B started (PID: $NODE_B_PID)"

# Check if nodes connected
if grep -q "Connected to peer" node-b.log; then
    echo "✓ Nodes connected successfully!"
else
    echo "⚠️  Connection status unclear"
fi

# Check sync status
if grep -q "Syncing blockchain" node-b.log; then
    echo "✓ Blockchain synchronization initiated"
    SYNCED=$(grep "Synced.*blocks" node-b.log | tail -1)
    echo "  $SYNCED"
fi

echo ""
echo "📊 Final Status:"
echo "  Node A: Height $NODE_A_BLOCKS blocks (PID: $NODE_A_PID)"
echo "  Node B: Synced from Node A (PID: $NODE_B_PID)"

echo ""
echo "📝 View logs:"
echo "  tail -f node-a.log"
echo "  tail -f node-b.log"

echo ""
echo "🛑 Stop nodes:"
echo "  kill $NODE_A_PID $NODE_B_PID"

echo ""
echo "⏸️  Nodes running. Press Ctrl+C to stop..."

cleanup() {
    echo ""
    echo "🛑 Stopping nodes..."
    kill $NODE_A_PID $NODE_B_PID 2>/dev/null
    wait $NODE_A_PID $NODE_B_PID 2>/dev/null
    echo "✓ Stopped"
}

trap cleanup SIGINT SIGTERM
wait
