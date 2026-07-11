package executil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var ErrCommandNotFound = errors.New("command not available")

type CommandError struct {
	Command string
	Message string
}

func (e *CommandError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("command failed: %s", e.Command)
	}
	return e.Message
}

type Runner interface {
	Available(name string) bool
	Run(ctx context.Context, name string, args ...string) ([]string, error)
}

type OSRunner struct{}

func (OSRunner) Available(name string) bool {
	return IsCommandAvailable(name)
}

func (OSRunner) Run(ctx context.Context, name string, args ...string) ([]string, error) {
	return Run(ctx, name, args...)
}

func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func Run(ctx context.Context, name string, args ...string) ([]string, error) {
	if !IsCommandAvailable(name) {
		return nil, ErrCommandNotFound
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, &CommandError{Command: name, Message: msg}
	}
	return splitLines(stdout.String()), nil
}

func RunWithTimeout(name string, timeout time.Duration, args ...string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return Run(ctx, name, args...)
}

func splitLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}
