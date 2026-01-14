#!/bin/bash

# Multi-Node Test Script
# This script starts multiple blockchain nodes and connects them

echo "🚀 Starting Multi-Node Blockchain Test"
echo "========================================"

# Clean up old data
echo "🗑️  Cleaning up old data..."
rm -rf ./node1.db ./node2.db ./node3.db
rm -f ./node*.log

# Build the P2P node
echo "🔨 Building P2P node..."
go build -o bin/node-p2p cmd/node-p2p/main.go
if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi
echo "✓ Build successful"

# Start Node 1 (Bootstrap node)
echo ""
echo "🟢 Starting Node 1 (Bootstrap)..."
./bin/node-p2p -db ./node1.db -listen "/ip4/0.0.0.0/tcp/9001" -fresh > node1.log 2>&1 &
NODE1_PID=$!
echo "   PID: $NODE1_PID"
sleep 3

# Extract Node 1's peer ID from log
NODE1_ADDR=$(grep "p2p/" node1.log | head -1 | awk '{print $NF}')
if [ -z "$NODE1_ADDR" ]; then
    echo "❌ Failed to get Node 1 address"
    kill $NODE1_PID
    exit 1
fi
echo "   Address: $NODE1_ADDR"

# Start Node 2 (connects to Node 1)
echo ""
echo "🟡 Starting Node 2..."
./bin/node-p2p -db ./node2.db -listen "/ip4/0.0.0.0/tcp/9002" -connect "$NODE1_ADDR" -fresh > node2.log 2>&1 &
NODE2_PID=$!
echo "   PID: $NODE2_PID"
sleep 3

# Start Node 3 (connects to Node 1)
echo ""
echo "🔵 Starting Node 3..."
./bin/node-p2p -db ./node3.db -listen "/ip4/0.0.0.0/tcp/9003" -connect "$NODE1_ADDR" -fresh > node3.log 2>&1 &
NODE3_PID=$!
echo "   PID: $NODE3_PID"
sleep 2

echo ""
echo "✓ All nodes started!"
echo ""
echo "📊 Node Status:"
echo "   Node 1 (Bootstrap): PID $NODE1_PID, Port 9001"
echo "   Node 2: PID $NODE2_PID, Port 9002"
echo "   Node 3: PID $NODE3_PID, Port 9003"
echo ""
echo "📝 Logs:"
echo "   tail -f node1.log  # Node 1"
echo "   tail -f node2.log  # Node 2"
echo "   tail -f node3.log  # Node 3"
echo ""
echo "🛑 Stop all nodes:"
echo "   kill $NODE1_PID $NODE2_PID $NODE3_PID"
echo ""
echo "⏸️  Nodes are running. Press Ctrl+C to stop all..."

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Stopping all nodes..."
    kill $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null
    wait $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null
    echo "✓ All nodes stopped"
    exit 0
}

# Set up signal handler
trap cleanup SIGINT SIGTERM

# Wait for all background processes
wait
