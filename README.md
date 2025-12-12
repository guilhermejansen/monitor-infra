# Monitor-Infra

[![CI](https://github.com/guilhermejansen/monitor-infra/actions/workflows/ci.yml/badge.svg)](https://github.com/guilhermejansen/monitor-infra/actions/workflows/ci.yml)
[![Release](https://github.com/guilhermejansen/monitor-infra/actions/workflows/release.yml/badge.svg)](https://github.com/guilhermejansen/monitor-infra/actions/workflows/release.yml)
[![CodeQL](https://github.com/guilhermejansen/monitor-infra/actions/workflows/codeql.yml/badge.svg)](https://github.com/guilhermejansen/monitor-infra/actions/workflows/codeql.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/guilhermejansen/monitor-infra)](https://go.dev/)
[![License](https://img.shields.io/github/license/guilhermejansen/monitor-infra)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/guilhermejansen/monitor-infra)](https://github.com/guilhermejansen/monitor-infra/releases)

Sistema de monitoramento de infraestrutura VPS com suporte a Docker Swarm.

## Funcionalidades

- Monitoramento de CPU, Memória e Disco
- Contagem de containers Docker (rodando/parados)
- Detecção de roles Docker Swarm (manager/worker)
- Dashboard web moderno e responsivo (dark mode)
- Auto-registro de máquinas via POST
- API REST com autenticação por token
- Retenção configurável de métricas
- Suporte a múltiplas arquiteturas (amd64/arm64)

## Arquitetura

```
┌─────────────────┐     POST /api/metrics      ┌─────────────────┐
│   VPS Agent     │ ─────────────────────────► │    Server       │
│  (cada máquina) │      (a cada hora)         │   (central)     │
└─────────────────┘                            └────────┬────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │     SQLite      │
                                               │   (dados 90d)   │
                                               └─────────────────┘
```

## Quick Start

### Docker (Recomendado)

```bash
docker run -d \
  --name monitor-infra \
  -p 8080:8080 \
  -v monitor-data:/app/data \
  -e AUTH_TOKEN=seu-token-secreto \
  setupautomatizado/monitor-infra:latest
```

### Docker Compose / Portainer

Veja o arquivo `docker-compose.yml` para deploy completo com Traefik v3.

### Instalação do Agent nas VPS

```bash
curl -sSL https://seu-servidor/install.sh | bash -s -- \
  --server https://seu-servidor \
  --token seu-token-secreto \
  --group producao
```

## Configuração

### Variáveis de Ambiente (Server)

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `AUTH_TOKEN` | Token de autenticação | (obrigatório) |
| `RETENTION_DAYS` | Dias de retenção | 90 |
| `TZ` | Timezone | America/Sao_Paulo |

### Parâmetros CLI (Server)

```bash
./server --help

Flags:
  --port      Porta do servidor (default: 8080)
  --db        Caminho do banco SQLite (default: ./data/monitor.db)
  --token     Token de autenticação
  --retention Dias de retenção (default: 90)
```

### Parâmetros CLI (Agent)

```bash
./agent --help

Flags:
  --server    URL do servidor (obrigatório)
  --token     Token de autenticação
  --name      Nome da máquina (default: hostname)
  --group     Grupo da máquina (default: default)
  --interval  Intervalo em minutos (default: 60)
  --once      Executar apenas uma vez
```

## API

### Endpoints

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/` | Dashboard web |
| GET | `/api/health` | Health check |
| POST | `/api/metrics` | Receber métricas (requer token) |
| GET | `/api/machines` | Listar máquinas (requer token) |
| GET | `/api/machines/:id` | Detalhes de uma máquina |
| GET | `/api/machines/:id/history` | Histórico de métricas |
| GET | `/api/stats` | Estatísticas gerais |
| GET | `/install.sh` | Script de instalação |
| GET | `/download/agent-linux-{arch}` | Download do agent |

### Exemplo de Payload (POST /api/metrics)

```json
{
  "hostname": "vps-prod-01",
  "ip": "192.168.1.100",
  "group": "producao",
  "swarm_role": "manager",
  "cpu_percent": 45.2,
  "memory_percent": 67.8,
  "disk_percent": 23.1,
  "docker_running": 5,
  "docker_stopped": 2
}
```

## Deploy no Portainer

### 1. Configurar Secrets no GitHub

Vá em **Settings > Secrets and variables > Actions**:

- `DOCKERHUB_TOKEN`: Token de acesso do DockerHub

E em **Variables**:

- `DOCKERHUB_USERNAME`: Seu usuário do DockerHub

### 2. Deploy no Portainer

1. Vá em **Stacks > Add stack**
2. Escolha **Repository** e aponte para este repositório
3. Configure as variáveis de ambiente:

```env
MONITOR_DOMAIN=monitor.seudominio.com
MONITOR_AUTH_TOKEN=seu-token-secreto
MONITOR_RETENTION_DAYS=90
MONITOR_VERSION=latest
TZ=America/Sao_Paulo
```

4. Clique em **Deploy the stack**

## Desenvolvimento

### Requisitos

- Go 1.23+
- Docker & Docker Buildx
- Make

### Build Local

```bash
# Compilar tudo
make build

# Apenas server
make server

# Apenas agent (multi-arch)
make agent

# Executar localmente
make run-server
```

### Docker Local

```bash
# Build local
make docker-build

# Build multi-arch (requer buildx)
make docker-buildx

# Executar container
make docker-run
```

## Conventional Commits

Este projeto usa [Conventional Commits](https://www.conventionalcommits.org/) para versionamento automático:

```bash
# Patch (0.0.X) - Correções
git commit -m "fix: corrige cálculo de CPU"

# Minor (0.X.0) - Novas funcionalidades
git commit -m "feat: adiciona suporte a alertas"

# Major (X.0.0) - Breaking changes
git commit -m "feat!: nova API de autenticação

BREAKING CHANGE: Token agora é obrigatório no header"
```

## Estrutura do Projeto

```
monitor-infra/
├── cmd/
│   ├── agent/          # Executável do agent
│   └── server/         # Executável do server
├── internal/
│   ├── collector/      # Coleta de métricas
│   ├── dashboard/      # Templates HTML
│   └── storage/        # Persistência SQLite
├── scripts/
│   └── install.sh      # Script de instalação
├── .github/
│   └── workflows/      # CI/CD
├── Dockerfile          # Build multi-arch
├── docker-compose.yml  # Stack Portainer/Traefik
├── Makefile
└── README.md
```

## Licença

MIT License - veja [LICENSE](LICENSE) para detalhes.
