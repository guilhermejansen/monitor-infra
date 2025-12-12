package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Storage representa a conexão com o banco de dados SQLite
type Storage struct {
	db            *sql.DB
	retentionDays int
}

// Machine representa uma máquina cadastrada
type Machine struct {
	ID         int64     `json:"id"`
	Hostname   string    `json:"hostname"`
	IP         string    `json:"ip"`
	GroupName  string    `json:"group"`
	SwarmRole  string    `json:"swarm_role"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	IsOnline   bool      `json:"is_online"`
	Metrics    *Metrics  `json:"metrics,omitempty"`
}

// Metrics representa as métricas coletadas
type Metrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	DockerRunning int     `json:"docker_running"`
	DockerStopped int     `json:"docker_stopped"`
}

// MetricPayload representa o payload recebido do agent
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

// parseDateTime tenta fazer parse de datetime em múltiplos formatos do SQLite
func parseDateTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	// Formatos comuns do SQLite
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}

// New cria uma nova conexão com o banco de dados
func New(dbPath string, retentionDays int) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco de dados: %w", err)
	}

	// Configurar pool de conexões
	db.SetMaxOpenConns(1) // SQLite funciona melhor com uma conexão
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	storage := &Storage{
		db:            db,
		retentionDays: retentionDays,
	}

	// Inicializar schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("erro ao inicializar schema: %w", err)
	}

	return storage, nil
}

// initSchema cria as tabelas se não existirem
func (s *Storage) initSchema() error {
	schema := `
	-- Máquinas cadastradas automaticamente
	CREATE TABLE IF NOT EXISTS machines (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		hostname    TEXT UNIQUE NOT NULL,
		ip          TEXT,
		group_name  TEXT DEFAULT 'default',
		swarm_role  TEXT DEFAULT 'none',
		first_seen  DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_seen   DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Métricas coletadas
	CREATE TABLE IF NOT EXISTS metrics (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		machine_id      INTEGER NOT NULL,
		collected_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		cpu_percent     REAL,
		memory_percent  REAL,
		disk_percent    REAL,
		docker_running  INTEGER DEFAULT 0,
		docker_stopped  INTEGER DEFAULT 0,
		FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE CASCADE
	);

	-- Índices para performance
	CREATE INDEX IF NOT EXISTS idx_metrics_machine_time ON metrics(machine_id, collected_at DESC);
	CREATE INDEX IF NOT EXISTS idx_machines_hostname ON machines(hostname);
	CREATE INDEX IF NOT EXISTS idx_metrics_collected ON metrics(collected_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// UpsertMachine cria ou atualiza uma máquina e retorna seu ID
func (s *Storage) UpsertMachine(hostname, ip, groupName, swarmRole string) (int64, error) {
	// Tentar inserir
	result, err := s.db.Exec(`
		INSERT INTO machines (hostname, ip, group_name, swarm_role, last_seen)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(hostname) DO UPDATE SET
			ip = excluded.ip,
			group_name = COALESCE(NULLIF(excluded.group_name, ''), group_name),
			swarm_role = excluded.swarm_role,
			last_seen = CURRENT_TIMESTAMP
	`, hostname, ip, groupName, swarmRole)

	if err != nil {
		return 0, fmt.Errorf("erro ao upsert máquina: %w", err)
	}

	// Se foi insert, retorna o ID inserido
	id, err := result.LastInsertId()
	if err == nil && id > 0 {
		return id, nil
	}

	// Se foi update, busca o ID existente
	var machineID int64
	err = s.db.QueryRow("SELECT id FROM machines WHERE hostname = ?", hostname).Scan(&machineID)
	if err != nil {
		return 0, fmt.Errorf("erro ao buscar ID da máquina: %w", err)
	}

	return machineID, nil
}

// InsertMetrics insere novas métricas para uma máquina
func (s *Storage) InsertMetrics(machineID int64, m *Metrics) error {
	_, err := s.db.Exec(`
		INSERT INTO metrics (machine_id, cpu_percent, memory_percent, disk_percent, docker_running, docker_stopped)
		VALUES (?, ?, ?, ?, ?, ?)
	`, machineID, m.CPUPercent, m.MemoryPercent, m.DiskPercent, m.DockerRunning, m.DockerStopped)

	if err != nil {
		return fmt.Errorf("erro ao inserir métricas: %w", err)
	}

	return nil
}

// SaveMetrics salva métricas completas (upsert machine + insert metrics)
func (s *Storage) SaveMetrics(payload *MetricPayload) (int64, error) {
	// Upsert da máquina
	machineID, err := s.UpsertMachine(payload.Hostname, payload.IP, payload.GroupName, payload.SwarmRole)
	if err != nil {
		return 0, err
	}

	// Inserir métricas
	metrics := &Metrics{
		CPUPercent:    payload.CPUPercent,
		MemoryPercent: payload.MemoryPercent,
		DiskPercent:   payload.DiskPercent,
		DockerRunning: payload.DockerRunning,
		DockerStopped: payload.DockerStopped,
	}

	if err := s.InsertMetrics(machineID, metrics); err != nil {
		return 0, err
	}

	return machineID, nil
}

// GetMachinesWithMetrics retorna todas as máquinas com suas últimas métricas
func (s *Storage) GetMachinesWithMetrics() ([]Machine, error) {
	query := `
		SELECT
			m.id, m.hostname, m.ip, m.group_name, m.swarm_role,
			m.first_seen, m.last_seen,
			COALESCE(met.cpu_percent, 0),
			COALESCE(met.memory_percent, 0),
			COALESCE(met.disk_percent, 0),
			COALESCE(met.docker_running, 0),
			COALESCE(met.docker_stopped, 0)
		FROM machines m
		LEFT JOIN (
			SELECT machine_id, cpu_percent, memory_percent, disk_percent,
				   docker_running, docker_stopped,
				   ROW_NUMBER() OVER (PARTITION BY machine_id ORDER BY collected_at DESC) as rn
			FROM metrics
		) met ON m.id = met.machine_id AND met.rn = 1
		ORDER BY m.group_name, m.hostname
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar máquinas: %w", err)
	}
	defer rows.Close()

	var machines []Machine
	now := time.Now()
	onlineThreshold := 70 * time.Minute // 1h + 10min de margem

	for rows.Next() {
		var m Machine
		var metrics Metrics
		var firstSeen, lastSeen string

		err := rows.Scan(
			&m.ID, &m.Hostname, &m.IP, &m.GroupName, &m.SwarmRole,
			&firstSeen, &lastSeen,
			&metrics.CPUPercent, &metrics.MemoryPercent, &metrics.DiskPercent,
			&metrics.DockerRunning, &metrics.DockerStopped,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao escanear linha: %w", err)
		}

		// Parse das datas (tentar múltiplos formatos do SQLite)
		m.FirstSeen = parseDateTime(firstSeen)
		m.LastSeen = parseDateTime(lastSeen)

		// Determinar se está online
		m.IsOnline = now.Sub(m.LastSeen) < onlineThreshold

		m.Metrics = &metrics
		machines = append(machines, m)
	}

	return machines, rows.Err()
}

// GetMachineByID retorna uma máquina específica com suas métricas
func (s *Storage) GetMachineByID(id int64) (*Machine, error) {
	query := `
		SELECT
			m.id, m.hostname, m.ip, m.group_name, m.swarm_role,
			m.first_seen, m.last_seen,
			COALESCE(met.cpu_percent, 0),
			COALESCE(met.memory_percent, 0),
			COALESCE(met.disk_percent, 0),
			COALESCE(met.docker_running, 0),
			COALESCE(met.docker_stopped, 0)
		FROM machines m
		LEFT JOIN (
			SELECT machine_id, cpu_percent, memory_percent, disk_percent,
				   docker_running, docker_stopped
			FROM metrics
			WHERE machine_id = ?
			ORDER BY collected_at DESC
			LIMIT 1
		) met ON m.id = met.machine_id
		WHERE m.id = ?
	`

	var m Machine
	var metrics Metrics
	var firstSeen, lastSeen string

	err := s.db.QueryRow(query, id, id).Scan(
		&m.ID, &m.Hostname, &m.IP, &m.GroupName, &m.SwarmRole,
		&firstSeen, &lastSeen,
		&metrics.CPUPercent, &metrics.MemoryPercent, &metrics.DiskPercent,
		&metrics.DockerRunning, &metrics.DockerStopped,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar máquina: %w", err)
	}

	// Parse das datas (tentar múltiplos formatos do SQLite)
	m.FirstSeen = parseDateTime(firstSeen)
	m.LastSeen = parseDateTime(lastSeen)

	// Determinar se está online
	now := time.Now()
	onlineThreshold := 70 * time.Minute
	m.IsOnline = now.Sub(m.LastSeen) < onlineThreshold

	m.Metrics = &metrics
	return &m, nil
}

// GetMetricsHistory retorna o histórico de métricas de uma máquina
func (s *Storage) GetMetricsHistory(machineID int64, hours int) ([]map[string]interface{}, error) {
	query := `
		SELECT collected_at, cpu_percent, memory_percent, disk_percent,
			   docker_running, docker_stopped
		FROM metrics
		WHERE machine_id = ? AND collected_at > datetime('now', ?)
		ORDER BY collected_at DESC
	`

	rows, err := s.db.Query(query, machineID, fmt.Sprintf("-%d hours", hours))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar histórico: %w", err)
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var collectedAt string
		var cpu, mem, disk float64
		var running, stopped int

		if err := rows.Scan(&collectedAt, &cpu, &mem, &disk, &running, &stopped); err != nil {
			return nil, err
		}

		history = append(history, map[string]interface{}{
			"collected_at":   collectedAt,
			"cpu_percent":    cpu,
			"memory_percent": mem,
			"disk_percent":   disk,
			"docker_running": running,
			"docker_stopped": stopped,
		})
	}

	return history, rows.Err()
}

// CleanupOldMetrics remove métricas antigas baseado na política de retenção
func (s *Storage) CleanupOldMetrics() (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM metrics
		WHERE collected_at < datetime('now', ?)
	`, fmt.Sprintf("-%d days", s.retentionDays))

	if err != nil {
		return 0, fmt.Errorf("erro ao limpar métricas antigas: %w", err)
	}

	return result.RowsAffected()
}

// GetStats retorna estatísticas gerais
func (s *Storage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total de máquinas
	var totalMachines int
	s.db.QueryRow("SELECT COUNT(*) FROM machines").Scan(&totalMachines)
	stats["total_machines"] = totalMachines

	// Máquinas online (última métrica < 70 min)
	var onlineMachines int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM machines
		WHERE last_seen > datetime('now', '-70 minutes')
	`).Scan(&onlineMachines)
	stats["online"] = onlineMachines

	// Máquinas com atenção (CPU/MEM/Disco > 85%)
	var warningMachines int
	s.db.QueryRow(`
		SELECT COUNT(DISTINCT m.id) FROM machines m
		JOIN (
			SELECT machine_id, cpu_percent, memory_percent, disk_percent,
				   ROW_NUMBER() OVER (PARTITION BY machine_id ORDER BY collected_at DESC) as rn
			FROM metrics
		) met ON m.id = met.machine_id AND met.rn = 1
		WHERE met.cpu_percent > 85 OR met.memory_percent > 85 OR met.disk_percent > 85
	`).Scan(&warningMachines)
	stats["warning"] = warningMachines

	// Máquinas offline
	stats["offline"] = totalMachines - onlineMachines

	// Total de containers rodando
	var totalContainers int
	s.db.QueryRow(`
		SELECT COALESCE(SUM(met.docker_running), 0) FROM machines m
		JOIN (
			SELECT machine_id, docker_running,
				   ROW_NUMBER() OVER (PARTITION BY machine_id ORDER BY collected_at DESC) as rn
			FROM metrics
		) met ON m.id = met.machine_id AND met.rn = 1
	`).Scan(&totalContainers)
	stats["total_containers"] = totalContainers

	return stats, nil
}

// Close fecha a conexão com o banco de dados
func (s *Storage) Close() error {
	return s.db.Close()
}
