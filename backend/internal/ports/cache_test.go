package ports

import (
	"context"
	"testing"
	"time"
)

func TestScanCacheRefresh(t *testing.T) {
	scans := 0
	scanner := mockScanner{
		data: []PortInfo{{Port: 8080, Protocol: "TCP", State: "LISTEN"}},
	}
	service := NewServiceWithOptions(mockScannerFunc(func(ctx context.Context) ([]PortInfo, error) {
		scans++
		return scanner.data, scanner.err
	}), 2*time.Second)

	first, err := service.ScanAllPorts(context.Background())
	if err != nil || len(first) != 1 {
		t.Fatalf("first scan failed: %#v err=%v", first, err)
	}
	second, err := service.ScanAllPorts(context.Background())
	if err != nil || len(second) != 1 {
		t.Fatalf("cached scan failed: %#v err=%v", second, err)
	}
	if scans != 1 {
		t.Fatalf("expected one scanner call, got %d", scans)
	}

	refreshed, err := service.ScanAllPortsRefresh(context.Background(), true)
	if err != nil || len(refreshed) != 1 {
		t.Fatalf("refresh scan failed: %#v err=%v", refreshed, err)
	}
	if scans != 2 {
		t.Fatalf("expected refresh to bypass cache, got %d scans", scans)
	}
}

type mockScannerFunc func(ctx context.Context) ([]PortInfo, error)

func (f mockScannerFunc) ScanAllPorts(ctx context.Context) ([]PortInfo, error) {
	return f(ctx)
}
