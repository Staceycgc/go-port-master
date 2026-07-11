package config

import (
	"os"
	"testing"
)

func TestEnvIntInvalidFormat(t *testing.T) {
	t.Setenv("PORT_MASTER_TEST_INT", "not-a-number")
	if _, err := EnvInt("PORT_MASTER_TEST_INT", 8080); err == nil {
		t.Fatal("expected parse error for invalid integer env")
	}
}

func TestEnvInt64InvalidFormat(t *testing.T) {
	t.Setenv("PORT_MASTER_TEST_INT64", "bad")
	if _, err := EnvInt64("PORT_MASTER_TEST_INT64", 3000); err == nil {
		t.Fatal("expected parse error for invalid int64 env")
	}
}

func TestEnvIntFallback(t *testing.T) {
	os.Unsetenv("PORT_MASTER_TEST_MISSING")
	value, err := EnvInt("PORT_MASTER_TEST_MISSING", 42)
	if err != nil || value != 42 {
		t.Fatalf("expected fallback 42, got %d err=%v", value, err)
	}
}
