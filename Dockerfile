# ============================================================================
# MONITOR-INFRA SERVER - Multi-stage Dockerfile
# Suporta: linux/amd64, linux/arm64
# ============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build
# -----------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

# Build arguments para multi-arch
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

# Instalar dependências de build (CGO para SQLite)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fonte
COPY . .

# Build com CGO habilitado para SQLite
RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X main.Version=${VERSION}" \
    -o server ./cmd/server

# -----------------------------------------------------------------------------
# Stage 2: Runtime
# -----------------------------------------------------------------------------
FROM alpine:3.20

# Labels OCI
LABEL org.opencontainers.image.title="Monitor-Infra Server"
LABEL org.opencontainers.image.description="Servidor de monitoramento de infraestrutura VPS"
LABEL org.opencontainers.image.vendor="Monitor-Infra"
LABEL org.opencontainers.image.source="https://github.com/Setpar-IA-Setup-Automatizado/monitor-infra"
LABEL org.opencontainers.image.licenses="MIT"

# Instalar runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sqlite-libs \
    && rm -rf /var/cache/apk/*

# Criar usuário não-root
RUN addgroup -g 1000 monitor && \
    adduser -u 1000 -G monitor -s /bin/sh -D monitor

# Criar diretórios necessários
RUN mkdir -p /app/data && chown -R monitor:monitor /app

WORKDIR /app

# Copiar binário do builder
COPY --from=builder /build/server /app/server

# Usar usuário não-root
USER monitor

# Porta padrão
EXPOSE 8080

# Volume para dados persistentes
VOLUME ["/app/data"]

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1

# Entrypoint
ENTRYPOINT ["/app/server"]

# Argumentos padrão
CMD ["--port", "8080", "--db", "/app/data/monitor.db"]
