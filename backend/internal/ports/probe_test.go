package ports

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateProbePort(t *testing.T) {
	if err := ValidateProbePort(8080); err != nil {
		t.Fatalf("8080 should be valid: %v", err)
	}
	if err := ValidateProbePort(0); err == nil {
		t.Fatal("expected invalid port")
	}
	if err := ValidateBatchPorts(nil); err == nil {
		t.Fatal("expected empty batch error")
	}
	if err := ValidateBatchPorts(make([]int, 101)); err == nil {
		t.Fatal("expected batch limit error")
	}
}

func TestNormalizeProbeHostDNS(t *testing.T) {
	host, err := normalizeProbeHost("example.com")
	if err != nil || host != "example.com" {
		t.Fatalf("expected example.com, got %q err=%v", host, err)
	}
	if _, err = normalizeProbeHost("127.0.0.1:8080"); err == nil {
		t.Fatal("expected host:port rejection")
	}
	if _, err = normalizeProbeHost("bad host!!!"); err == nil {
		t.Fatal("expected invalid host error")
	}
}

func TestNormalizeProbeHostIPv6(t *testing.T) {
	host, err := normalizeProbeHost("::1")
	if err != nil || host != "[::1]" {
		t.Fatalf("expected bracketed ::1, got %q err=%v", host, err)
	}
	host, err = normalizeProbeHost("[::1]")
	if err != nil || host != "[::1]" {
		t.Fatalf("expected [::1], got %q err=%v", host, err)
	}
}

func TestValidateHTTPPathTotalLength(t *testing.T) {
	longQuery := strings.Repeat("a", MaxPathLength)
	_, _, err := ValidateHTTPPath("/p?" + longQuery)
	if err == nil {
		t.Fatal("expected path too long when query included")
	}
}

func TestValidateProbeTimeoutExplicit(t *testing.T) {
	if _, err := ValidateProbeTimeout(-1, true); err == nil {
		t.Fatal("expected error for negative timeout")
	}
	if _, err := ValidateProbeTimeout(40000, true); err == nil {
		t.Fatal("expected error for timeout above max")
	}
	d, err := ValidateProbeTimeout(0, false)
	if err != nil || d != time.Duration(DefaultProbeTimeoutMs)*time.Millisecond {
		t.Fatalf("expected default timeout, got %v err=%v", d, err)
	}
}

func TestValidateHTTPPathWithQuery(t *testing.T) {
	path, query, err := ValidateHTTPPath("/health?full=true")
	if err != nil || path != "/health" || query != "full=true" {
		t.Fatalf("unexpected path/query: %q ? %q err=%v", path, query, err)
	}
}

func TestProbeHTTPWithQuerySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" || r.URL.Query().Get("full") != "true" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host, portStr, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	port, _ := net.LookupPort("tcp", portStr)

	result := ProbeHTTP(context.Background(), host, port, "/health?full=true", DefaultProbeTimeoutMs, false)
	if !result.Reachable || result.HTTPStatus != http.StatusOK {
		t.Fatalf("expected successful HTTP probe, got %#v", result)
	}
}

func TestProbeTLSSuccess(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host, portStr, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "https://"))
	port, _ := net.LookupPort("tcp", portStr)

	result := ProbeTLS(context.Background(), host, port, DefaultProbeTimeoutMs, false)
	if !result.Reachable {
		t.Fatalf("expected successful TLS probe, got %#v", result)
	}
	_ = tls.VersionTLS12
}

func TestProbeTCPWithContextCancel(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := ProbeTCP(ctx, "127.0.0.1", port, 500, false)
	if result.Reachable {
		t.Fatalf("cancelled probe should not be reachable: %#v", result)
	}
}

func TestProbeTCPSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result := ProbeTCP(ctx, "127.0.0.1", addr.Port, 500, false)
	if !result.Reachable {
		t.Fatalf("expected reachable probe, got %#v", result)
	}
}

func TestProbeTCPIPv6Loopback(t *testing.T) {
	listener, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("ipv6 loopback not available: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result := ProbeTCP(ctx, "::1", addr.Port, 500, false)
	if !result.Reachable {
		t.Fatalf("expected reachable ipv6 probe, got %#v", result)
	}
}

func TestProbeTCPBatchOrderAndTimeout(t *testing.T) {
	var accepts int32
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	openPort := ln.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			atomic.AddInt32(&accepts, 1)
			_ = conn.Close()
		}
	}()

	ports := []int{openPort, 1, 2, 3}
	start := time.Now()
	results, err := ProbeTCPBatch(context.Background(), "127.0.0.1", ports, 1000, false)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("batch probe failed: %v", err)
	}
	if len(results) != len(ports) {
		t.Fatalf("expected %d results, got %d", len(ports), len(results))
	}
	if results[0].Port != openPort || !results[0].Reachable {
		t.Fatalf("expected first result reachable on open port, got %#v", results[0])
	}
	if elapsed > 3500*time.Millisecond {
		t.Fatalf("batch probe took too long: %v", elapsed)
	}
}

func TestProbeHTTPInvalidPath(t *testing.T) {
	result := ProbeHTTP(context.Background(), "127.0.0.1", 8080, "bad path", 500, false)
	if result.Reachable || !strings.Contains(result.Message, "invalid path") {
		t.Fatalf("expected invalid path, got %#v", result)
	}
}

func TestMonitorPortsProtocolIsolation(t *testing.T) {
	service := NewServiceWithScanner(mockScanner{data: []PortInfo{
		{Protocol: "TCP", Port: 53, ProcessName: "tcp-dns"},
		{Protocol: "UDP", Port: 53, ProcessName: "udp-dns"},
	}})
	results, err := service.MonitorPorts(context.Background(), PortMonitorRequest{Ports: []MonitorPortItem{
		{Port: 53, Protocol: "TCP"},
		{Port: 53, Protocol: "UDP"},
	}})
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}
	if !results[0].Occupied || results[0].ProcessName != "tcp-dns" {
		t.Fatalf("expected tcp 53 occupied, got %#v", results[0])
	}
	if !results[1].Occupied || results[1].ProcessName != "udp-dns" {
		t.Fatalf("expected udp 53 occupied, got %#v", results[1])
	}
}
