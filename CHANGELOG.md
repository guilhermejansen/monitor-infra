# Changelog

Todas as mudanças notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [1.2.0](https://github.com/guilhermejansen/monitor-infra/compare/v1.1.1...v1.2.0) (2025-12-12)


### Features

* **ci:** adiciona automações para gerar atividade no perfil ([1f7efdb](https://github.com/guilhermejansen/monitor-infra/commit/1f7efdb1a695e35f6442129086f9f62093631c2a))

## [1.1.1](https://github.com/guilhermejansen/monitor-infra/compare/v1.1.0...v1.1.1) (2025-12-12)


### Bug Fixes

* **ci:** usa PAT_TOKEN para commits do Release Please ([6b4a7ac](https://github.com/guilhermejansen/monitor-infra/commit/6b4a7ac5bb9de8e4107bc87e0a45559d9ed854e4))
* **docker:** corrige build multi-arch com CGO ([e563c37](https://github.com/guilhermejansen/monitor-infra/commit/e563c37406f574779ab58a55660f737264ce6cd0))

## [1.1.0](https://github.com/guilhermejansen/monitor-infra/compare/v1.0.0...v1.1.0) (2025-12-12)


### Features

* **docker:** adiciona .env.example e simplifica docker-compose ([801270a](https://github.com/guilhermejansen/monitor-infra/commit/801270a73634b80814ea7963c2e3b84da8d6fe30))


### Bug Fixes

* **ci:** altera registry para GitHub Container Registry ([8ba1957](https://github.com/guilhermejansen/monitor-infra/commit/8ba1957cfb584f3adbb99c62427f6ed1b0a1c7f4))
* **ci:** corrige nome da imagem para setupautomatizado/monitor-infra ([adb3dac](https://github.com/guilhermejansen/monitor-infra/commit/adb3daced2db5781b110b2381c4b4589de50f7ff))

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
