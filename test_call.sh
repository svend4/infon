#!/bin/bash

# Test two-way video call

echo "=== TVCP Two-Way Call Test ==="
echo "Testing bidirectional video communication"
echo ""

# Start first participant (Alice) on port 5000
echo "Starting Alice on port 5000..."
timeout 10 ./bin/tvcp call localhost:5001 gradient 5000 > /tmp/tvcp_alice.log 2>&1 &
ALICE_PID=$!
sleep 1

# Start second participant (Bob) on port 5001
echo "Starting Bob on port 5001..."
timeout 10 ./bin/tvcp call localhost:5000 bounce 5001 > /tmp/tvcp_bob.log 2>&1 &
BOB_PID=$!

echo ""
echo "Call in progress... (10 seconds)"
sleep 10

# Wait for processes
wait $ALICE_PID 2>/dev/null
wait $BOB_PID 2>/dev/null

echo ""
echo "=== Alice Log (last 15 lines) ==="
tail -15 /tmp/tvcp_alice.log

echo ""
echo "=== Bob Log (last 15 lines) ==="
tail -15 /tmp/tvcp_bob.log

echo ""
echo "=== Test Complete ==="
