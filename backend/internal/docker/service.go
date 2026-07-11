package docker

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"port-master/backend/internal/executil"
)

var (
	ErrUnavailable   = errors.New("docker is not available")
	ErrCommandFailed = errors.New("docker command failed")
)

var containerIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{12,64}$`)
var portMappingPattern = regexp.MustCompile(
	`(?:(?P<hostIp>[\d.]+):)?(?P<hostPort>\d+)->(?P<containerPort>\d+)/(?P<protocol>tcp|udp)`)

type PortMapping struct {
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

type Container struct {
	ContainerID  string        `json:"containerId"`
	Name         string        `json:"name"`
	Image        string        `json:"image"`
	Status       string        `json:"status"`
	PortMappings []PortMapping `json:"portMappings"`
}

type Service struct {
	runner executil.Runner
}

func NewService() *Service {
	return NewServiceWithRunner(executil.OSRunner{})
}

func NewServiceWithRunner(runner executil.Runner) *Service {
	return &Service{runner: runner}
}

func (s *Service) Available() bool {
	return s.runner.Available("docker")
}

func (s *Service) ListContainers(ctx context.Context, all bool) ([]Container, error) {
	if !s.Available() {
		return nil, ErrUnavailable
	}
	args := []string{"ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}"}
	if all {
		args = []string{"ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}"}
	}
	lines, err := s.runner.Run(ctx, "docker", args...)
	if err != nil {
		return nil, wrapCommandError(err)
	}

	containers := make([]Container, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		portsStr := ""
		if len(parts) > 4 {
			portsStr = parts[4]
		}
		containers = append(containers, Container{
			ContainerID:  parts[0],
			Name:         parts[1],
			Image:        parts[2],
			Status:       parts[3],
			PortMappings: parsePortMappings(portsStr),
		})
	}
	return containers, nil
}

func (s *Service) StopContainer(ctx context.Context, containerID string) (string, error) {
	if err := validateContainerID(containerID); err != nil {
		return "", err
	}
	if !s.Available() {
		return "", ErrUnavailable
	}
	if _, err := s.runner.Run(ctx, "docker", "stop", containerID); err != nil {
		return "", wrapCommandError(err)
	}
	return fmt.Sprintf("container %s stopped", containerID), nil
}

func (s *Service) RestartContainer(ctx context.Context, containerID string) (string, error) {
	if err := validateContainerID(containerID); err != nil {
		return "", err
	}
	if !s.Available() {
		return "", ErrUnavailable
	}
	if _, err := s.runner.Run(ctx, "docker", "restart", containerID); err != nil {
		return "", wrapCommandError(err)
	}
	return fmt.Sprintf("container %s restarted", containerID), nil
}

func wrapCommandError(err error) error {
	if errors.Is(err, executil.ErrCommandNotFound) {
		return ErrUnavailable
	}
	var cmdErr *executil.CommandError
	if errors.As(err, &cmdErr) {
		return fmt.Errorf("%w: %s", ErrCommandFailed, cmdErr.Message)
	}
	return fmt.Errorf("%w: %s", ErrCommandFailed, err.Error())
}

func validateContainerID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("container id is required")
	}
	if !containerIDPattern.MatchString(id) {
		return errors.New("invalid container id")
	}
	return nil
}

func parsePortMappings(portsStr string) []PortMapping {
	mappings := make([]PortMapping, 0)
	if strings.TrimSpace(portsStr) == "" {
		return mappings
	}
	for _, segment := range strings.Split(portsStr, ",") {
		matches := portMappingPattern.FindStringSubmatch(strings.TrimSpace(segment))
		if matches == nil {
			continue
		}
		groups := map[string]string{}
		for i, name := range portMappingPattern.SubexpNames() {
			if i > 0 && name != "" {
				groups[name] = matches[i]
			}
		}
		mappings = append(mappings, PortMapping{
			HostPort:      groups["hostPort"],
			ContainerPort: groups["containerPort"],
			Protocol:      strings.ToUpper(groups["protocol"]),
		})
	}
	return mappings
}

func DefaultTimeout() time.Duration {
	return 30 * time.Second
}
