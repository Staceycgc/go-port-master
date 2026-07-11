package ports

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Service struct {
	scanner    Scanner
	cacheTTL   time.Duration
	cacheMu    sync.RWMutex
	cachedScan []PortInfo
	cacheTime  time.Time
}

func NewService() *Service {
	return NewServiceWithOptions(GopsutilScanner{}, 3*time.Second)
}

func NewServiceWithScanner(scanner Scanner) *Service {
	return NewServiceWithOptions(scanner, 0)
}

func NewServiceWithOptions(scanner Scanner, cacheTTL time.Duration) *Service {
	return &Service{scanner: scanner, cacheTTL: cacheTTL}
}

func (s *Service) ScanAllPorts(ctx context.Context) ([]PortInfo, error) {
	return s.scanAllPorts(ctx, false)
}

func (s *Service) ScanAllPortsRefresh(ctx context.Context, forceRefresh bool) ([]PortInfo, error) {
	return s.scanAllPorts(ctx, forceRefresh)
}

func (s *Service) SetCacheTTL(ttl time.Duration) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cacheTTL = ttl
}

func (s *Service) InvalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cachedScan = nil
	s.cacheTime = time.Time{}
}

func (s *Service) scanAllPorts(ctx context.Context, forceRefresh bool) ([]PortInfo, error) {
	s.cacheMu.RLock()
	ttl := s.cacheTTL
	cached := s.cachedScan
	cachedAt := s.cacheTime
	s.cacheMu.RUnlock()

	if !forceRefresh && ttl > 0 && cached != nil && time.Since(cachedAt) < ttl {
		return append([]PortInfo(nil), cached...), nil
	}

	result, err := s.scanner.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	if ttl > 0 {
		s.cacheMu.Lock()
		s.cachedScan = append([]PortInfo(nil), result...)
		s.cacheTime = time.Now()
		s.cacheMu.Unlock()
	}
	return result, nil
}

func (s *Service) QueryByPort(ctx context.Context, port int) ([]PortInfo, error) {
	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]PortInfo, 0)
	for _, item := range all {
		if item.Port == port {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Service) QueryByPorts(ctx context.Context, spec string) ([]PortInfo, error) {
	portSet, err := ParsePortSpec(spec)
	if err != nil {
		return nil, err
	}
	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]PortInfo, 0)
	for _, item := range all {
		if _, ok := portSet[item.Port]; ok {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Service) QueryByRange(ctx context.Context, start, end int) ([]PortInfo, error) {
	min, max, err := normalizeRange(start, end)
	if err != nil {
		return nil, err
	}
	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]PortInfo, 0)
	for _, item := range all {
		if item.Port >= min && item.Port <= max {
			result = append(result, item)
		}
	}
	return result, nil
}

func ParsePortSpec(spec string) (map[int]struct{}, error) {
	result := map[int]struct{}{}
	for _, raw := range strings.Split(spec, ",") {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 || strings.TrimSpace(bounds[0]) == "" || strings.TrimSpace(bounds[1]) == "" {
				return nil, fmt.Errorf("invalid port range %q", part)
			}
			start, err := parsePort(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil, err
			}
			end, err := parsePort(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, err
			}
			min, max, err := normalizeRange(start, end)
			if err != nil {
				return nil, err
			}
			for port := min; port <= max; port++ {
				result[port] = struct{}{}
			}
			continue
		}
		port, err := parsePort(part)
		if err != nil {
			return nil, err
		}
		result[port] = struct{}{}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid ports specified")
	}
	return result, nil
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q", value)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port %d is out of range 1-65535", port)
	}
	return port, nil
}

func normalizeRange(start, end int) (int, int, error) {
	if start < 1 || start > 65535 || end < 1 || end > 65535 {
		return 0, 0, fmt.Errorf("port range must be within 1-65535")
	}
	min, max := start, end
	if min > max {
		min, max = max, min
	}
	if max-min > 10000 {
		return 0, 0, fmt.Errorf("port range cannot exceed 10000")
	}
	return min, max, nil
}

func (s *Service) QueryByProcessName(ctx context.Context, name string) ([]PortInfo, error) {
	keyword := strings.ToLower(strings.TrimSpace(name))
	if keyword == "" {
		return []PortInfo{}, nil
	}
	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]PortInfo, 0)
	for _, item := range all {
		if strings.Contains(strings.ToLower(item.ProcessName), keyword) ||
			strings.Contains(strings.ToLower(item.ProgramPath), keyword) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Service) QueryByPID(ctx context.Context, pid int64) ([]PortInfo, error) {
	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]PortInfo, 0)
	for _, item := range all {
		if item.PID != nil && *item.PID == pid {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Service) DetectConflicts(ctx context.Context) ([]PortConflict, error) {
	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	grouped := map[string][]PortInfo{}
	for _, item := range all {
		if strings.EqualFold(item.State, "LISTEN") && item.PID != nil {
			key := strings.ToUpper(item.Protocol) + ":" + strconv.Itoa(item.Port)
			grouped[key] = append(grouped[key], item)
		}
	}

	conflicts := make([]PortConflict, 0)
	for _, group := range grouped {
		pidSet := map[int64]struct{}{}
		nameSet := map[string]struct{}{}
		for _, item := range group {
			pidSet[*item.PID] = struct{}{}
			if item.ProcessName != "" {
				nameSet[item.ProcessName] = struct{}{}
			}
		}
		if len(pidSet) <= 1 {
			continue
		}
		first := group[0]
		pids := keysInt64(pidSet)
		names := keysString(nameSet)
		conflicts = append(conflicts, PortConflict{
			Port:         first.Port,
			Protocol:     first.Protocol,
			PIDs:         pids,
			ProcessNames: names,
			Message:      fmt.Sprintf("port %d (%s) is listened by %d processes", first.Port, first.Protocol, len(pids)),
		})
	}
	sort.SliceStable(conflicts, func(i, j int) bool {
		return conflicts[i].Port < conflicts[j].Port
	})
	return conflicts, nil
}

func (s *Service) GenerateFreePorts(ctx context.Context, startPort, count int) (FreePortResult, error) {
	result := FreePortResult{
		StartPort: startPort,
		Count:     count,
		FreePorts: []int{},
	}
	if startPort < 1 || startPort > 65535 {
		result.Message = "start port must be within 1-65535"
		return result, nil
	}
	if count < 1 || count > 100 {
		result.Message = "count must be within 1-100"
		return result, nil
	}

	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return result, err
	}
	occupied := map[int]struct{}{}
	for _, item := range all {
		occupied[item.Port] = struct{}{}
	}
	for port := startPort; port <= 65535 && len(result.FreePorts) < count; port++ {
		if _, exists := occupied[port]; !exists {
			result.FreePorts = append(result.FreePorts, port)
		}
	}
	if len(result.FreePorts) < count {
		result.Message = fmt.Sprintf("only found %d free ports before reaching 65535", len(result.FreePorts))
	} else {
		result.Message = fmt.Sprintf("found %d free ports", len(result.FreePorts))
	}
	return result, nil
}

func (s *Service) MonitorPorts(ctx context.Context, request PortMonitorRequest) ([]PortMonitorResult, error) {
	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return nil, err
	}
	portMap := map[string][]PortInfo{}
	for _, item := range all {
		protocol := strings.ToUpper(strings.TrimSpace(item.Protocol))
		if protocol == "" {
			protocol = "TCP"
		}
		key := protocol + ":" + fmt.Sprintf("%d", item.Port)
		portMap[key] = append(portMap[key], item)
	}

	results := make([]PortMonitorResult, 0, len(request.Ports))
	for _, item := range request.Ports {
		protocol := strings.ToUpper(strings.TrimSpace(item.Protocol))
		if protocol == "" {
			protocol = "TCP"
		}
		key := protocol + ":" + fmt.Sprintf("%d", item.Port)
		matched := portMap[key]
		result := PortMonitorResult{
			Port:     item.Port,
			Protocol: protocol,
			Occupied: len(matched) > 0,
			Remark:   item.Remark,
			State:    "FREE",
		}
		if len(matched) > 0 {
			first := matched[0]
			result.ProcessName = first.ProcessName
			result.PID = first.PID
			result.State = first.State
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) Summary(ctx context.Context) (PortSummary, error) {
	all, err := s.ScanAllPorts(ctx)
	if err != nil {
		return PortSummary{}, err
	}
	summary := PortSummary{Total: len(all)}
	uniquePorts := map[int]struct{}{}
	uniquePIDs := map[int64]struct{}{}

	for _, item := range all {
		uniquePorts[item.Port] = struct{}{}
		if item.PID != nil {
			uniquePIDs[*item.PID] = struct{}{}
		}
		if strings.EqualFold(item.Protocol, "TCP") {
			summary.TCPCount++
		}
		if strings.EqualFold(item.Protocol, "UDP") {
			summary.UDPCount++
		}
		if strings.EqualFold(item.State, "LISTEN") {
			summary.ListenCount++
		}
		if strings.EqualFold(item.State, "ESTABLISHED") {
			summary.EstablishedCount++
		}
		addr := strings.ToLower(item.LocalAddress)
		if strings.Contains(addr, "127.0.0.1") || strings.Contains(addr, "localhost") || strings.Contains(addr, "[::1]") {
			summary.LocalhostCount++
		}
		if strings.HasPrefix(addr, "*:") || strings.HasPrefix(addr, "0.0.0.0:") || strings.HasPrefix(addr, "[::]:") || strings.Contains(addr, "*.") {
			summary.AllInterfaceCount++
		}
	}
	summary.UniquePortCount = len(uniquePorts)
	summary.UniquePIDCount = len(uniquePIDs)
	return summary, nil
}

func keysInt64(values map[int64]struct{}) []int64 {
	result := make([]int64, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func keysString(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
