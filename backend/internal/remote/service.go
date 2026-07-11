package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"port-master/backend/internal/config"
	"port-master/backend/internal/ports"
)

var (
	ErrInvalidInput = errors.New("invalid remote host request")
	ErrAuthFailed   = errors.New("ssh authentication failed")
	ErrConnect      = errors.New("ssh connection failed")
)

const HostKeyPolicy = "accept-unknown"

type Service struct {
	cfg config.Config
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) TestConnection(ctx context.Context, req HostRequest) (bool, error) {
	client, err := s.dial(ctx, req)
	if err != nil {
		return false, err
	}
	defer client.Close()

	output, err := s.run(ctx, client, "echo ok")
	if err != nil {
		return false, err
	}
	return len(output) > 0 && strings.Contains(output[0], "ok"), nil
}

func (s *Service) SystemInfo(ctx context.Context, req HostRequest) (string, error) {
	client, err := s.dial(ctx, req)
	if err != nil {
		return "unknown", err
	}
	defer client.Close()

	osType, err := s.detectOS(ctx, client)
	if err != nil {
		return "unknown", nil
	}
	cmd := "uname -a"
	if osType == "windows" {
		cmd = "ver"
	}
	output, err := s.run(ctx, client, cmd)
	if err != nil || len(output) == 0 {
		return osType, nil
	}
	return strings.Join(output, " "), nil
}

func (s *Service) ScanPorts(ctx context.Context, req HostRequest) ([]ports.PortInfo, error) {
	client, err := s.dial(ctx, req)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	osType, err := s.detectOS(ctx, client)
	if err != nil {
		return nil, err
	}
	output, err := s.run(ctx, client, scanCommand(osType))
	if err != nil {
		return nil, err
	}
	result := ParseRemoteOutput(osType, output)
	host := strings.TrimSpace(req.Host)
	for i := range result {
		result[i].ProgramPath = "[remote:" + host + "]"
		if result[i].ProcessName == "" || result[i].ProcessName == "-" {
			result[i].ProcessName = "[remote]"
		}
	}
	return result, nil
}

func (s *Service) KillProcess(ctx context.Context, req KillRequest) (bool, error) {
	if req.PID <= 0 {
		return false, fmt.Errorf("%w: invalid pid", ErrInvalidInput)
	}
	client, err := s.dial(ctx, req.HostRequest)
	if err != nil {
		return false, err
	}
	defer client.Close()

	osType, err := s.detectOS(ctx, client)
	if err != nil {
		return false, err
	}
	_, err = s.run(ctx, client, killCommand(osType, req.PID, req.Force))
	return err == nil, err
}

func (s *Service) dial(ctx context.Context, req HostRequest) (*ssh.Client, error) {
	host, port, authType, err := validateRequest(req)
	if err != nil {
		return nil, err
	}

	authMethods, err := authMethods(req, authType)
	if err != nil {
		return nil, err
	}

	connectTimeout := s.cfg.SSHConnectTimeout()
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	sshConfig := &ssh.ClientConfig{
		User:            strings.TrimSpace(req.Username),
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         connectTimeout,
	}

	dialer := &net.Dialer{Timeout: connectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %s", ErrConnect, sanitizeError(err))
	}

	handshakeDeadline := time.Now().Add(connectTimeout)
	if err := conn.SetDeadline(handshakeDeadline); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: %s", ErrConnect, sanitizeError(err))
	}

	closed := int32(0)
	stopWatch := make(chan struct{})
	defer close(stopWatch)
	go func() {
		select {
		case <-ctx.Done():
			if atomic.CompareAndSwapInt32(&closed, 0, 1) {
				_ = conn.Close()
			}
		case <-stopWatch:
		}
	}()

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
	if err != nil {
		if atomic.CompareAndSwapInt32(&closed, 0, 1) {
			_ = conn.Close()
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if strings.Contains(strings.ToLower(err.Error()), "unable to authenticate") {
			return nil, ErrAuthFailed
		}
		return nil, fmt.Errorf("%w: %s", ErrConnect, sanitizeError(err))
	}

	if ctx.Err() != nil {
		_ = sshConn.Close()
		if atomic.CompareAndSwapInt32(&closed, 0, 1) {
			_ = conn.Close()
		}
		return nil, ctx.Err()
	}

	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(sshConn, chans, reqs), nil
}

func validateRequest(req HostRequest) (host string, port int, authType string, err error) {
	host = strings.TrimSpace(req.Host)
	if host == "" {
		return "", 0, "", fmt.Errorf("%w: host is required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Username) == "" {
		return "", 0, "", fmt.Errorf("%w: username is required", ErrInvalidInput)
	}
	port = req.Port
	if port < 0 {
		return "", 0, "", fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidInput)
	}
	if port == 0 {
		port = 22
	}
	if port < 1 || port > 65535 {
		return "", 0, "", fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidInput)
	}
	authType = strings.TrimSpace(strings.ToLower(req.AuthType))
	if authType == "" {
		authType = "password"
	}
	if authType != "password" && authType != "key" {
		return "", 0, "", fmt.Errorf("%w: authType must be password or key", ErrInvalidInput)
	}
	credential := strings.TrimSpace(req.Credential)
	if authType == "key" && credential == "" {
		return "", 0, "", fmt.Errorf("%w: private key is required", ErrInvalidInput)
	}
	if authType == "password" && credential == "" {
		return "", 0, "", fmt.Errorf("%w: password is required", ErrInvalidInput)
	}
	return host, port, authType, nil
}

func authMethods(req HostRequest, authType string) ([]ssh.AuthMethod, error) {
	if authType == "key" {
		signer, err := ssh.ParsePrivateKey([]byte(req.Credential))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid private key", ErrInvalidInput)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	return []ssh.AuthMethod{ssh.Password(req.Credential)}, nil
}

func (s *Service) detectOS(ctx context.Context, client *ssh.Client) (string, error) {
	output, err := s.run(ctx, client, "uname -s 2>/dev/null || echo windows")
	if err != nil {
		return "linux", nil
	}
	if len(output) == 0 {
		return "linux", nil
	}
	line := strings.ToLower(strings.TrimSpace(output[0]))
	switch {
	case strings.Contains(line, "windows"):
		return "windows", nil
	case strings.Contains(line, "darwin"):
		return "macos", nil
	default:
		return "linux", nil
	}
}

func (s *Service) run(ctx context.Context, client *ssh.Client, command string) ([]string, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &bytes.Buffer{}

	commandTimeout := s.cfg.SSHCommandTimeout()
	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-cmdCtx.Done():
		_ = session.Signal(ssh.SIGTERM)
		_ = session.Close()
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return nil, errors.New("ssh command timed out")
		}
		return nil, cmdCtx.Err()
	case err := <-done:
		if err != nil {
			return nil, errors.New("remote command failed")
		}
		lines := strings.Split(strings.ReplaceAll(stdout.String(), "\r\n", "\n"), "\n")
		result := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				result = append(result, line)
			}
		}
		return result, nil
	}
}

func scanCommand(osType string) string {
	switch osType {
	case "windows":
		return "netstat -ano"
	case "macos":
		return "lsof -i -P -n 2>/dev/null; netstat -an 2>/dev/null"
	default:
		return "ss -tulnp 2>/dev/null || ss -tunap 2>/dev/null || netstat -tulnp 2>/dev/null || lsof -i -P -n 2>/dev/null"
	}
}

func killCommand(osType string, pid int64, force bool) string {
	if osType == "windows" {
		if force {
			return fmt.Sprintf("taskkill /F /PID %d", pid)
		}
		return fmt.Sprintf("taskkill /PID %d", pid)
	}
	if force {
		return fmt.Sprintf("kill -9 %d", pid)
	}
	return fmt.Sprintf("kill -15 %d", pid)
}

func sanitizeError(err error) string {
	msg := err.Error()
	lower := strings.ToLower(msg)
	for _, secret := range []string{"password", "credential", "private key", "passphrase"} {
		if strings.Contains(lower, secret) {
			return "connection error"
		}
	}
	return msg
}
