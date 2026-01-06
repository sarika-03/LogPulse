#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   LogPulse Complete Setup Script      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}\n"

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}[1/7] Checking prerequisites...${NC}"
    
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}✗ Docker is not installed${NC}"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        echo -e "${RED}✗ Docker Compose is not installed${NC}"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        echo -e "${YELLOW}⚠ Go is not installed (needed for local development)${NC}"
    fi
    
    echo -e "${GREEN}✓ Prerequisites check passed${NC}\n"
}

# Create directory structure
create_directories() {
    echo -e "${YELLOW}[2/7] Creating directory structure...${NC}"
    
    mkdir -p backend/data/logs
    mkdir -p backend/configs
    mkdir -p backend/docker
    mkdir -p backend/scripts
    mkdir -p backend/clients
    
    echo -e "${GREEN}✓ Directories created${NC}\n"
}

# Setup configuration files
setup_configs() {
    echo -e "${YELLOW}[3/7] Setting up configuration files...${NC}"
    
    # Create default config if not exists
    if [ ! -f backend/configs/config.yaml ]; then
        cat > backend/configs/config.yaml <<EOF
server:
  port: "8080"

storage:
  path: "./data/logs"
  chunk_size_bytes: 1048576
  retention_days: 7

ingest:
  buffer_size: 1000
  flush_interval_ms: 5000

auth:
  enabled: false
  api_key: ""
EOF
    fi
    
    # Create webhooks config
    if [ ! -f backend/configs/webhooks.yaml ]; then
        cat > backend/configs/webhooks.yaml <<EOF
webhooks:
  - url: "http://localhost:9000/webhook"
    events: ["log", "alert"]
EOF
    fi
    
    # Create alerts config
    if [ ! -f backend/configs/alerts.yaml ]; then
        cat > backend/configs/alerts.yaml <<EOF
alerts:
  - name: "High Error Rate"
    expr: '{level="error"}'
    threshold: 10
    window: 5m
    channels: ["webhook"]
    labels:
      severity: "warning"
EOF
    fi
    
    # Create agent config
    if [ ! -f backend/configs/agent-config.yaml ]; then
        cat > backend/configs/agent-config.yaml <<EOF
server:
  url: "http://logpulse:8080"
  api_key: ""

positions_file: "./positions.json"

targets:
  - path: "/var/log/syslog"
    labels:
      job: "syslog"
      env: "prod"
EOF
    fi
    
    echo -e "${GREEN}✓ Configuration files created${NC}\n"
}

# Build Docker images
build_images() {
    echo -e "${YELLOW}[4/7] Building Docker images...${NC}"
    
    cd backend/docker
    docker-compose build
    cd ../..
    
    echo -e "${GREEN}✓ Docker images built${NC}\n"
}

# Initialize database/storage
init_storage() {
    echo -e "${YELLOW}[5/7] Initializing storage...${NC}"
    
    mkdir -p backend/data/logs
    chmod -R 755 backend/data
    
    echo -e "${GREEN}✓ Storage initialized${NC}\n"
}

# Start services
start_services() {
    echo -e "${YELLOW}[6/7] Starting services...${NC}"
    
    cd backend/docker
    docker-compose up -d
    cd ../..
    
    echo -e "${GREEN}✓ Services started${NC}\n"
}

# Wait for services and verify
verify_services() {
    echo -e "${YELLOW}[7/7] Verifying services...${NC}"
    
    echo "Waiting for LogPulse to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            echo -e "${GREEN}✓ LogPulse is ready${NC}"
            break
        fi
        sleep 2
    done
    
    echo "Waiting for Prometheus to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:9090/-/ready > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Prometheus is ready${NC}"
            break
        fi
        sleep 2
    done
    
    echo "Waiting for Grafana to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:3000/api/health > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Grafana is ready${NC}"
            break
        fi
        sleep 2
    done
    
    echo ""
}

# Print access information
print_info() {
    echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║         Setup Complete!                ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════╝${NC}\n"
    
    echo -e "${GREEN}Access your services:${NC}"
    echo -e "  • LogPulse API:    ${BLUE}http://localhost:8080${NC}"
    echo -e "  • Health Check:    ${BLUE}http://localhost:8080/health${NC}"
    echo -e "  • Metrics:         ${BLUE}http://localhost:8080/metrics${NC}"
    echo -e "  • WebSocket:       ${BLUE}ws://localhost:8080/stream${NC}"
    echo -e "  • Prometheus:      ${BLUE}http://localhost:9090${NC}"
    echo -e "  • Grafana:         ${BLUE}http://localhost:3000${NC} (admin/admin)"
    echo -e "  • AlertManager:    ${BLUE}http://localhost:9093${NC}\n"
    
    echo -e "${GREEN}Quick commands:${NC}"
    echo -e "  • View logs:       ${BLUE}cd backend/docker && docker-compose logs -f${NC}"
    echo -e "  • Stop services:   ${BLUE}cd backend/docker && docker-compose down${NC}"
    echo -e "  • Restart:         ${BLUE}cd backend/docker && docker-compose restart${NC}"
    echo -e "  • Test ingest:     ${BLUE}bash backend/scripts/generate_logs.sh 10${NC}"
    echo -e "  • Test streaming:  ${BLUE}bash backend/scripts/test_streaming.sh${NC}\n"
    
    echo -e "${YELLOW}Next steps:${NC}"
    echo "  1. Configure your log sources in backend/configs/agent-config.yaml"
    echo "  2. Set up alerts in backend/configs/alerts.yaml"
    echo "  3. Configure webhooks in backend/configs/webhooks.yaml"
    echo "  4. Import Grafana dashboards from backend/docker/grafana-dashboard.json"
    echo ""
}

# Main execution
main() {
    check_prerequisites
    create_directories
    setup_configs
    build_images
    init_storage
    start_services
    verify_services
    print_info
}

# Run main function
main