package docker

import (
	"context"
	"errors"
	"testing"
)

type mockRunner struct {
	available bool
	lines     []string
	err       error
	calls     [][]string
}

func (m *mockRunner) Available(name string) bool {
	return m.available && name == "docker"
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) ([]string, error) {
	m.calls = append(m.calls, append([]string{name}, args...))
	if m.err != nil {
		return nil, m.err
	}
	return m.lines, nil
}

func TestListContainersUnavailable(t *testing.T) {
	svc := NewServiceWithRunner(&mockRunner{available: false})
	_, err := svc.ListContainers(context.Background(), false)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

func TestListContainersCommandFailure(t *testing.T) {
	svc := NewServiceWithRunner(&mockRunner{
		available: true,
		err:       errors.New("daemon not running"),
	})
	_, err := svc.ListContainers(context.Background(), false)
	if !errors.Is(err, ErrCommandFailed) {
		t.Fatalf("expected command failed, got %v", err)
	}
}

func TestListContainersSuccess(t *testing.T) {
	svc := NewServiceWithRunner(&mockRunner{
		available: true,
		lines:     []string{"abc123\tweb\tnginx:latest\tUp 1 hour\t0.0.0.0:8080->80/tcp"},
	})
	items, err := svc.ListContainers(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].ContainerID != "abc123" {
		t.Fatalf("unexpected items: %#v", items)
	}
}
