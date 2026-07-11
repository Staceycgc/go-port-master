package remote

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"port-master/backend/internal/config"
)

func TestValidateRequestUnknownAuthType(t *testing.T) {
	_, _, _, err := validateRequest(HostRequest{
		Host: "127.0.0.1", Username: "root", AuthType: "token", Credential: "x",
	})
	if err == nil {
		t.Fatal("expected unknown authType error")
	}
}

func TestValidateRequestNegativePort(t *testing.T) {
	_, _, _, err := validateRequest(HostRequest{
		Host: "127.0.0.1", Username: "root", Port: -1, AuthType: "password", Credential: "x",
	})
	if err == nil {
		t.Fatal("expected negative port error")
	}
}

func TestValidateRequestZeroPortDefaultsTo22(t *testing.T) {
	_, port, authType, err := validateRequest(HostRequest{
		Host: "127.0.0.1", Username: "root", Port: 0, AuthType: "password", Credential: "x",
	})
	if err != nil || port != 22 || authType != "password" {
		t.Fatalf("expected default port 22, got port=%d auth=%s err=%v", port, authType, err)
	}
}

func TestDialCancelClosesTCPConnection(t *testing.T) {
	for i := 0; i < 20; i++ {
		runDialCancelClosesTCPConnection(t)
	}
}

func runDialCancelClosesTCPConnection(t *testing.T) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			accepted <- conn
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	svc := NewService(config.Default())
	errCh := make(chan error, 1)
	go func() {
		_, err := svc.dial(ctx, HostRequest{
			Host:       "127.0.0.1",
			Port:       addr.Port,
			Username:   "root",
			AuthType:   "password",
			Credential: "secret",
		})
		errCh <- err
	}()

	var serverConn net.Conn
	select {
	case serverConn = <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for tcp accept")
	}
	defer serverConn.Close()

	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected dial error after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dial did not return after cancel")
	}

	if err := readUntilConnectionClosed(serverConn, 2*time.Second); err != nil {
		t.Fatalf("expected server connection to close: %v", err)
	}
}

func readUntilConnectionClosed(conn net.Conn, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 4096)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return errors.New("timeout waiting for connection close")
		}
		_ = conn.SetReadDeadline(time.Now().Add(remaining))
		_, err := conn.Read(buf)
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) || isConnClosedErr(err) {
			return nil
		}
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return errors.New("timeout waiting for connection close")
		}
		return err
	}
}

func isConnClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "closed") ||
		strings.Contains(msg, "reset") ||
		strings.Contains(msg, "forcibly closed") ||
		strings.Contains(msg, "broken pipe")
}
