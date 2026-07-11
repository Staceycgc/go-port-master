package monitor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"port-master/backend/internal/ports"
)

const (
	MaxMonitorPorts    = 100
	DefaultPollTimeout = 30 * time.Second
)

var (
	ErrInvalidMonitorConfig = errors.New("invalid monitor config")
	validExpectedStates     = map[string]struct{}{
		"any":      {},
		"occupied": {},
		"free":     {},
	}
)

type AlertEvent struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Occupied    bool   `json:"occupied"`
	ProcessName string `json:"processName,omitempty"`
	PID         *int64 `json:"pid,omitempty"`
	Remark      string `json:"remark,omitempty"`
	Reason      string `json:"reason"`
	Timestamp   string `json:"timestamp"`
}

func monitorKey(protocol string, port int) string {
	return strings.ToUpper(strings.TrimSpace(protocol)) + ":" + strconvItoa(port)
}

func strconvItoa(v int) string {
	return fmt.Sprintf("%d", v)
}

type Registry struct {
	mu              sync.RWMutex
	enabled         bool
	ports           []ports.MonitorPortItem
	lastOccupied    map[string]bool
	lastAlertReason map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		lastOccupied:    map[string]bool{},
		lastAlertReason: map[string]string{},
	}
}

func (r *Registry) Update(enabled bool, items []ports.MonitorPortItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = enabled
	r.ports = append([]ports.MonitorPortItem(nil), items...)
	active := map[string]struct{}{}
	for _, item := range r.ports {
		active[monitorKey(item.Protocol, item.Port)] = struct{}{}
	}
	for key := range r.lastOccupied {
		if _, ok := active[key]; !ok {
			delete(r.lastOccupied, key)
			delete(r.lastAlertReason, key)
		}
	}
	if !enabled {
		r.lastOccupied = map[string]bool{}
		r.lastAlertReason = map[string]string{}
	}
}

func (r *Registry) Enabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled
}

func (r *Registry) Ports() []ports.MonitorPortItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]ports.MonitorPortItem(nil), r.ports...)
}

func (r *Registry) PortCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.ports)
}

func (r *Registry) InitSnapshot(key string, occupied bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastOccupied[key] = occupied
}

func (r *Registry) LastOccupied(key string) (bool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.lastOccupied[key]
	return value, ok
}

func (r *Registry) SetLastOccupied(key string, occupied bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastOccupied[key] = occupied
}

func (r *Registry) ShouldAlert(key, reason string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lastAlertReason[key] == reason {
		return false
	}
	r.lastAlertReason[key] = reason
	return true
}

func (r *Registry) ClearAlertReason(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lastAlertReason, key)
}

func (r *Registry) ClearSnapshots() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastOccupied = map[string]bool{}
	r.lastAlertReason = map[string]string{}
}

func alertReason(prev bool, prevOK bool, occupied bool, expected string) string {
	if expected == "occupied" && !occupied {
		return "expected_occupied"
	}
	if expected == "free" && occupied {
		return "expected_free"
	}
	if expected != "any" {
		return ""
	}
	if prevOK && prev != occupied {
		if occupied {
			return "occupied"
		}
		return "released"
	}
	return ""
}

func complianceMet(expected string, occupied bool) bool {
	switch expected {
	case "occupied":
		return occupied
	case "free":
		return !occupied
	default:
		return false
	}
}

func ParseMonitorItems(raw []ports.MonitorPortItem) ([]ports.MonitorPortItem, error) {
	if len(raw) > MaxMonitorPorts {
		return nil, fmt.Errorf("%w: at most %d ports", ErrInvalidMonitorConfig, MaxMonitorPorts)
	}
	items := make([]ports.MonitorPortItem, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		if err := ports.ValidateProbePort(item.Port); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidMonitorConfig, err)
		}
		protocol := strings.ToUpper(strings.TrimSpace(item.Protocol))
		if protocol == "" {
			protocol = "TCP"
		}
		if protocol != "TCP" && protocol != "UDP" {
			return nil, fmt.Errorf("%w: unsupported protocol %q", ErrInvalidMonitorConfig, item.Protocol)
		}
		expected := strings.TrimSpace(item.ExpectedState)
		if expected == "" {
			expected = "any"
		}
		if _, ok := validExpectedStates[expected]; !ok {
			return nil, fmt.Errorf("%w: invalid expectedState %q", ErrInvalidMonitorConfig, item.ExpectedState)
		}
		key := monitorKey(protocol, item.Port)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, ports.MonitorPortItem{
			Port:          item.Port,
			Protocol:      protocol,
			Remark:        item.Remark,
			ExpectedState: expected,
		})
	}
	return items, nil
}

type Hub struct {
	mu       sync.RWMutex
	sessions map[*wsSession]struct{}
}

func NewHub() *Hub {
	return &Hub{sessions: map[*wsSession]struct{}{}}
}

func (h *Hub) Add(session *wsSession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[session] = struct{}{}
}

func (h *Hub) Remove(session *wsSession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, session)
}

func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

func (h *Hub) Broadcast(payload interface{}) {
	h.mu.RLock()
	sessions := make([]*wsSession, 0, len(h.sessions))
	for session := range h.sessions {
		sessions = append(sessions, session)
	}
	h.mu.RUnlock()
	for _, session := range sessions {
		if !session.sendJSON(payload) {
			h.Remove(session)
			session.close()
		}
	}
}

func (h *Hub) Shutdown() {
	h.mu.Lock()
	sessions := make([]*wsSession, 0, len(h.sessions))
	for session := range h.sessions {
		sessions = append(sessions, session)
	}
	h.sessions = map[*wsSession]struct{}{}
	h.mu.Unlock()
	for _, session := range sessions {
		session.close()
	}
}

type Scheduler struct {
	registry *Registry
	hub      *Hub
	ports    *ports.Service
	interval time.Duration
	pollTO   time.Duration

	mu         sync.Mutex
	generation uint64
	cancel     context.CancelFunc
	done       chan struct{}
}

func NewScheduler(registry *Registry, hub *Hub, portService *ports.Service, interval, pollTimeout time.Duration) *Scheduler {
	if interval <= 0 {
		interval = time.Second
	}
	if pollTimeout <= 0 {
		pollTimeout = DefaultPollTimeout
	}
	return &Scheduler{
		registry: registry,
		hub:      hub,
		ports:    portService,
		interval: interval,
		pollTO:   pollTimeout,
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	gen := s.generation + 1
	s.generation = gen
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.mu.Unlock()

	go func() {
		defer close(done)
		s.loop(ctx)
		s.mu.Lock()
		if s.generation == gen {
			s.cancel = nil
			s.done = nil
		}
		s.mu.Unlock()
	}()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollOnce(ctx)
		}
	}
}

func (s *Scheduler) pollOnce(parent context.Context) {
	if !s.registry.Enabled() || s.registry.PortCount() == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(parent, s.pollTO)
	defer cancel()

	items := s.registry.Ports()
	results, err := s.ports.MonitorPorts(ctx, ports.PortMonitorRequest{Ports: items})
	if err != nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	alerts := evaluatePollAlerts(s.registry, items, results, now)
	if len(alerts) > 0 {
		s.hub.Broadcast(map[string]interface{}{
			"type":   "alert",
			"alerts": alerts,
		})
	}
}

func evaluatePollAlerts(reg *Registry, items []ports.MonitorPortItem, results []ports.PortMonitorResult, now string) []AlertEvent {
	alerts := make([]AlertEvent, 0)
	for _, result := range results {
		key := monitorKey(result.Protocol, result.Port)
		prev, ok := reg.LastOccupied(key)
		if !ok {
			reg.InitSnapshot(key, result.Occupied)
			continue
		}
		item := findItem(items, result.Protocol, result.Port)
		expected := "any"
		if item != nil {
			expected = item.ExpectedState
		}
		reason := alertReason(prev, ok, result.Occupied, expected)
		reg.SetLastOccupied(key, result.Occupied)
		if complianceMet(expected, result.Occupied) {
			reg.ClearAlertReason(key)
		}
		if reason == "" {
			continue
		}
		if !reg.ShouldAlert(key, reason) {
			continue
		}
		alerts = append(alerts, buildAlert(result, item, reason, now))
	}
	return alerts
}

func findItem(items []ports.MonitorPortItem, protocol string, port int) *ports.MonitorPortItem {
	for i := range items {
		if items[i].Port == port && strings.EqualFold(items[i].Protocol, protocol) {
			return &items[i]
		}
	}
	return nil
}

func buildAlert(result ports.PortMonitorResult, item *ports.MonitorPortItem, reason, timestamp string) AlertEvent {
	remark := ""
	if item != nil {
		remark = item.Remark
	}
	return AlertEvent{
		Port:        result.Port,
		Protocol:    result.Protocol,
		Occupied:    result.Occupied,
		ProcessName: result.ProcessName,
		PID:         result.PID,
		Remark:      remark,
		Reason:      reason,
		Timestamp:   timestamp,
	}
}
