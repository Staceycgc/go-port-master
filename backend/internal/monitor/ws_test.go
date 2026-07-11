package monitor

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckSameOriginSamePort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/ws/monitor", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://localhost:8080")
	if !checkSameOrigin(req) {
		t.Fatal("expected same origin for matching host and port")
	}
}

func TestCheckSameOriginDifferentPortRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/ws/monitor", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://localhost:5173")
	if checkSameOrigin(req) {
		t.Fatal("expected different ports to be rejected")
	}
}

func TestCheckSameOriginMalformedRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/ws/monitor", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "not-a-valid-origin")
	if checkSameOrigin(req) {
		t.Fatal("expected malformed origin to be rejected")
	}
}

func TestCheckSameOriginMissingOriginAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/ws/monitor", nil)
	req.Host = "localhost:8080"
	if !checkSameOrigin(req) {
		t.Fatal("expected missing origin to be allowed")
	}
}
