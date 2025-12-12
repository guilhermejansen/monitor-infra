.PHONY: all build agent server clean test run-server run-agent deps docker-build docker-push docker-run

# =============================================================================
# VARIÁVEIS
# =============================================================================
VERSION := $(shell cat .release-please-manifest.json | grep -o '"\.": "[^"]*"' | cut -d'"' -f4)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Docker
DOCKER_REGISTRY := docker.io
DOCKER_IMAGE := guilhermejansen/monitor-infra
DOCKER_TAG := $(VERSION)
PLATFORMS := linux/amd64,linux/arm64

# =============================================================================
# BUILD LOCAL
# =============================================================================
all: deps build

deps:
	go mod tidy

build: agent server

agent:
	@echo "Compilando agent para Linux amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/agent-linux-amd64 ./cmd/agent
	@echo "Compilando agent para Linux arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/agent-linux-arm64 ./cmd/agent
	@echo "Agent compilado com sucesso!"

server:
	@echo "Compilando server..."
	CGO_ENABLED=1 go build $(LDFLAGS) -o dist/server ./cmd/server
	@echo "Server compilado com sucesso!"

# =============================================================================
# EXECUÇÃO LOCAL
# =============================================================================
run-server: data
	go run ./cmd/server --port 8080 --db ./data/monitor.db --token dev-token

run-agent:
	go run ./cmd/agent --server http://localhost:8080 --token dev-token --interval 1 --once

# =============================================================================
# DOCKER
# =============================================================================
docker-build:
	@echo "Building Docker image (local)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest \
		--build-arg VERSION=$(VERSION) .

docker-buildx:
	@echo "Building Docker image (multi-arch: $(PLATFORMS))..."
	docker buildx build \
		--platform $(PLATFORMS) \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest \
		--push .

docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest

docker-run:
	@echo "Running Docker container..."
	docker run -d \
		--name monitor-infra \
		-p 8080:8080 \
		-v $(PWD)/data:/app/data \
		-e AUTH_TOKEN=dev-token \
		$(DOCKER_IMAGE):latest

docker-stop:
	docker stop monitor-infra || true
	docker rm monitor-infra || true

docker-logs:
	docker logs -f monitor-infra

# =============================================================================
# DOCKER COMPOSE (LOCAL)
# =============================================================================
compose-up:
	MONITOR_DOMAIN=localhost \
	MONITOR_AUTH_TOKEN=dev-token \
	docker-compose up -d

compose-down:
	docker-compose down

compose-logs:
	docker-compose logs -f

# =============================================================================
# TESTES
# =============================================================================
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	go vet ./...
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Go files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi

# =============================================================================
# LIMPEZA
# =============================================================================
clean:
	rm -rf dist/
	rm -f data/monitor.db*
	rm -f coverage.out

# =============================================================================
# UTILITÁRIOS
# =============================================================================
data:
	mkdir -p data

version:
	@echo "Version: $(VERSION)"

help:
	@echo "Monitor-Infra Makefile"
	@echo ""
	@echo "Comandos disponíveis:"
	@echo "  make build        - Compila agent e server"
	@echo "  make agent        - Compila apenas o agent (linux amd64/arm64)"
	@echo "  make server       - Compila apenas o server"
	@echo "  make run-server   - Executa server localmente"
	@echo "  make run-agent    - Executa agent localmente (uma vez)"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build - Build imagem Docker local"
	@echo "  make docker-buildx- Build multi-arch e push"
	@echo "  make docker-run   - Executa container local"
	@echo "  make docker-stop  - Para container local"
	@echo ""
	@echo "Compose:"
	@echo "  make compose-up   - Sobe stack local"
	@echo "  make compose-down - Derruba stack local"
	@echo ""
	@echo "Testes:"
	@echo "  make test         - Executa testes"
	@echo "  make lint         - Verifica código"
	@echo ""
	@echo "Outros:"
	@echo "  make clean        - Limpa arquivos gerados"
	@echo "  make version      - Mostra versão atual"
