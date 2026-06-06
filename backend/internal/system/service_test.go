package system

import (
	"testing"

	"github.com/shirou/gopsutil/v4/cpu"
)

func TestCalculateCPUUsage(t *testing.T) {
	previous := cpu.TimesStat{User: 10, System: 5, Idle: 85}
	current := cpu.TimesStat{User: 30, System: 15, Idle: 155}

	got := calculateCPUUsage(previous, current)
	want := 30.0
	if got != want {
		t.Fatalf("expected %.1f, got %.1f", want, got)
	}
}

func TestCalculateCPUUsageWithNoDelta(t *testing.T) {
	previous := cpu.TimesStat{User: 10, System: 5, Idle: 85}
	current := previous

	got := calculateCPUUsage(previous, current)
	if got != 0 {
		t.Fatalf("expected zero for no delta, got %.1f", got)
	}
}
