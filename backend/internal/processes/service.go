package processes

import (
	"context"
	"fmt"
	"sort"
	"time"

	gprocess "github.com/shirou/gopsutil/v4/process"

	"port-master/backend/internal/ports"
)

type Service struct {
	ports *ports.Service
}

func NewService(portService *ports.Service) *Service {
	return &Service{ports: portService}
}

func (s *Service) ListAllProcesses(ctx context.Context) ([]ProcessInfo, error) {
	portCounts, err := s.portCounts(ctx)
	if err != nil {
		return nil, err
	}

	procs, err := gprocess.Processes()
	if err != nil {
		return nil, err
	}

	result := make([]ProcessInfo, 0, len(procs))
	for _, proc := range procs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info := processInfo(proc)
		info.PortCount = portCounts[info.PID]
		result = append(result, info)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].PID < result[j].PID })
	return result, nil
}

func (s *Service) Detail(ctx context.Context, pid int64) (ProcessDetail, error) {
	proc, err := gprocess.NewProcess(int32(pid))
	if err != nil {
		return ProcessDetail{}, err
	}
	info := processInfo(proc)
	path, _ := proc.Exe()
	if path == "" {
		path = info.CommandLine
	}
	createTime := ""
	if ms, err := proc.CreateTime(); err == nil && ms > 0 {
		createTime = time.UnixMilli(ms).Format("2006-01-02 15:04:05")
	}
	boundPorts, err := s.ports.QueryByPID(ctx, pid)
	if err != nil {
		return ProcessDetail{}, err
	}
	return ProcessDetail{
		PID:           info.PID,
		ProcessName:   info.ProcessName,
		ProgramPath:   path,
		CommandLine:   info.CommandLine,
		CPUPercent:    info.CPUPercent,
		MemoryPercent: info.MemoryPercent,
		MemoryUsage:   info.MemoryUsage,
		CreateTime:    createTime,
		BoundPorts:    boundPorts,
	}, nil
}

func (s *Service) KillByPort(ctx context.Context, port int, force bool) ([]string, error) {
	matches, err := s.ports.QueryByPort(ctx, port)
	if err != nil {
		return nil, err
	}
	pidSet := map[int64]struct{}{}
	for _, item := range matches {
		if item.PID != nil {
			pidSet[*item.PID] = struct{}{}
		}
	}
	if len(pidSet) == 0 {
		return []string{fmt.Sprintf("port %d is not currently occupied", port)}, nil
	}
	pids := make([]int64, 0, len(pidSet))
	for pid := range pidSet {
		pids = append(pids, pid)
	}
	sort.Slice(pids, func(i, j int) bool { return pids[i] < pids[j] })

	results := []string{fmt.Sprintf("port %d is associated with %d processes", port, len(pids))}
	results = append(results, s.KillProcesses(pids, force)...)
	return results, nil
}

func (s *Service) KillProcesses(pids []int64, force bool) []string {
	results := make([]string, 0, len(pids))
	for _, pid := range pids {
		if err := KillProcess(pid, force); err != nil {
			results = append(results, fmt.Sprintf("PID %d: failed - %s", pid, err.Error()))
			continue
		}
		results = append(results, fmt.Sprintf("PID %d: success", pid))
	}
	return results
}

func KillProcess(pid int64, force bool) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	return terminateProcess(pid, force)
}

func (s *Service) portCounts(ctx context.Context) (map[int64]int, error) {
	all, err := s.ports.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	counts := map[int64]int{}
	for _, item := range all {
		if item.PID != nil {
			counts[*item.PID]++
		}
	}
	return counts, nil
}

func processInfo(proc *gprocess.Process) ProcessInfo {
	pid := int64(proc.Pid)
	name, _ := proc.Name()
	cmd, _ := proc.Cmdline()
	if cmd == "" {
		cmd = name
	}
	cpuPercent, _ := proc.CPUPercent()
	memPercent, _ := proc.MemoryPercent()
	memUsage := ""
	if memInfo, err := proc.MemoryInfo(); err == nil && memInfo != nil {
		memUsage = fmt.Sprintf("%.1f MB", float64(memInfo.RSS)/(1024.0*1024.0))
	}
	return ProcessInfo{
		PID:           pid,
		ProcessName:   name,
		CommandLine:   cmd,
		CPUPercent:    cpuPercent,
		MemoryPercent: memPercent,
		MemoryUsage:   memUsage,
	}
}
