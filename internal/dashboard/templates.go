package dashboard

// GetHTML retorna o HTML completo do dashboard
func GetHTML() string {
	return `<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="theme-color" content="#0f172a">
    <title>Monitor Infra</title>
    <style>
        :root {
            --bg-primary: #0f172a;
            --bg-card: #1e293b;
            --bg-hover: #334155;
            --border-color: #334155;
            --text-primary: #f8fafc;
            --text-muted: #94a3b8;
            --green: #22c55e;
            --yellow: #eab308;
            --red: #ef4444;
            --blue: #3b82f6;
            --cyan: #06b6d4;
            --purple: #8b5cf6;
            --orange: #f97316;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.5;
        }

        /* Header */
        header {
            background: var(--bg-card);
            border-bottom: 1px solid var(--border-color);
            padding: 1rem 1.5rem;
            position: sticky;
            top: 0;
            z-index: 100;
        }

        .header-content {
            max-width: 1400px;
            margin: 0 auto;
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 1rem;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .logo-icon {
            font-size: 1.5rem;
        }

        .logo h1 {
            font-size: 1.25rem;
            font-weight: 600;
        }

        .header-info {
            display: flex;
            align-items: center;
            gap: 1rem;
            font-size: 0.875rem;
            color: var(--text-muted);
        }

        /* Stats Cards */
        .stats-container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 1.5rem;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }

        .stat-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 0.75rem;
            padding: 1.25rem;
            text-align: center;
        }

        .stat-value {
            font-size: 2rem;
            font-weight: 700;
            line-height: 1;
        }

        .stat-label {
            font-size: 0.75rem;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-top: 0.5rem;
        }

        .stat-card.online .stat-value { color: var(--green); }
        .stat-card.warning .stat-value { color: var(--yellow); }
        .stat-card.offline .stat-value { color: var(--red); }
        .stat-card.containers .stat-value { color: var(--cyan); }

        /* Main Content */
        main {
            max-width: 1400px;
            margin: 0 auto;
            padding: 0 1.5rem 2rem;
        }

        /* Groups */
        .group {
            margin-bottom: 2rem;
        }

        .group-header {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem 0;
            margin-bottom: 1rem;
            border-bottom: 1px solid var(--border-color);
            color: var(--text-muted);
            font-size: 0.875rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        /* Machine Grid */
        .machines-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
            gap: 1rem;
        }

        /* Machine Card */
        .machine-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 0.75rem;
            padding: 1.25rem;
            transition: transform 0.2s, box-shadow 0.2s;
            border-left: 4px solid var(--green);
        }

        .machine-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
        }

        .machine-card.warning {
            border-left-color: var(--yellow);
        }

        .machine-card.offline {
            border-left-color: var(--red);
            opacity: 0.7;
        }

        .card-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 1rem;
        }

        .hostname {
            font-weight: 600;
            font-size: 1.1rem;
        }

        .badges {
            display: flex;
            gap: 0.5rem;
            flex-wrap: wrap;
        }

        .badge {
            font-size: 0.65rem;
            padding: 0.2rem 0.5rem;
            border-radius: 1rem;
            font-weight: 600;
            text-transform: uppercase;
        }

        .badge-online { background: var(--green); color: #000; }
        .badge-warning { background: var(--yellow); color: #000; }
        .badge-offline { background: var(--red); color: #fff; }
        .badge-manager { background: var(--blue); color: #fff; }
        .badge-worker { background: var(--purple); color: #fff; }

        /* Metrics */
        .metrics {
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
            margin-bottom: 1rem;
        }

        .metric {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .metric-label {
            width: 50px;
            font-size: 0.75rem;
            color: var(--text-muted);
            text-transform: uppercase;
        }

        .bar-container {
            flex: 1;
            height: 8px;
            background: var(--border-color);
            border-radius: 4px;
            overflow: hidden;
        }

        .bar-fill {
            height: 100%;
            border-radius: 4px;
            transition: width 0.5s ease;
        }

        .bar-fill.cpu { background: var(--cyan); }
        .bar-fill.mem { background: var(--purple); }
        .bar-fill.disk { background: var(--orange); }

        .bar-fill.warning { background: var(--yellow); }
        .bar-fill.critical { background: var(--red); }

        .metric-value {
            width: 45px;
            text-align: right;
            font-size: 0.875rem;
            font-weight: 500;
        }

        /* Docker Info */
        .docker-info {
            display: flex;
            gap: 1rem;
            padding-top: 0.75rem;
            border-top: 1px solid var(--border-color);
            font-size: 0.875rem;
        }

        .docker-stat {
            display: flex;
            align-items: center;
            gap: 0.25rem;
            color: var(--text-muted);
        }

        .docker-up { color: var(--green); font-weight: 600; }
        .docker-down { color: var(--red); font-weight: 600; }

        /* Footer */
        .card-footer {
            margin-top: 0.75rem;
            padding-top: 0.75rem;
            border-top: 1px solid var(--border-color);
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 0.75rem;
            color: var(--text-muted);
        }

        /* Empty State */
        .empty-state {
            text-align: center;
            padding: 4rem 2rem;
            color: var(--text-muted);
        }

        .empty-state h2 {
            font-size: 1.5rem;
            margin-bottom: 1rem;
            color: var(--text-primary);
        }

        .empty-state p {
            margin-bottom: 1.5rem;
        }

        .empty-state code {
            display: block;
            background: var(--bg-card);
            padding: 1rem;
            border-radius: 0.5rem;
            font-family: 'Monaco', 'Consolas', monospace;
            font-size: 0.875rem;
            overflow-x: auto;
            text-align: left;
            color: var(--cyan);
        }

        /* Loading */
        .loading {
            text-align: center;
            padding: 4rem;
            color: var(--text-muted);
        }

        .spinner {
            display: inline-block;
            width: 40px;
            height: 40px;
            border: 3px solid var(--border-color);
            border-top-color: var(--cyan);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        /* Pulse Animation */
        .pulse {
            animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        /* Responsive */
        @media (max-width: 768px) {
            header {
                padding: 1rem;
            }

            .header-content {
                flex-direction: column;
                align-items: flex-start;
            }

            .stats-container, main {
                padding: 1rem;
            }

            .stats-grid {
                grid-template-columns: repeat(2, 1fr);
            }

            .machines-grid {
                grid-template-columns: 1fr;
            }

            .machine-card {
                padding: 1rem;
            }
        }
    </style>
</head>
<body>
    <header>
        <div class="header-content">
            <div class="logo">
                <span class="logo-icon">📊</span>
                <h1>Monitor Infra</h1>
            </div>
            <div class="header-info">
                <span id="update-time">Atualizando...</span>
                <span class="pulse">●</span>
            </div>
        </div>
    </header>

    <div class="stats-container">
        <div class="stats-grid" id="stats">
            <div class="stat-card online">
                <div class="stat-value" id="stat-online">-</div>
                <div class="stat-label">Online</div>
            </div>
            <div class="stat-card warning">
                <div class="stat-value" id="stat-warning">-</div>
                <div class="stat-label">Atenção</div>
            </div>
            <div class="stat-card offline">
                <div class="stat-value" id="stat-offline">-</div>
                <div class="stat-label">Offline</div>
            </div>
            <div class="stat-card containers">
                <div class="stat-value" id="stat-containers">-</div>
                <div class="stat-label">Containers</div>
            </div>
        </div>
    </div>

    <main id="main">
        <div class="loading">
            <div class="spinner"></div>
            <p>Carregando...</p>
        </div>
    </main>

    <script>
        const REFRESH_INTERVAL = 60000; // 60 segundos
        const ONLINE_THRESHOLD_MINUTES = 70;
        const WARNING_THRESHOLD = 85;

        function getBarClass(type, value) {
            if (value >= 95) return type + ' critical';
            if (value >= WARNING_THRESHOLD) return type + ' warning';
            return type;
        }

        function getStatusClass(machine) {
            const lastSeen = new Date(machine.last_seen);
            const now = new Date();
            const diffMinutes = (now - lastSeen) / 1000 / 60;

            if (diffMinutes > ONLINE_THRESHOLD_MINUTES) return 'offline';

            const m = machine.metrics;
            if (m && (m.cpu_percent > WARNING_THRESHOLD || m.memory_percent > WARNING_THRESHOLD || m.disk_percent > WARNING_THRESHOLD)) {
                return 'warning';
            }

            return 'online';
        }

        function formatTime(dateStr) {
            const date = new Date(dateStr);
            const now = new Date();
            const diffMinutes = Math.floor((now - date) / 1000 / 60);

            if (diffMinutes < 1) return 'Agora';
            if (diffMinutes < 60) return 'Há ' + diffMinutes + ' min';
            if (diffMinutes < 1440) return 'Há ' + Math.floor(diffMinutes / 60) + 'h';
            return 'Há ' + Math.floor(diffMinutes / 1440) + 'd';
        }

        function renderMachine(machine) {
            const status = getStatusClass(machine);
            const m = machine.metrics || {};

            let swarmBadge = '';
            if (machine.swarm_role === 'manager') {
                swarmBadge = '<span class="badge badge-manager">Manager</span>';
            } else if (machine.swarm_role === 'worker') {
                swarmBadge = '<span class="badge badge-worker">Worker</span>';
            }

            let statusBadge = '';
            if (status === 'online') {
                statusBadge = '<span class="badge badge-online">Online</span>';
            } else if (status === 'warning') {
                statusBadge = '<span class="badge badge-warning">Atenção</span>';
            } else {
                statusBadge = '<span class="badge badge-offline">Offline</span>';
            }

            return '<div class="machine-card ' + status + '">' +
                '<div class="card-header">' +
                    '<span class="hostname">' + machine.hostname + '</span>' +
                    '<div class="badges">' + swarmBadge + statusBadge + '</div>' +
                '</div>' +
                '<div class="metrics">' +
                    '<div class="metric">' +
                        '<span class="metric-label">CPU</span>' +
                        '<div class="bar-container">' +
                            '<div class="bar-fill ' + getBarClass('cpu', m.cpu_percent || 0) + '" style="width: ' + (m.cpu_percent || 0) + '%"></div>' +
                        '</div>' +
                        '<span class="metric-value">' + (m.cpu_percent || 0).toFixed(1) + '%</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">MEM</span>' +
                        '<div class="bar-container">' +
                            '<div class="bar-fill ' + getBarClass('mem', m.memory_percent || 0) + '" style="width: ' + (m.memory_percent || 0) + '%"></div>' +
                        '</div>' +
                        '<span class="metric-value">' + (m.memory_percent || 0).toFixed(1) + '%</span>' +
                    '</div>' +
                    '<div class="metric">' +
                        '<span class="metric-label">DISCO</span>' +
                        '<div class="bar-container">' +
                            '<div class="bar-fill ' + getBarClass('disk', m.disk_percent || 0) + '" style="width: ' + (m.disk_percent || 0) + '%"></div>' +
                        '</div>' +
                        '<span class="metric-value">' + (m.disk_percent || 0).toFixed(1) + '%</span>' +
                    '</div>' +
                '</div>' +
                '<div class="docker-info">' +
                    '<span class="docker-stat"><span class="docker-up">' + (m.docker_running || 0) + '</span> rodando</span>' +
                    '<span class="docker-stat"><span class="docker-down">' + (m.docker_stopped || 0) + '</span> parados</span>' +
                '</div>' +
                '<div class="card-footer">' +
                    '<span>' + (machine.ip || 'IP desconhecido') + '</span>' +
                    '<span>' + formatTime(machine.last_seen) + '</span>' +
                '</div>' +
            '</div>';
        }

        function renderEmptyState() {
            const serverUrl = window.location.origin;
            return '<div class="empty-state">' +
                '<h2>Nenhuma máquina cadastrada</h2>' +
                '<p>Instale o agent em suas VPS para começar o monitoramento:</p>' +
                '<code>curl -sSL ' + serverUrl + '/install.sh | bash -s -- --server ' + serverUrl + ' --token SEU_TOKEN --name "minha-vps"</code>' +
            '</div>';
        }

        function groupMachines(machines) {
            const groups = {};
            machines.forEach(function(m) {
                const group = m.group || 'default';
                if (!groups[group]) groups[group] = [];
                groups[group].push(m);
            });

            // Ordenar cada grupo: offline primeiro, depois por nome
            Object.keys(groups).forEach(function(key) {
                groups[key].sort(function(a, b) {
                    const statusA = getStatusClass(a);
                    const statusB = getStatusClass(b);

                    // Offline primeiro
                    if (statusA === 'offline' && statusB !== 'offline') return -1;
                    if (statusA !== 'offline' && statusB === 'offline') return 1;

                    // Warning depois
                    if (statusA === 'warning' && statusB === 'online') return -1;
                    if (statusA === 'online' && statusB === 'warning') return 1;

                    // Por nome
                    return a.hostname.localeCompare(b.hostname);
                });
            });

            return groups;
        }

        function render(data) {
            const main = document.getElementById('main');
            const machines = data.machines || [];

            // Atualizar stats
            let online = 0, warning = 0, offline = 0, containers = 0;
            machines.forEach(function(m) {
                const status = getStatusClass(m);
                if (status === 'online') online++;
                else if (status === 'warning') warning++;
                else offline++;

                if (m.metrics) containers += m.metrics.docker_running || 0;
            });

            document.getElementById('stat-online').textContent = online;
            document.getElementById('stat-warning').textContent = warning;
            document.getElementById('stat-offline').textContent = offline;
            document.getElementById('stat-containers').textContent = containers;

            // Renderizar máquinas
            if (machines.length === 0) {
                main.innerHTML = renderEmptyState();
                return;
            }

            const groups = groupMachines(machines);
            let html = '';

            const sortedGroups = Object.keys(groups).sort();
            sortedGroups.forEach(function(groupName) {
                html += '<div class="group">' +
                    '<div class="group-header">' +
                        '<span>📁</span> ' + groupName + ' (' + groups[groupName].length + ')' +
                    '</div>' +
                    '<div class="machines-grid">';

                groups[groupName].forEach(function(machine) {
                    html += renderMachine(machine);
                });

                html += '</div></div>';
            });

            main.innerHTML = html;
        }

        function updateTime() {
            const now = new Date();
            const time = now.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
            document.getElementById('update-time').textContent = 'Atualizado: ' + time;
        }

        async function fetchData() {
            try {
                const response = await fetch('/api/machines');
                const data = await response.json();
                render(data);
                updateTime();
            } catch (error) {
                console.error('Erro ao carregar dados:', error);
                document.getElementById('main').innerHTML =
                    '<div class="empty-state"><h2>Erro ao carregar dados</h2><p>' + error.message + '</p></div>';
            }
        }

        // Inicializar
        fetchData();
        setInterval(fetchData, REFRESH_INTERVAL);
    </script>
</body>
</html>`
}

// GetInstallScript retorna o script de instalação do agent
func GetInstallScript() string {
	return `#!/bin/bash
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
ExecStart=${INSTALL_DIR}/monitor-agent \\
    --server \${SERVER_URL} \\
    --token \${AUTH_TOKEN} \\
    --name \${MACHINE_NAME} \\
    --group \${GROUP_NAME} \\
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
`
}
