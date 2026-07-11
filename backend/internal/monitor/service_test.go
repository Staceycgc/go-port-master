package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"port-master/backend/internal/ports"
)

type mockPortService struct {
	results []ports.PortMonitorResult
}

func (m *mockPortService) MonitorPorts(ctx context.Context, request ports.PortMonitorRequest) ([]ports.PortMonitorResult, error) {
	return m.results, nil
}

func TestRegistryUpdateClearsRemovedSnapshots(t *testing.T) {
	reg := NewRegistry()
	reg.Update(true, []ports.MonitorPortItem{{Port: 8080, Protocol: "TCP", ExpectedState: "any"}})
	reg.InitSnapshot(monitorKey("TCP", 8080), true)
	reg.Update(true, []ports.MonitorPortItem{{Port: 9090, Protocol: "TCP", ExpectedState: "any"}})
	if _, ok := reg.LastOccupied(monitorKey("TCP", 8080)); ok {
		t.Fatal("removed port snapshot should be cleared")
	}
}

func TestShouldAlertDedup(t *testing.T) {
	reg := NewRegistry()
	key := monitorKey("TCP", 8080)
	if !reg.ShouldAlert(key, "expected_occupied") {
		t.Fatal("first alert should pass")
	}
	if reg.ShouldAlert(key, "expected_occupied") {
		t.Fatal("duplicate alert should be suppressed")
	}
	if !reg.ShouldAlert(key, "occupied") {
		t.Fatal("different reason should pass")
	}
}

func TestAlertReasonExpectedOccupiedPriority(t *testing.T) {
	reason := alertReason(true, true, false, "occupied")
	if reason != "expected_occupied" {
		t.Fatalf("expected expected_occupied, got %q", reason)
	}
	reason = alertReason(true, true, false, "any")
	if reason != "released" {
		t.Fatalf("expected released for any expectation, got %q", reason)
	}
}

func TestPollOnceExpectedOccupiedSequence(t *testing.T) {
	reg := NewRegistry()
	items := []ports.MonitorPortItem{{Port: 8080, Protocol: "TCP", ExpectedState: "occupied", Remark: "api"}}
	reg.Update(true, items)
	reg.InitSnapshot(monitorKey("TCP", 8080), true)
	now := time.Now().UTC().Format(time.RFC3339)

	alerts := evaluatePollAlerts(reg, items, []ports.PortMonitorResult{{Port: 8080, Protocol: "TCP", Occupied: true}}, now)
	if len(alerts) != 0 {
		t.Fatalf("expected no alert while occupied, got %#v", alerts)
	}

	alerts = evaluatePollAlerts(reg, items, []ports.PortMonitorResult{{Port: 8080, Protocol: "TCP", Occupied: false}}, now)
	if len(alerts) != 1 || alerts[0].Reason != "expected_occupied" {
		t.Fatalf("expected single expected_occupied alert, got %#v", alerts)
	}

	alerts = evaluatePollAlerts(reg, items, []ports.PortMonitorResult{{Port: 8080, Protocol: "TCP", Occupied: false}}, now)
	if len(alerts) != 0 {
		t.Fatalf("expected deduped stable violation, got %#v", alerts)
	}

	alerts = evaluatePollAlerts(reg, items, []ports.PortMonitorResult{{Port: 8080, Protocol: "TCP", Occupied: true}}, now)
	if len(alerts) != 0 {
		t.Fatalf("expected no alert on recovery, got %#v", alerts)
	}

	alerts = evaluatePollAlerts(reg, items, []ports.PortMonitorResult{{Port: 8080, Protocol: "TCP", Occupied: false}}, now)
	if len(alerts) != 1 || alerts[0].Reason != "expected_occupied" {
		t.Fatalf("expected alert after recovery and re-violation, got %#v", alerts)
	}
}

func TestSchedulerConcurrentStartStop(t *testing.T) {
	reg := NewRegistry()
	hub := NewHub()
	svc := ports.NewServiceWithScanner(mockScanner{})
	scheduler := NewScheduler(reg, hub, svc, 20*time.Millisecond, time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scheduler.Start()
			time.Sleep(5 * time.Millisecond)
			scheduler.Stop()
		}()
	}
	wg.Wait()
	scheduler.Stop()
}

type mockScanner struct{}

func (mockScanner) ScanAllPorts(ctx context.Context) ([]ports.PortInfo, error) {
	return nil, nil
}

func TestParseMonitorItemsValidation(t *testing.T) {
	_, err := ParseMonitorItems([]ports.MonitorPortItem{{Port: 0, Protocol: "TCP"}})
	if err == nil {
		t.Fatal("expected invalid port error")
	}
	_, err = ParseMonitorItems([]ports.MonitorPortItem{{Port: 8080, Protocol: "TCP", ExpectedState: "invalid"}})
	if err == nil {
		t.Fatal("expected invalid expected state")
	}
}

func TestSchedulerStartStop(t *testing.T) {
	reg := NewRegistry()
	hub := NewHub()
	svc := ports.NewServiceWithScanner(mockScanner{})
	scheduler := NewScheduler(reg, hub, svc, 50*time.Millisecond, time.Second)
	scheduler.Start()
	scheduler.Start()
	time.Sleep(20 * time.Millisecond)
	scheduler.Stop()
	scheduler.Stop()
}
