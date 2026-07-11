package config

import (
	"fmt"
	"time"
)

const Version = "2.1.0"

const (
	MaxScanCacheTTLMs = 300_000
	MinMonitorPollMs  = 1_000
	MaxMonitorPollMs  = 300_000
	MinSSHConnectMs   = 1_000
	MaxSSHConnectMs   = 120_000
	MinSSHCommandSec  = 1
	MaxSSHCommandSec  = 600
)

type Config struct {
	MonitorPollIntervalMs int64
	ScanCacheTTLMs        int64
	SSHConnectTimeoutMs   int
	SSHCommandTimeoutSec  int
}

func Default() Config {
	return Config{
		MonitorPollIntervalMs: 5000,
		ScanCacheTTLMs:        3000,
		SSHConnectTimeoutMs:   10000,
		SSHCommandTimeoutSec:  60,
	}
}

func (c Config) Validate() (Config, error) {
	out := c
	if out.MonitorPollIntervalMs == 0 {
		out.MonitorPollIntervalMs = Default().MonitorPollIntervalMs
	}
	if out.SSHConnectTimeoutMs == 0 {
		out.SSHConnectTimeoutMs = Default().SSHConnectTimeoutMs
	}
	if out.SSHCommandTimeoutSec == 0 {
		out.SSHCommandTimeoutSec = Default().SSHCommandTimeoutSec
	}
	if out.ScanCacheTTLMs < 0 || out.ScanCacheTTLMs > MaxScanCacheTTLMs {
		return Config{}, fmt.Errorf("scan cache TTL must be between 0 and %d ms", MaxScanCacheTTLMs)
	}
	if out.MonitorPollIntervalMs < MinMonitorPollMs || out.MonitorPollIntervalMs > MaxMonitorPollMs {
		return Config{}, fmt.Errorf("monitor poll interval must be between %d and %d ms", MinMonitorPollMs, MaxMonitorPollMs)
	}
	if out.SSHConnectTimeoutMs < MinSSHConnectMs || out.SSHConnectTimeoutMs > MaxSSHConnectMs {
		return Config{}, fmt.Errorf("ssh connect timeout must be between %d and %d ms", MinSSHConnectMs, MaxSSHConnectMs)
	}
	if out.SSHCommandTimeoutSec < MinSSHCommandSec || out.SSHCommandTimeoutSec > MaxSSHCommandSec {
		return Config{}, fmt.Errorf("ssh command timeout must be between %d and %d seconds", MinSSHCommandSec, MaxSSHCommandSec)
	}
	return out, nil
}

func (c Config) MonitorPollInterval() time.Duration {
	return time.Duration(c.MonitorPollIntervalMs) * time.Millisecond
}

func (c Config) ScanCacheTTL() time.Duration {
	if c.ScanCacheTTLMs <= 0 {
		return 0
	}
	return time.Duration(c.ScanCacheTTLMs) * time.Millisecond
}

func (c Config) SSHConnectTimeout() time.Duration {
	return time.Duration(c.SSHConnectTimeoutMs) * time.Millisecond
}

func (c Config) SSHCommandTimeout() time.Duration {
	return time.Duration(c.SSHCommandTimeoutSec) * time.Second
}

func (c Config) PublicMap() map[string]interface{} {
	return map[string]interface{}{
		"monitorPollIntervalMs": c.MonitorPollIntervalMs,
		"scanCacheTtlMs":        c.ScanCacheTTLMs,
		"sshConnectTimeoutMs":   c.SSHConnectTimeoutMs,
		"sshCommandTimeoutSec":  c.SSHCommandTimeoutSec,
		"version":               Version,
	}
}
