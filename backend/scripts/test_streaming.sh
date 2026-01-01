#!/bin/bash

# Test streaming reliability with comprehensive diagnostics
# This script tests the full pipeline: ingest -> buffer -> storage -> stream

set -e

SERVER_URL="http://localhost:8080"
WS_URL="ws://localhost:8080"
LOG_DIR="/tmp/logpulse-test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}  LogPulse Streaming Reliability Test${NC}"
echo -e "${BLUE}=========================================${NC}\n"

# Function to check if server is running
check_server() {
    echo -e "${YELLOW}[1/5] Checking if LogPulse server is running...${NC}"
    if curl -s "$SERVER_URL/health" > /dev/null; then
        echo -e "${GREEN}✓ Server is running${NC}\n"
    else
        echo -e "${RED}✗ Server is not running on $SERVER_URL${NC}"
        echo "Start the server with: cd backend && ./logpulse"
        exit 1
    fi
}

# Function to get health metrics
get_metrics() {
    curl -s "$SERVER_URL/health" | python3 -m json.tool
}

# Function to get Prometheus metrics
get_prometheus_metrics() {
    echo -e "${YELLOW}[5/5] Checking Prometheus metrics...${NC}"
    metrics=$(curl -s "$SERVER_URL/metrics")
    
    echo "Key streaming metrics:"
    echo "$metrics" | grep "logpulse_" | grep -E "(broadcasted|dropped|stream_clients|ingested)" || echo "No streaming metrics found"
    echo ""
}

# Function to test ingestion
test_ingestion() {
    echo -e "${YELLOW}[2/5] Testing log ingestion with multiple streams...${NC}"
    
    # Generate test logs with different labels
    for i in {1..5}; do
        PAYLOAD=$(cat <<EOF
{
  "streams": [
    {
      "labels": {
        "service": "api-gateway",
        "env": "prod",
        "level": "info"
      },
      "entries": [
        {
          "ts": "$(date -u +'%Y-%m-%dT%H:%M:%SZ')",
          "line": "Request processed successfully [Request #$i]"
        }
      ]
    },
    {
      "labels": {
        "service": "database",
        "env": "prod",
        "level": "debug"
      },
      "entries": [
        {
          "ts": "$(date -u +'%Y-%m-%dT%H:%M:%SZ')",
          "line": "Query executed in 45ms [Query #$i]"
        }
      ]
    },
    {
      "labels": {
        "service": "api-gateway",
        "env": "prod",
        "level": "error"
      },
      "entries": [
        {
          "ts": "$(date -u +'%Y-%m-%dT%H:%M:%SZ')",
          "line": "Connection timeout to cache [Error #$i]"
        }
      ]
    }
  ]
}
EOF
)
        
        RESPONSE=$(curl -s -X POST "$SERVER_URL/ingest" \
            -H "Content-Type: application/json" \
            -d "$PAYLOAD")
        
        ACCEPTED=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['accepted'])" 2>/dev/null || echo "0")
        echo "  Batch $i: Ingested $ACCEPTED entries"
        sleep 0.5
    done
    echo -e "${GREEN}✓ Ingestion test completed${NC}\n"
}

# Function to test WebSocket streaming
test_websocket_streaming() {
    echo -e "${YELLOW}[3/5] Testing WebSocket streaming with filters...${NC}"
    
    # Create a test client script
    TEST_CLIENT=$(cat <<'SCRIPT'
#!/usr/bin/env python3
import asyncio
import json
import sys
import time

try:
    import websockets
except ImportError:
    print("ERROR: websockets library not found. Install with: pip install websockets")
    sys.exit(1)

async def test_stream(filter_labels, duration=10):
    """Connect to stream and receive logs with timeout"""
    url = "ws://localhost:8080/stream"
    
    # Build URL with query params
    query_parts = ["{}".format(k) + "=" + v for k, v in filter_labels.items()]
    if query_parts:
        url += "?" + "&".join(query_parts)
    
    print(f"[WS] Connecting to {url}")
    
    received = []
    try:
        async with websockets.connect(url, ping_interval=20, ping_timeout=10) as websocket:
            print(f"[WS] Connected! Listening for {duration} seconds...")
            start = time.time()
            
            while time.time() - start < duration:
                try:
                    msg = await asyncio.wait_for(websocket.recv(), timeout=duration)
                    data = json.loads(msg)
                    
                    if data.get('type') == 'connected':
                        print(f"[WS] Connected message: {data.get('message')}")
                    elif data.get('type') == 'log':
                        log_data = data.get('data', {})
                        print(f"[WS] ✓ Received log: service={log_data.get('labels', {}).get('service')}, msg={log_data.get('message')[:50]}")
                        received.append(data)
                except asyncio.TimeoutError:
                    break
    except Exception as e:
        print(f"[WS] Error: {e}")
        return []
    
    print(f"[WS] Received {len(received)} logs total\n")
    return received

async def main():
    print("Testing WebSocket streaming...\n")
    
    # Test 1: No filter (receive all logs)
    print("[Test 1] No filter - should receive all logs:")
    logs = await test_stream({}, duration=15)
    if logs:
        print(f"  ✓ Received {len(logs)} logs\n")
    else:
        print("  ✗ No logs received\n")
    
    # Test 2: Filter by service
    print("[Test 2] Filter by service=api-gateway:")
    logs = await test_stream({"service": "api-gateway"}, duration=15)
    if logs:
        print(f"  ✓ Received {len(logs)} api-gateway logs\n")
    else:
        print("  ✗ No api-gateway logs received\n")
    
    # Test 3: Filter by level
    print("[Test 3] Filter by level=error:")
    logs = await test_stream({"level": "error"}, duration=15)
    if logs:
        print(f"  ✓ Received {len(logs)} error logs\n")
    else:
        print("  ✗ No error logs received\n")

if __name__ == "__main__":
    asyncio.run(main())
SCRIPT
)
    
    echo "$TEST_CLIENT" > /tmp/ws_test.py
    python3 /tmp/ws_test.py || echo -e "${YELLOW}Note: WebSocket test requires Python websockets library${NC}"
    echo ""
}

# Function to test query endpoint
test_query() {
    echo -e "${YELLOW}[4/5] Testing query endpoint...${NC}"
    
    # Query logs
    RESPONSE=$(curl -s "$SERVER_URL/query?query=%7Bservice%3D%22api-gateway%22%7D&limit=10")
    COUNT=$(echo "$RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(len(data.get('entries', [])))" 2>/dev/null || echo "0")
    
    echo "  Queried logs with filter service=api-gateway"
    echo "  Found: $COUNT entries"
    echo -e "${GREEN}✓ Query test completed${NC}\n"
}

# Main execution
check_server
sleep 1
test_ingestion
sleep 2
test_websocket_streaming
test_query
get_prometheus_metrics

echo -e "${BLUE}=========================================${NC}"
echo -e "${GREEN}Streaming Test Complete!${NC}"
echo -e "${BLUE}=========================================${NC}\n"

echo "Summary of stream metrics:"
get_metrics
