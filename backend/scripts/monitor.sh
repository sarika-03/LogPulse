#!/bin/bash

# LogPulse Monitoring & Health Check Script

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SERVER_URL="${SERVER_URL:-http://localhost:8080}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"

# Function to display header
show_header() {
    clear
    echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║           LogPulse System Monitor                     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}\n"
    echo -e "Monitoring: ${BLUE}$SERVER_URL${NC}"
    echo -e "Updated: $(date '+%Y-%m-%d %H:%M:%S')\n"
}

# Function to check service health
check_health() {
    echo -e "${YELLOW}=== Service Health ===${NC}"
    
    # LogPulse
    if curl -s "$SERVER_URL/health" > /dev/null 2>&1; then
        echo -e "  LogPulse:      ${GREEN}✓ Running${NC}"
        
        HEALTH=$(curl -s "$SERVER_URL/health" | python3 -c "import sys, json; data=json.load(sys.stdin); print('Status: {}, Rate: {}/s, Storage: {} MB, Clients: {}'.format(data.get('status'), data.get('ingestionRate'), data.get('storageUsed', 0)//1048576, data.get('streamClients')))")
        echo -e "                 $HEALTH"
    else
        echo -e "  LogPulse:      ${RED}✗ Down${NC}"
    fi
    
    # Prometheus
    if curl -s "$PROMETHEUS_URL/-/ready" > /dev/null 2>&1; then
        echo -e "  Prometheus:    ${GREEN}✓ Running${NC}"
    else
        echo -e "  Prometheus:    ${RED}✗ Down${NC}"
    fi
    
    # Grafana
    if curl -s "http://localhost:3000/api/health" > /dev/null 2>&1; then
        echo -e "  Grafana:       ${GREEN}✓ Running${NC}"
    else
        echo -e "  Grafana:       ${RED}✗ Down${NC}"
    fi
    
    echo ""
}

# Function to show metrics
show_metrics() {
    echo -e "${YELLOW}=== Key Metrics ===${NC}"
    
    METRICS=$(curl -s "$SERVER_URL/metrics")
    
    # Ingestion metrics
    INGESTED_LINES=$(echo "$METRICS" | grep "lokiclone_ingested_lines_total" | grep -v "#" | awk '{print $2}')
    INGESTED_BYTES=$(echo "$METRICS" | grep "lokiclone_ingested_bytes_total" | grep -v "#" | awk '{print $2}')
    
    # Streaming metrics
    BROADCASTED=$(echo "$METRICS" | grep "lokiclone_broadcasted_lines_total" | grep -v "#" | awk '{print $2}')
    DROPPED=$(echo "$METRICS" | grep "lokiclone_dropped_broadcasts_total" | grep -v "#" | awk '{print $2}')
    CLIENTS=$(echo "$METRICS" | grep "lokiclone_stream_clients_connected" | grep -v "#" | awk '{print $2}')
    
    # Storage metrics
    CHUNKS=$(echo "$METRICS" | grep "lokiclone_chunks_stored_total" | grep -v "#" | awk '{print $2}')
    STORAGE=$(echo "$METRICS" | grep "lokiclone_storage_bytes" | grep -v "#" | awk '{print $2}')
    
    # Calculate rates
    if [ -f /tmp/logpulse_prev_metrics ]; then
        PREV_LINES=$(cat /tmp/logpulse_prev_metrics | grep "lines:" | cut -d: -f2)
        PREV_TIME=$(cat /tmp/logpulse_prev_metrics | grep "time:" | cut -d: -f2)
        CURR_TIME=$(date +%s)
        
        TIME_DIFF=$((CURR_TIME - PREV_TIME))
        if [ $TIME_DIFF -gt 0 ]; then
            LINE_DIFF=$((INGESTED_LINES - PREV_LINES))
            RATE=$((LINE_DIFF / TIME_DIFF))
        else
            RATE=0
        fi
    else
        RATE=0
    fi
    
    # Save current metrics
    echo "lines:$INGESTED_LINES" > /tmp/logpulse_prev_metrics
    echo "time:$(date +%s)" >> /tmp/logpulse_prev_metrics
    
    echo -e "  Ingestion:"
    echo -e "    Total Lines:     $(printf "%'d" ${INGESTED_LINES:-0})"
    echo -e "    Total Bytes:     $(numfmt --to=iec-i --suffix=B ${INGESTED_BYTES:-0} 2>/dev/null || echo "${INGESTED_BYTES:-0} B")"
    echo -e "    Current Rate:    $RATE lines/sec"
    
    echo -e "\n  Streaming:"
    echo -e "    Broadcasted:     $(printf "%'d" ${BROADCASTED:-0})"
    echo -e "    Dropped:         $(printf "%'d" ${DROPPED:-0})"
    echo -e "    Active Clients:  ${CLIENTS:-0}"
    
    echo -e "\n  Storage:"
    echo -e "    Total Chunks:    ${CHUNKS:-0}"
    echo -e "    Disk Usage:      $(numfmt --to=iec-i --suffix=B ${STORAGE:-0} 2>/dev/null || echo "${STORAGE:-0} B")"
    
    echo ""
}

# Function to show recent logs
show_recent_logs() {
    echo -e "${YELLOW}=== Recent Activity ===${NC}"
    
    QUERY_URL="${SERVER_URL}/query?query={}&limit=5"
    RESPONSE=$(curl -s "$QUERY_URL")
    
    if echo "$RESPONSE" | python3 -c "import sys, json; json.load(sys.stdin)" 2>/dev/null; then
        echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
logs = data.get('logs', [])
for log in logs[:5]:
    ts = log.get('timestamp', '')[:19].replace('T', ' ')
    level = log.get('level', 'info').upper()
    service = log.get('labels', {}).get('service', 'unknown')
    msg = log.get('message', '')[:60]
    print(f'  [{ts}] {level:5} [{service:15}] {msg}')
" 2>/dev/null || echo "  No recent logs found"
    else
        echo "  Could not fetch recent logs"
    fi
    
    echo ""
}

# Function to show alerts
show_alerts() {
    echo -e "${YELLOW}=== Active Alerts ===${NC}"
    
    ALERTS=$(curl -s "$PROMETHEUS_URL/api/v1/alerts" 2>/dev/null)
    
    if [ ! -z "$ALERTS" ]; then
        echo "$ALERTS" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    alerts = data.get('data', {}).get('alerts', [])
    if not alerts:
        print('  No active alerts')
    else:
        for alert in alerts:
            name = alert.get('labels', {}).get('alertname', 'Unknown')
            state = alert.get('state', 'unknown')
            severity = alert.get('labels', {}).get('severity', 'info')
            print(f'  • {name} [{severity}] - {state}')
except:
    print('  Could not fetch alerts')
" 2>/dev/null
    else
        echo "  Prometheus not available"
    fi
    
    echo ""
}

# Function to show docker stats
show_docker_stats() {
    echo -e "${YELLOW}=== Container Resources ===${NC}"
    
    if command -v docker &> /dev/null; then
        docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep -E "logpulse|NAME" | head -6
    else
        echo "  Docker not available"
    fi
    
    echo ""
}

# Function for continuous monitoring
monitor_continuous() {
    while true; do
        show_header
        check_health
        show_metrics
        show_recent_logs
        show_alerts
        show_docker_stats
        
        echo -e "${BLUE}Press Ctrl+C to exit${NC}"
        sleep 5
    done
}

# Function for single check
monitor_once() {
    show_header
    check_health
    show_metrics
    show_recent_logs
    show_alerts
    show_docker_stats
}

# Parse arguments
case "${1:-}" in
    -c|--continuous)
        monitor_continuous
        ;;
    -h|--help)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  -c, --continuous    Run in continuous monitoring mode"
        echo "  -h, --help          Show this help message"
        echo ""
        echo "Environment variables:"
        echo "  SERVER_URL          LogPulse server URL (default: http://localhost:8080)"
        echo "  PROMETHEUS_URL      Prometheus URL (default: http://localhost:9090)"
        ;;
    *)
        monitor_once
        ;;
esac