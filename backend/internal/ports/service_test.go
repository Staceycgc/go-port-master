package ports

import (
	"context"
	"net"
	"strconv"
	"testing"
)

type mockScanner struct {
	data []PortInfo
	err  error
}

func (m mockScanner) ScanAllPorts(ctx context.Context) ([]PortInfo, error) {
	return m.data, m.err
}

func TestParsePortSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    []int
		wantErr bool
	}{
		{name: "single", spec: "8080", want: []int{8080}},
		{name: "range", spec: "8000-8002", want: []int{8000, 8001, 8002}},
		{name: "mixed", spec: "8080, 9000-9001", want: []int{8080, 9000, 9001}},
		{name: "reverse range", spec: "8002-8000", want: []int{8000, 8001, 8002}},
		{name: "invalid", spec: "abc", wantErr: true},
		{name: "out of range", spec: "70000", wantErr: true},
		{name: "range too large", spec: "1-10002", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePortSpec(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, port := range tt.want {
				if _, ok := got[port]; !ok {
					t.Fatalf("expected port %d in parsed spec", port)
				}
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d ports, got %d", len(tt.want), len(got))
			}
		})
	}
}

func TestProbeTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	defer listener.Close()

	reachable := ProbeTCP(context.Background(), "127.0.0.1", addr.Port, 500, false)
	if !reachable.Reachable {
		t.Fatalf("expected reachable probe, got %#v", reachable)
	}

	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	closed := ProbeTCP(context.Background(), "127.0.0.1", addr.Port, 200, false)
	if closed.Reachable {
		t.Fatalf("expected closed probe to be unreachable, got %#v", closed)
	}
}

func TestSummaryConflictAndMonitor(t *testing.T) {
	pid1 := int64(1001)
	pid2 := int64(1002)
	service := NewServiceWithScanner(mockScanner{data: []PortInfo{
		{Protocol: "TCP", Port: 8080, LocalAddress: "0.0.0.0:8080", ForeignAddress: "*:*", PID: &pid1, ProcessName: "api-a", State: "LISTEN"},
		{Protocol: "TCP", Port: 8080, LocalAddress: "127.0.0.1:8080", ForeignAddress: "*:*", PID: &pid2, ProcessName: "api-b", State: "LISTEN"},
		{Protocol: "TCP", Port: 9000, LocalAddress: "127.0.0.1:9000", ForeignAddress: "127.0.0.1:50000", PID: &pid1, ProcessName: "api-a", State: "ESTABLISHED"},
		{Protocol: "UDP", Port: 5353, LocalAddress: "*:5353", ForeignAddress: "*:*", State: "LISTEN"},
	}})

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Total != 4 || summary.TCPCount != 3 || summary.UDPCount != 1 || summary.ListenCount != 3 || summary.EstablishedCount != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.UniquePortCount != 3 || summary.UniquePIDCount != 2 || summary.LocalhostCount != 2 || summary.AllInterfaceCount != 2 {
		t.Fatalf("unexpected unique/interface summary: %#v", summary)
	}

	conflicts, err := service.DetectConflicts(context.Background())
	if err != nil {
		t.Fatalf("conflicts: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected one conflict, got %d", len(conflicts))
	}
	if conflicts[0].Port != 8080 || len(conflicts[0].PIDs) != 2 {
		t.Fatalf("unexpected conflict: %#v", conflicts[0])
	}

	monitor, err := service.MonitorPorts(context.Background(), PortMonitorRequest{Ports: []MonitorPortItem{
		{Port: 8080, Protocol: "TCP", Remark: "api"},
		{Port: 7000, Protocol: "TCP", Remark: "free"},
	}})
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}
	if len(monitor) != 2 {
		t.Fatalf("expected two monitor results, got %d", len(monitor))
	}
	if !monitor[0].Occupied || monitor[0].PID == nil || *monitor[0].PID != pid1 {
		t.Fatalf("unexpected occupied monitor result: %#v", monitor[0])
	}
	if monitor[1].Occupied || monitor[1].State != "FREE" {
		t.Fatalf("unexpected free monitor result: %#v", monitor[1])
	}
}

func TestGenerateFreePorts(t *testing.T) {
	service := NewServiceWithScanner(mockScanner{data: []PortInfo{
		{Port: 8080},
		{Port: 8082},
	}})
	result, err := service.GenerateFreePorts(context.Background(), 8080, 3)
	if err != nil {
		t.Fatalf("GenerateFreePorts: %v", err)
	}
	got := map[string]struct{}{}
	for _, port := range result.FreePorts {
		got[strconv.Itoa(port)] = struct{}{}
	}
	for _, port := range []int{8081, 8083, 8084} {
		if _, ok := got[strconv.Itoa(port)]; !ok {
			t.Fatalf("expected free port %d in %#v", port, result.FreePorts)
		}
	}
}
