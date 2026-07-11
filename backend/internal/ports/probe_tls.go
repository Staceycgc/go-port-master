package ports

import (
	"context"
	"crypto/tls"
	"io"
	"net"
)

func tlsDialContext(ctx context.Context, dialer *net.Dialer, addr, serverName string) (*tls.Conn, error) {
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Client(rawConn, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = rawConn.Close()
		return nil, err
	}
	return tlsConn, nil
}

func ioCopyDiscard(ctx context.Context, reader io.Reader, limit int64) (int64, error) {
	if ctx == nil {
		return io.Copy(io.Discard, io.LimitReader(reader, limit))
	}
	type copyResult struct {
		n   int64
		err error
	}
	done := make(chan copyResult, 1)
	go func() {
		n, err := io.Copy(io.Discard, io.LimitReader(reader, limit))
		done <- copyResult{n, err}
	}()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-done:
		return res.n, res.err
	}
}
