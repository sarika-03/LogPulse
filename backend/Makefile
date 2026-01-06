.PHONY: help build run test clean docker-build docker-up docker-down logs setup monitor

# Variables
BINARY_NAME=logpulse
AGENT_BINARY=logpulse-agent
DOCKER_COMPOSE=cd docker && docker-compose

# Default target
help:
	@echo "LogPulse Development Commands"
	@echo "=============================="
	@echo ""
	@echo "Setup & Installation:"
	@echo "  make setup              - Complete system setup"
	@echo "  make install-deps       - Install Go dependencies"
	@echo ""
	@echo "Development:"
	@echo "  make build              - Build server binary"
	@echo "  make build-agent        - Build agent binary"
	@echo "  make run                - Run server locally"
	@echo "  make run-agent          - Run agent locally"
	@echo "  make test               - Run tests"
	@echo "  make lint               - Run linter"
	@echo ""
	@echo "Docker Operations:"
	@echo "  make docker-build       - Build Docker images"
	@echo "  make docker-up          - Start all services"
	@echo "  make docker-down        - Stop all services"
	@echo "  make docker-restart     - Restart all services"
	@echo "  make docker-clean       - Remove containers and volumes"
	@echo ""
	@echo "Monitoring & Testing:"
	@echo "  make logs               - View container logs"
	@echo "  make monitor            - Monitor system health"
	@echo "  make test-ingest        - Test log ingestion"
	@echo "  make test-stream        - Test WebSocket streaming"
	@echo "  make test-query         - Test query API"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make clean-data         - Clean log data"
	@echo "  make backup             - Backup configuration and data"

# Setup & Installation
setup:
	@echo "Running complete setup..."
	@bash scripts/setup.sh

install-deps:
	@echo "Installing Go dependencies..."
	@cd cmd/server && go mod download
	@cd cmd/agent && go mod download
	@echo "✓ Dependencies installed"

# Development
build:
	@echo "Building server..."
	@cd cmd/server && go build -o ../../$(BINARY_NAME) .
	@echo "✓ Server built: ./$(BINARY_NAME)"

build-agent:
	@echo "Building agent..."
	@cd cmd/agent && go build -o ../../$(AGENT_BINARY) .
	@echo "✓ Agent built: ./$(AGENT_BINARY)"

run: build
	@echo "Starting LogPulse server..."
	@./$(BINARY_NAME)

run-agent: build-agent
	@echo "Starting LogPulse agent..."
	@./$(AGENT_BINARY) -config configs/agent-config.yaml

test:
	@echo "Running tests..."
	@go test -v ./internal/...
	@echo "✓ Tests passed"

lint:
	@echo "Running linter..."
	@golangci-lint run ./...
	@echo "✓ Lint passed"

# Docker Operations
docker-build:
	@echo "Building Docker images..."
	@$(DOCKER_COMPOSE) build
	@echo "✓ Docker images built"

docker-up:
	@echo "Starting Docker services..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@make docker-health
	@echo ""
	@echo "Services are running:"
	@echo "  • LogPulse:    http://localhost:8080"
	@echo "  • Prometheus:  http://localhost:9090"
	@echo "  • Grafana:     http://localhost:3000 (admin/admin)"

docker-down:
	@echo "Stopping Docker services..."
	@$(DOCKER_COMPOSE) down
	@echo "✓ Services stopped"

docker-restart:
	@echo "Restarting Docker services..."
	@$(DOCKER_COMPOSE) restart
	@echo "✓ Services restarted"

docker-clean:
	@echo "Cleaning Docker containers and volumes..."
	@$(DOCKER_COMPOSE) down -v
	@docker system prune -f
	@echo "✓ Cleaned"

docker-health:
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health > /dev/null && echo "  ✓ LogPulse is healthy" || echo "  ✗ LogPulse is down"
	@curl -s http://localhost:9090/-/ready > /dev/null && echo "  ✓ Prometheus is ready" || echo "  ✗ Prometheus is down"
	@curl -s http://localhost:3000/api/health > /dev/null && echo "  ✓ Grafana is healthy" || echo "  ✗ Grafana is down"

# Monitoring & Testing
logs:
	@$(DOCKER_COMPOSE) logs -f

logs-server:
	@$(DOCKER_COMPOSE) logs -f logpulse

logs-prometheus:
	@$(DOCKER_COMPOSE) logs -f prometheus

logs-grafana:
	@$(DOCKER_COMPOSE) logs -f grafana

monitor:
	@bash scripts/monitor.sh -c

monitor-once:
	@bash scripts/monitor.sh

test-ingest:
	@echo "Testing log ingestion..."
	@bash scripts/generate_logs.sh 10
	@echo "✓ Test complete"

test-stream:
	@echo "Testing WebSocket streaming..."
	@bash scripts/test_streaming.sh
	@echo "✓ Test complete"

test-query:
	@echo "Testing query API..."
	@curl -s "http://localhost:8080/query?query={service=\"api-gateway\"}&limit=5" | python3 -m json.tool
	@echo "✓ Test complete"

test-all: test-ingest test-query
	@echo "✓ All tests complete"

# Maintenance
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME) $(AGENT_BINARY)
	@rm -rf vendor/
	@find . -name "*.log" -type f -delete
	@echo "✓ Cleaned"

clean-data:
	@echo "Cleaning log data..."
	@rm -rf data/logs/*
	@echo "✓ Data cleaned"

backup:
	@echo "Creating backup..."
	@mkdir -p backups
	@tar -czf backups/logpulse-backup-$(shell date +%Y%m%d-%H%M%S).tar.gz \
		configs/ data/ \
		--exclude='data/logs/*' \
		--exclude='*.log'
	@echo "✓ Backup created in backups/"

# Development helpers
dev-setup: install-deps
	@echo "Setting up development environment..."
	@mkdir -p data/logs
	@cp configs/config.yaml.example configs/config.yaml 2>/dev/null || true
	@cp configs/webhooks.yaml.example configs/webhooks.yaml 2>/dev/null || true
	@echo "✓ Development environment ready"

dev-run:
	@echo "Starting development mode..."
	@make build
	@make run

# Quick commands
quick-start: docker-up
	@sleep 3
	@make test-ingest
	@make monitor-once

quick-stop: docker-down clean

# Install tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Tools installed"