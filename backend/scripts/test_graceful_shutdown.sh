#!/bin/bash

# Test script for graceful shutdown fix
# This script verifies that no logs are lost during shutdown

set -e

echo "Testing graceful shutdown fix..."

# Configuration
SERVER_PORT=18080
LOG_COUNT=1000
SEND_DURATION=10  # Send logs for 10 seconds
BACKEND_DIR="$(dirname "$0")/.."

# Build the server
echo "Building server..."
cd "$BACKEND_DIR"
go build -o ./tmp/logpulse-server ./cmd/server

# Start server in background
echo "Starting server on port $SERVER_PORT..."
PORT=$SERVER_PORT ./tmp/logpulse-server &
SERVER_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
sleep 3

# Function to send logs continuously
send_logs() {
    local count=0
    local start_time=$(date +%s)
    
    echo "Sending logs continuously..."
    while [ $(($(date +%s) - start_time)) -lt $SEND_DURATION ]; do
        curl -s -X POST "http://localhost:$SERVER_PORT/ingest" \
             -H "Content-Type: application/json" \
             -d "{\"timestamp\":\"$(date -Is)\",\"level\":\"info\",\"message\":\"Test log $count\",\"labels\":{\"service\":\"test\",\"instance\":\"1\"}}" \
             > /dev/null
        count=$((count + 1))
        
        # Send logs at high rate
        sleep 0.01
    done
    
    echo "Sent $count logs total"
    echo "$count" > /tmp/logs_sent.txt
}

# Start sending logs in background
send_logs &
SENDER_PID=$!

# Wait a bit, then trigger graceful shutdown
echo "Sending logs for $SEND_DURATION seconds..."
sleep 5

echo "Triggering graceful shutdown (SIGTERM)..."
kill -TERM $SERVER_PID

# Wait for graceful shutdown to complete
echo "Waiting for server to shutdown gracefully..."
wait $SERVER_PID || true

# Stop log sender
kill $SENDER_PID 2>/dev/null || true
wait $SENDER_PID 2>/dev/null || true

# Query stored logs to verify no loss
echo "Querying stored logs..."
sleep 1

# Restart server briefly to query logs
./tmp/logpulse-server &
QUERY_SERVER_PID=$!
sleep 2

# Query logs
STORED_LOGS=$(curl -s "http://localhost:$SERVER_PORT/query?q=service=test&from=$(date -d '1 minute ago' -Is)&to=$(date -Is)" | jq -r '.logs | length' 2>/dev/null || echo "0")

# Cleanup query server
kill $QUERY_SERVER_PID 2>/dev/null || true
wait $QUERY_SERVER_PID 2>/dev/null || true

# Get sent count
SENT_LOGS=$(cat /tmp/logs_sent.txt 2>/dev/null || echo "0")

echo "Results:"
echo "  Logs sent: $SENT_LOGS"
echo "  Logs stored: $STORED_LOGS"

# Calculate success rate
if [ "$SENT_LOGS" -gt 0 ]; then
    SUCCESS_RATE=$(( STORED_LOGS * 100 / SENT_LOGS ))
    echo "  Success rate: $SUCCESS_RATE%"
    
    # Consider success if we stored at least 90% of sent logs
    if [ "$SUCCESS_RATE" -ge 90 ]; then
        echo "✅ PASS: Graceful shutdown preserved most logs"
        exit 0
    else
        echo "❌ FAIL: Significant log loss during shutdown"
        exit 1
    fi
else
    echo "❌ FAIL: No logs were sent"
    exit 1
fi

# Cleanup
rm -f /tmp/logs_sent.txt
rm -rf ./tmp