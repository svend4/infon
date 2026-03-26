#!/bin/bash
#
# Test script for packet loss recovery
# This script simulates packet loss using Linux tc (traffic control)
#

set -e

echo "=== TVCP Loss Recovery Test ==="
echo ""

# Check if running as root (needed for tc)
if [ "$EUID" -eq 0 ]; then
    USE_TC=true
    LOSS_RATE="10%"
    echo "Running with packet loss simulation: $LOSS_RATE"
    echo ""

    # Add packet loss to loopback interface
    tc qdisc add dev lo root netem loss $LOSS_RATE 2>/dev/null || true

    # Cleanup function
    cleanup() {
        echo ""
        echo "Cleaning up network emulation..."
        tc qdisc del dev lo root 2>/dev/null || true
    }
    trap cleanup EXIT
else
    USE_TC=false
    echo "Running without packet loss simulation (need root for tc)"
    echo "To enable packet loss simulation, run: sudo $0"
    echo ""
fi

# Build if needed
if [ ! -f "./bin/tvcp" ]; then
    echo "Building TVCP..."
    make build
    echo ""
fi

echo "Starting test call (10 seconds)..."
echo ""

# Start Alice
timeout 10 ./bin/tvcp call localhost:5001 gradient 5000 > /tmp/alice_loss.log 2>&1 &
ALICE_PID=$!

# Give Alice time to start
sleep 1

# Start Bob
timeout 10 ./bin/tvcp call localhost:5000 bounce 5001 > /tmp/bob_loss.log 2>&1 &
BOB_PID=$!

# Wait for both to finish
wait $ALICE_PID $BOB_PID 2>/dev/null || true

echo ""
echo "=== Test Results ==="
echo ""

# Extract statistics from logs
echo "Alice Statistics:"
grep -E "(Sent:|Received:|Packets|Lost|Retrans|NACK)" /tmp/alice_loss.log | tail -10
echo ""

echo "Bob Statistics:"
grep -E "(Sent:|Received:|Packets|Lost|Retrans|NACK)" /tmp/bob_loss.log | tail -10
echo ""

if [ "$USE_TC" = true ]; then
    echo "✓ Test completed with $LOSS_RATE packet loss simulation"
else
    echo "✓ Test completed without packet loss simulation"
fi
