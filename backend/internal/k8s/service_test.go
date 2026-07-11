package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type mockRunner struct {
	available bool
	output    string
	err       error
}

func (m *mockRunner) Available(name string) bool {
	return m.available && name == "kubectl"
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []string{m.output}, nil
}

func TestListPodsUnavailable(t *testing.T) {
	client := NewClientWithRunner(&mockRunner{available: false})
	_, err := client.ListPods(context.Background(), "")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

func TestListPodsCommandFailure(t *testing.T) {
	client := NewClientWithRunner(&mockRunner{available: true, err: errors.New("context deadline")})
	_, err := client.ListPods(context.Background(), "")
	if !errors.Is(err, ErrCommandFailed) {
		t.Fatalf("expected command failed, got %v", err)
	}
}

func TestParseTargetPortNamed(t *testing.T) {
	tp := parseTargetPort(json.RawMessage(`"http"`))
	if tp.Name != "http" || tp.Number != nil {
		t.Fatalf("unexpected named target port: %#v", tp)
	}
	raw, err := json.Marshal(tp)
	if err != nil || string(raw) != `"http"` {
		t.Fatalf("marshal named target port: %s err=%v", raw, err)
	}
}

func TestParseTargetPortNumeric(t *testing.T) {
	tp := parseTargetPort(json.RawMessage(`8080`))
	if tp.Number == nil || *tp.Number != 8080 {
		t.Fatalf("unexpected numeric target port: %#v", tp)
	}
}

func TestParsePodsInvalidJSON(t *testing.T) {
	_, err := parsePods("{not-json")
	if err == nil || !errors.Is(err, ErrCommandFailed) {
		t.Fatalf("expected command failed for invalid pods json, got %v", err)
	}
}

func TestParseServicesInvalidJSON(t *testing.T) {
	_, err := parseServices("{not-json")
	if err == nil || !errors.Is(err, ErrCommandFailed) {
		t.Fatalf("expected command failed for invalid services json, got %v", err)
	}
}

func TestValidateNamespace(t *testing.T) {
	if err := validateNamespace(""); err != nil {
		t.Fatalf("empty namespace should be valid: %v", err)
	}
	if err := validateNamespace("bad namespace"); err == nil {
		t.Fatal("expected invalid namespace")
	}
}
