package ports

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	DefaultProbeTimeoutMs = 3000
	MaxProbeTimeoutMs     = 30000
	MaxBatchProbePorts    = 100
	MaxHTTPResponseSize   = 64 * 1024
	MaxHostLength         = 253
	MaxPathLength         = 512
	MaxBatchConcurrency   = 10
)

var (
	ErrInvalidProbeInput = errors.New("invalid probe input")
	dnsLabelPattern      = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
)

func ResolveProbeHost(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "127.0.0.1", nil
	}
	return normalizeProbeHost(raw)
}

func normalizeProbeHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" || len(host) > MaxHostLength {
		return "", fmt.Errorf("%w: invalid host", ErrInvalidProbeInput)
	}
	if strings.Contains(host, "://") || strings.Contains(host, "/") || strings.ContainsAny(host, " \t\r\n") {
		return "", fmt.Errorf("%w: invalid host", ErrInvalidProbeInput)
	}

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		ip := net.ParseIP(host[1 : len(host)-1])
		if ip == nil || ip.To4() != nil {
			return "", fmt.Errorf("%w: invalid host", ErrInvalidProbeInput)
		}
		return host, nil
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			return host, nil
		}
		return "[" + host + "]", nil
	}

	if strings.Count(host, ":") == 1 {
		parts := strings.SplitN(host, ":", 2)
		if net.ParseIP(parts[0]) != nil {
			return "", fmt.Errorf("%w: host must not include port", ErrInvalidProbeInput)
		}
	}
	if strings.Contains(host, ":") {
		return "", fmt.Errorf("%w: host must not include port", ErrInvalidProbeInput)
	}

	if strings.EqualFold(host, "localhost") {
		return "127.0.0.1", nil
	}
	if !isValidDNSHostname(host) {
		return "", fmt.Errorf("%w: invalid host", ErrInvalidProbeInput)
	}
	return strings.ToLower(host), nil
}

func isValidDNSHostname(host string) bool {
	if len(host) == 0 || len(host) > MaxHostLength {
		return false
	}
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || len(label) > 63 || !dnsLabelPattern.MatchString(label) {
			return false
		}
	}
	return true
}

func ValidateProbePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidProbeInput)
	}
	return nil
}

func ValidatePortString(raw string) (int, error) {
	port, err := net.LookupPort("tcp", raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%w: invalid port", ErrInvalidProbeInput)
	}
	return port, nil
}

func ValidateProbeTimeout(timeoutMs int, explicit bool) (time.Duration, error) {
	if !explicit || timeoutMs == 0 {
		return time.Duration(DefaultProbeTimeoutMs) * time.Millisecond, nil
	}
	if timeoutMs < 1 || timeoutMs > MaxProbeTimeoutMs {
		return 0, fmt.Errorf("%w: timeout must be between 1 and %d ms", ErrInvalidProbeInput, MaxProbeTimeoutMs)
	}
	return time.Duration(timeoutMs) * time.Millisecond, nil
}

func ValidateHTTPPath(raw string) (path string, rawQuery string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/", "", nil
	}
	if strings.Contains(raw, "://") {
		return "", "", fmt.Errorf("%w: invalid path", ErrInvalidProbeInput)
	}
	pathPart := raw
	queryPart := ""
	if idx := strings.Index(raw, "?"); idx >= 0 {
		pathPart = raw[:idx]
		queryPart = raw[idx+1:]
	}
	if !strings.HasPrefix(pathPart, "/") {
		pathPart = "/" + pathPart
	}
	totalLen := len(pathPart)
	if queryPart != "" {
		totalLen += 1 + len(queryPart)
	}
	if totalLen > MaxPathLength {
		return "", "", fmt.Errorf("%w: path too long", ErrInvalidProbeInput)
	}
	if strings.Contains(pathPart, " ") {
		return "", "", fmt.Errorf("%w: invalid path", ErrInvalidProbeInput)
	}
	if strings.Contains(queryPart, " ") {
		return "", "", fmt.Errorf("%w: invalid path", ErrInvalidProbeInput)
	}
	return pathPart, queryPart, nil
}

func ValidateBatchPorts(ports []int) error {
	if len(ports) == 0 {
		return fmt.Errorf("%w: at least one port is required", ErrInvalidProbeInput)
	}
	if len(ports) > MaxBatchProbePorts {
		return fmt.Errorf("%w: at most %d ports per batch", ErrInvalidProbeInput, MaxBatchProbePorts)
	}
	for _, port := range ports {
		if err := ValidateProbePort(port); err != nil {
			return err
		}
	}
	return nil
}

func hostWithoutBrackets(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host[1 : len(host)-1]
	}
	return host
}

func ProbeTCP(ctx context.Context, host string, port int, timeoutMs int, timeoutExplicit bool) PortProbeResult {
	result := PortProbeResult{Host: host, Port: port, Protocol: "TCP", ProbeType: "TCP"}
	if err := ValidateProbePort(port); err != nil {
		result.Message = err.Error()
		return result
	}
	normalized, err := normalizeProbeHost(host)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	timeout, err := ValidateProbeTimeout(timeoutMs, timeoutExplicit)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(probeCtx, "tcp", net.JoinHostPort(hostWithoutBrackets(normalized), fmt.Sprintf("%d", port)))
	latency := time.Since(start).Milliseconds()
	result.Host = normalized
	result.LatencyMs = latency
	if err == nil {
		_ = conn.Close()
		result.Reachable = true
		result.Message = fmt.Sprintf("port %d is reachable (%dms)", port, latency)
		return result
	}
	if probeCtx.Err() != nil {
		result.Message = fmt.Sprintf("port %d probe cancelled: %v", port, probeCtx.Err())
		return result
	}
	result.Message = fmt.Sprintf("port %d is not reachable: %v", port, err)
	return result
}

func ProbeHTTP(ctx context.Context, host string, port int, path string, timeoutMs int, timeoutExplicit bool) PortProbeResult {
	result := PortProbeResult{Host: host, Port: port, Protocol: "HTTP", ProbeType: "HTTP"}
	if err := ValidateProbePort(port); err != nil {
		result.Message = err.Error()
		return result
	}
	normalized, err := normalizeProbeHost(host)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	normalizedPath, rawQuery, err := ValidateHTTPPath(path)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	timeout, err := ValidateProbeTimeout(timeoutMs, timeoutExplicit)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	targetURL := url.URL{
		Scheme:   "http",
		Host:     net.JoinHostPort(hostWithoutBrackets(normalized), fmt.Sprintf("%d", port)),
		Path:     normalizedPath,
		RawQuery: rawQuery,
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	result.Host = normalized
	result.LatencyMs = latency
	if err != nil {
		result.Message = fmt.Sprintf("HTTP probe failed: %v", err)
		return result
	}
	defer resp.Body.Close()
	_, _ = ioCopyDiscard(probeCtx, resp.Body, MaxHTTPResponseSize)
	ok := resp.StatusCode >= 200 && resp.StatusCode < 400
	result.Reachable = ok
	result.HTTPStatus = resp.StatusCode
	result.Message = fmt.Sprintf("HTTP %d (%dms) %s", resp.StatusCode, latency, targetURL.String())
	return result
}

func ProbeTLS(ctx context.Context, host string, port int, timeoutMs int, timeoutExplicit bool) PortProbeResult {
	result := PortProbeResult{Host: host, Port: port, Protocol: "TLS", ProbeType: "TLS"}
	if err := ValidateProbePort(port); err != nil {
		result.Message = err.Error()
		return result
	}
	normalized, err := normalizeProbeHost(host)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	timeout, err := ValidateProbeTimeout(timeoutMs, timeoutExplicit)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	dialer := &net.Dialer{}
	conn, err := tlsDialContext(probeCtx, dialer, net.JoinHostPort(hostWithoutBrackets(normalized), fmt.Sprintf("%d", port)), hostWithoutBrackets(normalized))
	latency := time.Since(start).Milliseconds()
	result.Host = normalized
	result.LatencyMs = latency
	if err != nil {
		if probeCtx.Err() != nil {
			result.Message = fmt.Sprintf("TLS probe cancelled: %v", probeCtx.Err())
		} else {
			result.Message = fmt.Sprintf("TLS probe failed: %v", err)
		}
		return result
	}
	defer conn.Close()
	state := conn.ConnectionState()
	certInfo := "no certificate"
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
		certInfo = fmt.Sprintf("Subject: %s | Issuer: %s | Expires: %s (%d days left)",
			cert.Subject.String(), cert.Issuer.String(), cert.NotAfter.Format("2006-01-02"), daysLeft)
	}
	result.Reachable = true
	result.CertInfo = certInfo
	result.Message = fmt.Sprintf("TLS handshake succeeded (%dms)", latency)
	return result
}

func ProbeTCPBatch(ctx context.Context, host string, ports []int, timeoutMs int, timeoutExplicit bool) ([]PortProbeResult, error) {
	if err := ValidateBatchPorts(ports); err != nil {
		return nil, err
	}
	normalized, err := normalizeProbeHost(host)
	if err != nil {
		return nil, err
	}
	timeout, err := ValidateProbeTimeout(timeoutMs, timeoutExplicit)
	if err != nil {
		return nil, err
	}
	batchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	results := make([]PortProbeResult, len(ports))
	sem := make(chan struct{}, MaxBatchConcurrency)
	var wg sync.WaitGroup
	for i, port := range ports {
		wg.Add(1)
		go func(i, port int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-batchCtx.Done():
				results[i] = PortProbeResult{
					Host: normalized, Port: port, Protocol: "TCP", ProbeType: "TCP",
					Message: fmt.Sprintf("port %d probe cancelled: %v", port, batchCtx.Err()),
				}
				return
			}
			results[i] = ProbeTCP(batchCtx, normalized, port, timeoutMs, timeoutExplicit)
		}(i, port)
	}
	wg.Wait()
	return results, nil
}
