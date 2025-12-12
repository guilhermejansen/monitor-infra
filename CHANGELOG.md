# Changelog

Todas as mudanças notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## 1.0.0 (2025-12-12)


### Features

* **project:** inicialização do projeto ([d42b921](https://github.com/guilhermejansen/monitor-infra/commit/d42b9210ecf293f4c657b2549a3269fcc100e52d))


### Bug Fixes

* **deps:** atualiza versão do Go para 1.24 ([69650f5](https://github.com/guilhermejansen/monitor-infra/commit/69650f53b105146f3dadfe561068974ea2886c7a))
* **repo:** ajuste de repositorio ([2341c48](https://github.com/guilhermejansen/monitor-infra/commit/2341c4886c590b615f454dc7869b97f1ae56d682))

## [0.1.0] - 2025-12-12

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
