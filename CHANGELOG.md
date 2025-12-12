# Changelog

Todas as mudanças notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [0.1.0] - 2024-12-12

### Funcionalidades

- Implementação inicial do servidor de monitoramento
- Dashboard web responsivo com dark mode
- Coleta de métricas: CPU, Memória, Disco
- Integração com Docker para contagem de containers
- Detecção de roles Docker Swarm (manager/worker)
- API REST com autenticação via Bearer token
- Auto-registro de máquinas via POST
- Script de instalação automática do agent
- Suporte a múltiplas arquiteturas (amd64/arm64)
- Pipeline CI/CD com GitHub Actions
- Docker image multi-arch para DockerHub
- Stack Docker Compose para Portainer com Traefik v3
