#!/bin/bash

# Test network transmission

PORT=7777

echo "=== TVCP Network Test ==="
echo "Testing localhost video streaming"
echo ""

# Start receiver
echo "Starting receiver on port $PORT..."
./bin/tvcp receive $PORT > /tmp/tvcp_recv.log 2>&1 &
RECV_PID=$!
sleep 1

# Start sender for 5 seconds
echo "Starting sender (5 seconds)..."
timeout 5 ./bin/tvcp send localhost:$PORT bounce > /tmp/tvcp_send.log 2>&1

# Stop receiver
echo "Stopping receiver..."
kill $RECV_PID 2>/dev/null
wait $RECV_PID 2>/dev/null

echo ""
echo "=== Sender Log ==="
tail -10 /tmp/tvcp_send.log

echo ""
echo "=== Receiver Log ==="
tail -10 /tmp/tvcp_recv.log

echo ""
echo "=== Test Complete ==="
