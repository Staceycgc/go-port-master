package config

import "testing"

func TestValidateDefaults(t *testing.T) {
	cfg, err := Default().Validate()
	if err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
	if cfg.MonitorPollIntervalMs != 5000 {
		t.Fatalf("unexpected monitor interval: %d", cfg.MonitorPollIntervalMs)
	}
}

func TestValidateScanCacheZeroAllowed(t *testing.T) {
	cfg := Default()
	cfg.ScanCacheTTLMs = 0
	if _, err := cfg.Validate(); err != nil {
		t.Fatalf("zero scan cache TTL should be allowed: %v", err)
	}
}

func TestValidateRejectsInvalidMonitorInterval(t *testing.T) {
	cfg := Default()
	cfg.MonitorPollIntervalMs = 500
	if _, err := cfg.Validate(); err == nil {
		t.Fatal("expected error for monitor interval below minimum")
	}
}

func TestValidateRejectsNegativeScanCache(t *testing.T) {
	cfg := Default()
	cfg.ScanCacheTTLMs = -1
	if _, err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative scan cache TTL")
	}
}
