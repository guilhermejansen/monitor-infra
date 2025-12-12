package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monitor-infra/internal/collector"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

// Config representa a configuração do agent
type Config struct {
	ServerURL    string
	Token        string
	MachineName  string
	GroupName    string
	IntervalMins int
}

// MetricPayload representa o payload enviado ao servidor
type MetricPayload struct {
	Hostname      string  `json:"hostname"`
	IP            string  `json:"ip"`
	GroupName     string  `json:"group"`
	SwarmRole     string  `json:"swarm_role"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	DockerRunning int     `json:"docker_running"`
	DockerStopped int     `json:"docker_stopped"`
}

func main() {
	// Flags de linha de comando
	serverURL := flag.String("server", getEnv("SERVER_URL", "http://localhost:8080"), "URL do servidor de monitoramento")
	token := flag.String("token", getEnv("AUTH_TOKEN", ""), "Token de autenticação")
	machineName := flag.String("name", getEnv("MACHINE_NAME", ""), "Nome da máquina (default: hostname)")
	groupName := flag.String("group", getEnv("GROUP_NAME", "default"), "Nome do grupo")
	intervalMins := flag.Int("interval", getEnvInt("INTERVAL_MINUTES", 60), "Intervalo de coleta em minutos")
	version := flag.Bool("version", false, "Mostrar versão")
	once := flag.Bool("once", false, "Executar apenas uma vez e sair")

	flag.Parse()

	if *version {
		fmt.Printf("Monitor-Infra Agent v%s (build: %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Validar configuração
	if *serverURL == "" {
		log.Fatal("Erro: URL do servidor é obrigatória (--server ou SERVER_URL)")
	}

	// Obter hostname se não especificado
	hostname := *machineName
	if hostname == "" {
		var err error
		hostname, err = os.Hostname()
		if err != nil {
			log.Fatalf("Erro ao obter hostname: %v", err)
		}
	}

	config := &Config{
		ServerURL:    *serverURL,
		Token:        *token,
		MachineName:  hostname,
		GroupName:    *groupName,
		IntervalMins: *intervalMins,
	}

	log.Printf("Monitor-Infra Agent v%s iniciando...", Version)
	log.Printf("Servidor: %s", config.ServerURL)
	log.Printf("Máquina: %s (grupo: %s)", config.MachineName, config.GroupName)
	log.Printf("Intervalo: %d minutos", config.IntervalMins)

	// Criar collector
	coll := collector.New()
	defer coll.Close()

	// Se modo "once", executar uma vez e sair
	if *once {
		if err := collectAndSend(config, coll); err != nil {
			log.Fatalf("Erro: %v", err)
		}
		log.Println("Métricas enviadas com sucesso!")
		return
	}

	// Enviar primeira métrica imediatamente
	log.Println("Enviando primeira coleta...")
	if err := collectAndSend(config, coll); err != nil {
		log.Printf("Aviso: falha na primeira coleta: %v", err)
	} else {
		log.Println("Primeira coleta enviada com sucesso!")
	}

	// Configurar ticker para coletas periódicas
	ticker := time.NewTicker(time.Duration(config.IntervalMins) * time.Minute)
	defer ticker.Stop()

	// Configurar signal handler para graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Agent rodando. Próxima coleta em %d minutos...", config.IntervalMins)

	// Loop principal
	for {
		select {
		case <-ticker.C:
			log.Println("Iniciando coleta...")
			if err := collectAndSend(config, coll); err != nil {
				log.Printf("Erro na coleta: %v", err)
			} else {
				log.Println("Métricas enviadas com sucesso!")
			}
			log.Printf("Próxima coleta em %d minutos...", config.IntervalMins)

		case sig := <-sigChan:
			log.Printf("Sinal recebido: %v. Encerrando...", sig)
			return
		}
	}
}

// collectAndSend coleta métricas e envia para o servidor
func collectAndSend(config *Config, coll *collector.Collector) error {
	// Coletar métricas
	metrics, err := coll.CollectAll()
	if err != nil {
		return fmt.Errorf("erro ao coletar métricas: %w", err)
	}

	// Obter IP local
	ip := getLocalIP()

	// Montar payload
	payload := &MetricPayload{
		Hostname:      config.MachineName,
		IP:            ip,
		GroupName:     config.GroupName,
		SwarmRole:     metrics.SwarmRole,
		CPUPercent:    metrics.CPUPercent,
		MemoryPercent: metrics.MemoryPercent,
		DiskPercent:   metrics.DiskPercent,
		DockerRunning: metrics.DockerRunning,
		DockerStopped: metrics.DockerStopped,
	}

	// Enviar para servidor
	return sendMetrics(config, payload)
}

// sendMetrics envia métricas para o servidor
func sendMetrics(config *Config, payload *MetricPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload: %w", err)
	}

	// Criar request
	url := config.ServerURL + "/api/metrics"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+config.Token)
	}

	// Enviar com timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar request: %w", err)
	}
	defer resp.Body.Close()

	// Verificar resposta
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("servidor retornou status %d", resp.StatusCode)
	}

	return nil
}

// getLocalIP obtém o IP local da máquina
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

// getEnv obtém variável de ambiente com valor default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt obtém variável de ambiente como int com valor default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := parseInt(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// parseInt converte string para int
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
