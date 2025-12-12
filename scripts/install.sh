#!/bin/bash
set -e

# ============================================================================
# MONITOR-INFRA AGENT INSTALLER
# ============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

INSTALL_DIR="/opt/monitor-agent"
SERVICE_NAME="monitor-agent"

SERVER_URL=""
AUTH_TOKEN=""
MACHINE_NAME=""
GROUP_NAME="default"
INTERVAL_MINUTES=60

print_header() {
    echo -e "${CYAN}"
    echo "+============================================================+"
    echo "|         MONITOR-INFRA - Instalacao do Agent                |"
    echo "+============================================================+"
    echo -e "${NC}"
}

print_success() { echo -e "${GREEN}[OK]${NC} $1"; }
print_error() { echo -e "${RED}[ERRO]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[!]${NC} $1"; }
print_info() { echo -e "${CYAN}[i]${NC} $1"; }

show_help() {
    echo "Uso: curl -sSL <server>/install.sh | bash -s -- [opcoes]"
    echo ""
    echo "Opcoes obrigatorias:"
    echo "  --server URL      URL do servidor de monitoramento"
    echo ""
    echo "Opcoes:"
    echo "  --token TOKEN     Token de autenticacao"
    echo "  --name NOME       Nome da maquina (default: hostname)"
    echo "  --group GRUPO     Grupo (default: default)"
    echo "  --interval MIN    Intervalo em minutos (default: 60)"
    echo "  -h, --help        Mostra esta ajuda"
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --server) SERVER_URL="$2"; shift 2 ;;
            --token) AUTH_TOKEN="$2"; shift 2 ;;
            --name) MACHINE_NAME="$2"; shift 2 ;;
            --group) GROUP_NAME="$2"; shift 2 ;;
            --interval) INTERVAL_MINUTES="$2"; shift 2 ;;
            -h|--help) show_help; exit 0 ;;
            *) print_error "Argumento desconhecido: $1"; show_help; exit 1 ;;
        esac
    done
}

validate_args() {
    if [[ -z "$SERVER_URL" ]]; then
        print_error "URL do servidor e obrigatoria (--server)"
        show_help
        exit 1
    fi
    SERVER_URL="${SERVER_URL%/}"

    if [[ -z "$MACHINE_NAME" ]]; then
        MACHINE_NAME=$(hostname)
    fi
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "Este script precisa ser executado como root"
        exit 1
    fi
}

detect_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) print_error "Arquitetura nao suportada: $ARCH"; exit 1 ;;
    esac
    print_success "Arquitetura detectada: $ARCH"
}

check_systemd() {
    if ! command -v systemctl &> /dev/null; then
        print_error "systemd nao encontrado"
        exit 1
    fi
}

download_binary() {
    print_info "Baixando agent..."
    mkdir -p "$INSTALL_DIR"

    local url="${SERVER_URL}/download/agent-linux-${ARCH}"

    if command -v curl &> /dev/null; then
        curl -sSL "$url" -o "$INSTALL_DIR/monitor-agent"
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O "$INSTALL_DIR/monitor-agent"
    else
        print_error "curl ou wget nao encontrado"
        exit 1
    fi

    chmod +x "$INSTALL_DIR/monitor-agent"
    print_success "Agent baixado"
}

create_config() {
    print_info "Criando configuracao..."

    cat > "$INSTALL_DIR/config.env" << EOF
SERVER_URL=${SERVER_URL}
AUTH_TOKEN=${AUTH_TOKEN}
MACHINE_NAME=${MACHINE_NAME}
GROUP_NAME=${GROUP_NAME}
INTERVAL_MINUTES=${INTERVAL_MINUTES}
EOF

    chmod 600 "$INSTALL_DIR/config.env"
    print_success "Configuracao criada"
}

install_service() {
    print_info "Instalando servico systemd..."

    cat > "/etc/systemd/system/${SERVICE_NAME}.service" << EOF
[Unit]
Description=Monitor-Infra Agent
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=${INSTALL_DIR}/config.env
ExecStart=${INSTALL_DIR}/monitor-agent \
    --server \${SERVER_URL} \
    --token \${AUTH_TOKEN} \
    --name \${MACHINE_NAME} \
    --group \${GROUP_NAME} \
    --interval \${INTERVAL_MINUTES}
Restart=always
RestartSec=30

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME" --quiet
    print_success "Servico instalado"
}

start_service() {
    print_info "Iniciando servico..."
    systemctl start "$SERVICE_NAME"

    sleep 2

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        print_success "Servico iniciado"
    else
        print_error "Falha ao iniciar servico"
        echo "Verifique: journalctl -u $SERVICE_NAME -n 50"
        exit 1
    fi
}

print_success_message() {
    echo ""
    echo -e "${CYAN}============================================================${NC}"
    echo -e "${GREEN}${BOLD}  Agent instalado com sucesso!${NC}"
    echo ""
    echo -e "  Maquina: ${BOLD}${MACHINE_NAME}${NC}"
    echo -e "  Grupo:   ${BOLD}${GROUP_NAME}${NC}"
    echo -e "  Server:  ${BOLD}${SERVER_URL}${NC}"
    echo -e "${CYAN}============================================================${NC}"
    echo ""
    echo -e "${BOLD}Comandos uteis:${NC}"
    echo -e "  ${CYAN}systemctl status ${SERVICE_NAME}${NC}    # Ver status"
    echo -e "  ${CYAN}systemctl restart ${SERVICE_NAME}${NC}   # Reiniciar"
    echo -e "  ${CYAN}journalctl -u ${SERVICE_NAME} -f${NC}    # Ver logs"
    echo ""
}

main() {
    print_header
    parse_args "$@"
    validate_args
    check_root
    detect_arch
    check_systemd
    download_binary
    create_config
    install_service
    start_service
    print_success_message
}

main "$@"
