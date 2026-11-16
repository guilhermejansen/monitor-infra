package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"monitor-infra/internal/dashboard"
	"monitor-infra/internal/storage"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

// Config representa a configuração do servidor
type Config struct {
	Port          int
	DBPath        string
	Token         string
	RetentionDays int
	InsecureDev   bool
}

// Server representa o servidor HTTP
type Server struct {
	config  *Config
	storage *storage.Storage
	mux     *http.ServeMux
}

func main() {
	// Flags de linha de comando
	port := flag.Int("port", getEnvInt("PORT", 8080), "Porta do servidor")
	dbPath := flag.String("db", getEnv("DB_PATH", "./data/monitor.db"), "Caminho do banco de dados SQLite")
	token := flag.String("token", getEnv("AUTH_TOKEN", ""), "Token de autenticação")
	retentionDays := flag.Int("retention", getEnvInt("RETENTION_DAYS", 90), "Dias de retenção de métricas")
	insecureDev := flag.Bool("insecure-dev", false, "Desativar autenticação somente para desenvolvimento em loopback")
	version := flag.Bool("version", false, "Mostrar versão")

	flag.Parse()

	if *version {
		fmt.Printf("Monitor-Infra Server v%s (build: %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	config := &Config{
		Port:          *port,
		DBPath:        *dbPath,
		Token:         *token,
		RetentionDays: *retentionDays,
		InsecureDev:   *insecureDev,
	}

	if strings.TrimSpace(config.Token) == "" && !config.InsecureDev {
		log.Fatal("AUTH_TOKEN/--token é obrigatório; --insecure-dev é apenas para desenvolvimento local")
	}

	// Criar diretório do banco se não existir
	dbDir := filepath.Dir(config.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Erro ao criar diretório do banco: %v", err)
	}

	// Inicializar storage
	store, err := storage.New(config.DBPath, config.RetentionDays)
	if err != nil {
		log.Fatalf("Erro ao inicializar banco de dados: %v", err)
	}
	defer store.Close()

	// Limpeza inicial de métricas antigas
	go func() {
		deleted, err := store.CleanupOldMetrics()
		if err != nil {
			log.Printf("Aviso: erro na limpeza de métricas antigas: %v", err)
		} else if deleted > 0 {
			log.Printf("Limpeza: %d métricas antigas removidas", deleted)
		}
	}()

	// Criar servidor
	server := &Server{
		config:  config,
		storage: store,
		mux:     http.NewServeMux(),
	}

	// Registrar rotas
	server.registerRoutes()

	// O modo sem autenticação nunca escuta fora do loopback.
	address := fmt.Sprintf(":%d", config.Port)
	if config.InsecureDev {
		address = fmt.Sprintf("127.0.0.1:%d", config.Port)
	}

	// Configurar HTTP server
	httpServer := &http.Server{
		Addr:         address,
		Handler:      server.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Iniciar servidor em goroutine
	go func() {
		log.Printf("Monitor-Infra Server v%s iniciando na porta %d...", Version, config.Port)
		if !config.InsecureDev {
			log.Println("Autenticação via token: ATIVADA")
		} else {
			log.Println("AVISO: --insecure-dev, sem autenticação, somente em loopback")
		}
		log.Printf("Dashboard: http://localhost:%d", config.Port)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro no servidor: %v", err)
		}
	}()

	// Agendar limpeza diária
	go server.scheduleDailyCleanup()

	// Aguardar sinal de término
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Encerrando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Erro ao encerrar servidor: %v", err)
	}

	log.Println("Servidor encerrado com sucesso")
}

// registerRoutes registra todas as rotas do servidor
func (s *Server) registerRoutes() {
	// API endpoints
	s.mux.HandleFunc("/api/metrics", s.authMiddleware(s.handleMetrics))
	s.mux.HandleFunc("/api/machines", s.authMiddleware(s.handleMachines))
	s.mux.HandleFunc("/api/machines/", s.authMiddleware(s.handleMachineDetail))
	s.mux.HandleFunc("/api/stats", s.authMiddleware(s.handleStats))
	s.mux.HandleFunc("/api/health", s.authMiddleware(s.handleHealth))

	// Downloads e instalação
	s.mux.HandleFunc("/install.sh", s.handleInstallScript)
	s.mux.HandleFunc("/download/", s.handleDownload)

	// Dashboard
	s.mux.HandleFunc("/", s.authMiddleware(s.handleDashboard))
}

// authMiddleware protege métricas, consultas e painel. Basic permite que o
// navegador reutilize as credenciais do painel nos fetches same-origin da API.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if s.config.InsecureDev {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(host)
			// Não confiar em X-Forwarded-For ou outros headers de proxy.
			if err == nil && ip != nil && ip.IsLoopback() {
				next(w, r)
				return
			}
		} else if strings.TrimSpace(s.config.Token) != "" {
			var supplied string
			auth := r.Header.Get("Authorization")
			if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
				supplied = auth[7:]
			} else if user, password, ok := r.BasicAuth(); ok && user == "monitor" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				supplied = password
			}
			// Hash first to compare fixed-length buffers, including wrong-length tokens.
			expected := sha256.Sum256([]byte(s.config.Token))
			actual := sha256.Sum256([]byte(supplied))
			if subtle.ConstantTimeCompare(expected[:], actual[:]) == 1 {
				next(w, r)
				return
			}
		}

		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("WWW-Authenticate", `Basic realm="Monitor-Infra", charset="UTF-8"`)
		} else {
			w.Header().Set("WWW-Authenticate", `Bearer realm="Monitor-Infra"`)
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

// handleMetrics recebe métricas do agent
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload storage.MetricPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, "Erro ao decodificar payload", http.StatusBadRequest)
		return
	}

	// Validar campos obrigatórios
	if payload.Hostname == "" {
		jsonError(w, "hostname é obrigatório", http.StatusBadRequest)
		return
	}

	// Salvar métricas
	machineID, err := s.storage.SaveMetrics(&payload)
	if err != nil {
		log.Printf("Erro ao salvar métricas: %v", err)
		jsonError(w, "Erro ao salvar métricas", http.StatusInternalServerError)
		return
	}

	log.Printf("Métricas recebidas: %s (ID: %d)", payload.Hostname, machineID)

	// Resposta de sucesso
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"machine_id": machineID,
	})
}

// handleMachines lista todas as máquinas
func (s *Server) handleMachines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	machines, err := s.storage.GetMachinesWithMetrics()
	if err != nil {
		log.Printf("Erro ao buscar máquinas: %v", err)
		jsonError(w, "Erro ao buscar máquinas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"machines": machines,
	})
}

// handleMachineDetail retorna detalhes de uma máquina específica
func (s *Server) handleMachineDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extrair ID da URL
	path := strings.TrimPrefix(r.URL.Path, "/api/machines/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		jsonError(w, "ID da máquina é obrigatório", http.StatusBadRequest)
		return
	}

	machineID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonError(w, "ID inválido", http.StatusBadRequest)
		return
	}

	// Verificar se é pedido de histórico
	if len(parts) > 1 && parts[1] == "metrics" {
		hours := 24
		if h := r.URL.Query().Get("hours"); h != "" {
			if parsed, err := strconv.Atoi(h); err == nil {
				hours = parsed
			}
		}

		history, err := s.storage.GetMetricsHistory(machineID, hours)
		if err != nil {
			log.Printf("Erro ao buscar histórico: %v", err)
			jsonError(w, "Erro ao buscar histórico", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"machine_id": machineID,
			"hours":      hours,
			"metrics":    history,
		})
		return
	}

	// Buscar máquina
	machine, err := s.storage.GetMachineByID(machineID)
	if err != nil {
		log.Printf("Erro ao buscar máquina: %v", err)
		jsonError(w, "Erro ao buscar máquina", http.StatusInternalServerError)
		return
	}

	if machine == nil {
		jsonError(w, "Máquina não encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(machine)
}

// handleStats retorna estatísticas gerais
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := s.storage.GetStats()
	if err != nil {
		log.Printf("Erro ao buscar estatísticas: %v", err)
		jsonError(w, "Erro ao buscar estatísticas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleHealth verifica se o servidor está funcionando
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": Version,
	})
}

// handleDashboard serve o dashboard HTML
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboard.GetHTML()))
}

// handleInstallScript serve o script de instalação
func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(dashboard.GetInstallScript()))
}

// handleDownload serve os binários do agent
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	// Extrair nome do arquivo
	filename := strings.TrimPrefix(r.URL.Path, "/download/")

	// Verificar se é um arquivo válido
	validFiles := map[string]string{
		"agent-linux-amd64": "dist/agent-linux-amd64",
		"agent-linux-arm64": "dist/agent-linux-arm64",
	}

	filePath, ok := validFiles[filename]
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Verificar se o arquivo existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Binary not available. Please build first.", http.StatusNotFound)
		return
	}

	// Servir o arquivo
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	http.ServeFile(w, r, filePath)
}

// scheduleDailyCleanup agenda limpeza diária de métricas antigas
func (s *Server) scheduleDailyCleanup() {
	for {
		// Calcular próxima execução (3h da manhã)
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		time.Sleep(time.Until(next))

		// Executar limpeza
		deleted, err := s.storage.CleanupOldMetrics()
		if err != nil {
			log.Printf("Erro na limpeza programada: %v", err)
		} else {
			log.Printf("Limpeza programada: %d métricas removidas", deleted)
		}
	}
}

// jsonError envia uma resposta de erro em JSON
func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": message,
	})
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
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
