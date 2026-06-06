package system

import (
	"context"
	"math"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"

	"port-master/backend/internal/ports"
)

type Service struct {
	ports        *ports.Service
	cpuProvider  cpuUsageProvider
	cpuMu        sync.Mutex
	lastCPU      *cpu.TimesStat
	lastCPUUsage float64
}

func NewService(portService *ports.Service) *Service {
	service := &Service{
		ports:       portService,
		cpuProvider: newCPUUsageProvider(),
	}
	if times, err := cpu.Times(false); err == nil && len(times) > 0 {
		initial := times[0]
		service.lastCPU = &initial
	}
	return service
}

func (s *Service) Stats(ctx context.Context) (Stats, error) {
	allPorts, err := s.ports.ScanAllPorts(ctx)
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{
		OSType:        runtime.GOOS,
		NeedAdminHint: needAdminHint(),
	}
	for _, item := range allPorts {
		if strings.EqualFold(item.State, "LISTEN") {
			stats.ListenPortCount++
		}
		if strings.EqualFold(item.State, "ESTABLISHED") {
			stats.ActiveConnectionCount++
		}
	}

	stats.CPUUsage = s.cpuUsage(ctx)
	if virtualMemory, err := mem.VirtualMemory(); err == nil && virtualMemory != nil {
		stats.MemoryUsage = virtualMemory.UsedPercent
		stats.MemoryUsedMB = float64(virtualMemory.Used) / (1024.0 * 1024.0)
		stats.MemoryTotalMB = float64(virtualMemory.Total) / (1024.0 * 1024.0)
	}
	if procs, err := process.Processes(); err == nil {
		stats.ProcessCount = len(procs)
	}

	return stats, nil
}

func (s *Service) cpuUsage(ctx context.Context) float64 {
	if s.cpuProvider != nil {
		if usage, ok := s.cpuProvider.Usage(ctx); ok {
			s.lastCPUUsage = usage
			return usage
		}
	}

	times, err := cpu.TimesWithContext(ctx, false)
	if err != nil || len(times) == 0 {
		return s.lastCPUUsage
	}

	current := times[0]
	s.cpuMu.Lock()
	defer s.cpuMu.Unlock()

	if s.lastCPU == nil {
		s.lastCPU = &current
		return s.lastCPUUsage
	}

	usage := calculateCPUUsage(*s.lastCPU, current)
	s.lastCPU = &current
	if math.IsNaN(usage) || math.IsInf(usage, 0) || usage < 0 {
		return s.lastCPUUsage
	}
	if usage > 100 {
		usage = 100
	}
	s.lastCPUUsage = usage
	return usage
}

func calculateCPUUsage(previous, current cpu.TimesStat) float64 {
	totalDelta := current.Total() - previous.Total()
	idleDelta := (current.Idle + current.Iowait) - (previous.Idle + previous.Iowait)
	if totalDelta <= 0 {
		return 0
	}
	return (totalDelta - idleDelta) / totalDelta * 100
}

type cpuUsageProvider interface {
	Usage(ctx context.Context) (float64, bool)
}

func Info(authRequired bool) map[string]interface{} {
	return map[string]interface{}{
		"osType":         runtime.GOOS,
		"osCategory":     osCategory(),
		"goVersion":      runtime.Version(),
		"authRequired":   authRequired,
		"permissionHint": permissionHint(),
	}
}

func osCategory() string {
	switch runtime.GOOS {
	case "windows":
		return "WINDOWS"
	case "darwin":
		return "MACOS"
	case "linux":
		return "LINUX"
	default:
		return "OTHER"
	}
}

func permissionHint() string {
	if runtime.GOOS == "windows" {
		return "Windows 可能需要以管理员身份运行 Port Master，才能结束进程或读取完整程序路径。"
	}
	return "Linux/macOS 可能需要 root 或 sudo 权限，才能结束受保护进程或读取完整程序路径。"
}

func needAdminHint() bool {
	if runtime.GOOS == "windows" {
		return exec.Command("net", "session").Run() != nil
	}
	current, err := user.Current()
	if err != nil {
		return true
	}
	return current.Username != "root"
}
