package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Metrics representa todas as métricas coletadas
type Metrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	DockerRunning int     `json:"docker_running"`
	DockerStopped int     `json:"docker_stopped"`
	SwarmRole     string  `json:"swarm_role"`
}

// Collector é responsável por coletar métricas do sistema
type Collector struct {
	dockerClient *client.Client
}

// New cria um novo collector
func New() *Collector {
	c := &Collector{}

	// Tentar conectar ao Docker (não é obrigatório)
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err == nil {
		// Testar conexão
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = dockerCli.Ping(ctx)
		if err == nil {
			c.dockerClient = dockerCli
		}
	}

	return c
}

// CollectAll coleta todas as métricas disponíveis
func (c *Collector) CollectAll() (*Metrics, error) {
	metrics := &Metrics{
		SwarmRole: "none",
	}

	// Coletar CPU
	cpu, err := c.CollectCPU()
	if err == nil {
		metrics.CPUPercent = cpu
	}

	// Coletar Memória
	mem, err := c.CollectMemory()
	if err == nil {
		metrics.MemoryPercent = mem
	}

	// Coletar Disco
	disk, err := c.CollectDisk()
	if err == nil {
		metrics.DiskPercent = disk
	}

	// Coletar Docker (se disponível)
	if c.dockerClient != nil {
		running, stopped, swarmRole, err := c.CollectDocker()
		if err == nil {
			metrics.DockerRunning = running
			metrics.DockerStopped = stopped
			metrics.SwarmRole = swarmRole
		}
	}

	return metrics, nil
}

// CollectCPU coleta o percentual de uso de CPU
func (c *Collector) CollectCPU() (float64, error) {
	// Primeira leitura
	idle1, total1, err := readCPUStat()
	if err != nil {
		return 0, err
	}

	// Aguardar 1 segundo
	time.Sleep(1 * time.Second)

	// Segunda leitura
	idle2, total2, err := readCPUStat()
	if err != nil {
		return 0, err
	}

	// Calcular percentual
	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)

	if totalDelta == 0 {
		return 0, nil
	}

	cpuPercent := 100 * (1 - idleDelta/totalDelta)
	return cpuPercent, nil
}

// readCPUStat lê /proc/stat e retorna idle e total
func readCPUStat() (idle, total uint64, err error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			return 0, 0, fmt.Errorf("formato inesperado em /proc/stat")
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0, fmt.Errorf("campos insuficientes em /proc/stat")
		}

		// cpu user nice system idle iowait irq softirq steal guest guest_nice
		var values []uint64
		for i := 1; i < len(fields); i++ {
			val, _ := strconv.ParseUint(fields[i], 10, 64)
			values = append(values, val)
			total += val
		}

		// idle é o 4º campo (índice 3)
		if len(values) > 3 {
			idle = values[3]
		}
	}

	return idle, total, scanner.Err()
}

// CollectMemory coleta o percentual de uso de memória
func (c *Collector) CollectMemory() (float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, _ := strconv.ParseUint(fields[1], 10, 64)

		switch fields[0] {
		case "MemTotal:":
			memTotal = value
		case "MemAvailable:":
			memAvailable = value
		}

		// Se já temos os dois valores, podemos parar
		if memTotal > 0 && memAvailable > 0 {
			break
		}
	}

	if memTotal == 0 {
		return 0, fmt.Errorf("não foi possível ler MemTotal")
	}

	memUsed := memTotal - memAvailable
	memPercent := float64(memUsed) / float64(memTotal) * 100

	return memPercent, nil
}

// CollectDisk coleta o percentual de uso de disco (partição root)
func (c *Collector) CollectDisk() (float64, error) {
	var stat syscall.Statfs_t

	err := syscall.Statfs("/", &stat)
	if err != nil {
		return 0, err
	}

	// Calcular espaço total e disponível em bytes
	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)

	if total == 0 {
		return 0, fmt.Errorf("disco total é zero")
	}

	used := total - available
	diskPercent := float64(used) / float64(total) * 100

	return diskPercent, nil
}

// CollectDocker coleta informações sobre containers Docker
func (c *Collector) CollectDocker() (running, stopped int, swarmRole string, err error) {
	if c.dockerClient == nil {
		return 0, 0, "none", fmt.Errorf("cliente Docker não disponível")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Listar todos os containers (incluindo parados)
	containers, err := c.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return 0, 0, "none", fmt.Errorf("erro ao listar containers: %w", err)
	}

	// Contar containers por status
	for _, cont := range containers {
		if cont.State == "running" {
			running++
		} else {
			stopped++
		}
	}

	// Verificar status do Swarm
	swarmRole = "none"
	info, err := c.dockerClient.Info(ctx)
	if err == nil {
		swarm := info.Swarm
		if swarm.LocalNodeState == "active" {
			if swarm.ControlAvailable {
				swarmRole = "manager"
			} else {
				swarmRole = "worker"
			}
		}
	}

	return running, stopped, swarmRole, nil
}

// GetContainerList retorna a lista de containers com detalhes
func (c *Collector) GetContainerList() ([]map[string]string, error) {
	if c.dockerClient == nil {
		return nil, fmt.Errorf("cliente Docker não disponível")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containers, err := c.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var result []map[string]string
	for _, cont := range containers {
		name := ""
		if len(cont.Names) > 0 {
			name = strings.TrimPrefix(cont.Names[0], "/")
		}

		result = append(result, map[string]string{
			"id":     cont.ID[:12],
			"name":   name,
			"image":  cont.Image,
			"status": cont.State,
		})
	}

	return result, nil
}

// GetSystemInfo retorna informações do sistema
func (c *Collector) GetSystemInfo() map[string]string {
	info := make(map[string]string)

	// Hostname
	hostname, err := os.Hostname()
	if err == nil {
		info["hostname"] = hostname
	}

	// OS Release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				info["os"] = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	// Kernel
	if data, err := os.ReadFile("/proc/version"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			info["kernel"] = fields[2]
		}
	}

	// Uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 1 {
			if uptime, err := strconv.ParseFloat(fields[0], 64); err == nil {
				info["uptime_seconds"] = fmt.Sprintf("%.0f", uptime)
			}
		}
	}

	return info
}

// Close fecha o cliente Docker
func (c *Collector) Close() error {
	if c.dockerClient != nil {
		return c.dockerClient.Close()
	}
	return nil
}
