package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"port-master/backend/internal/api"
	appconfig "port-master/backend/internal/config"
)

func main() {
	host := flag.String("host", appconfig.EnvString("PORT_MASTER_HOST", "127.0.0.1"), "address to bind")
	port := flag.Int("port", mustEnvInt("PORT_MASTER_PORT", 8080), "port to bind")
	token := flag.String("token", "", "authentication token")
	noAuth := flag.Bool("no-auth", false, "disable token authentication")

	scanCacheTTL := flag.Int64("scan-cache-ttl-ms", mustEnvInt64("PORT_MASTER_SCAN_CACHE_TTL_MS", appconfig.Default().ScanCacheTTLMs),
		"scan result cache TTL in ms (0 disables cache)")
	monitorPoll := flag.Int64("monitor-poll-ms", mustEnvInt64("PORT_MASTER_MONITOR_POLL_MS", appconfig.Default().MonitorPollIntervalMs),
		"background monitor poll interval in ms")
	sshConnectMs := flag.Int("ssh-connect-timeout-ms", mustEnvInt("PORT_MASTER_SSH_CONNECT_TIMEOUT_MS", appconfig.Default().SSHConnectTimeoutMs),
		"SSH TCP connect and handshake timeout in ms")
	sshCommandSec := flag.Int("ssh-command-timeout-sec", mustEnvInt("PORT_MASTER_SSH_COMMAND_TIMEOUT_SEC", appconfig.Default().SSHCommandTimeoutSec),
		"SSH remote command execution timeout in seconds")

	flag.Parse()

	appCfg, err := appconfig.Config{
		ScanCacheTTLMs:        *scanCacheTTL,
		MonitorPollIntervalMs: *monitorPoll,
		SSHConnectTimeoutMs:   *sshConnectMs,
		SSHCommandTimeoutSec:  *sshCommandSec,
	}.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(2)
	}

	authEnabled := !*noAuth
	authToken := firstNonEmpty(*token, os.Getenv("PORT_MASTER_TOKEN"))
	generatedToken := false
	if authEnabled && authToken == "" {
		authToken = generateToken()
		generatedToken = true
	}

	serverConfig := api.ServerConfig{
		Auth: api.AuthConfig{
			Enabled: authEnabled,
			Token:   authToken,
		},
		App: appCfg,
	}

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	appServer, err := api.NewServer(serverConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(2)
	}
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           appServer.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	printStartup(*host, *port, authEnabled, generatedToken, authToken, appCfg)
	defer appServer.Close()
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func mustEnvInt(key string, fallback int) int {
	value, err := appconfig.EnvInt(key, fallback)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(2)
	}
	return value
}

func mustEnvInt64(key string, fallback int64) int64 {
	value, err := appconfig.EnvInt64(key, fallback)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(2)
	}
	return value
}

func printStartup(host string, port int, authEnabled bool, generatedToken bool, token string, cfg appconfig.Config) {
	displayHost := host
	if host == "0.0.0.0" || host == "::" {
		displayHost = "127.0.0.1"
	}
	fmt.Printf("Port Master %s listening on http://%s\n", appconfig.Version, net.JoinHostPort(displayHost, strconv.Itoa(port)))
	fmt.Printf("Scan cache TTL: %d ms | Monitor poll: %d ms | SSH connect: %d ms | SSH command: %d s\n",
		cfg.ScanCacheTTLMs, cfg.MonitorPollIntervalMs, cfg.SSHConnectTimeoutMs, cfg.SSHCommandTimeoutSec)
	if authEnabled {
		if generatedToken {
			fmt.Printf("One-time token: %s\n", token)
		} else {
			fmt.Println("Authentication enabled.")
		}
		return
	}
	fmt.Println("Authentication disabled.")
}

func generateToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
