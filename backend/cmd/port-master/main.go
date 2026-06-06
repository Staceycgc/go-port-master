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
)

func main() {
	host := flag.String("host", envString("PORT_MASTER_HOST", "127.0.0.1"), "address to bind")
	port := flag.Int("port", envInt("PORT_MASTER_PORT", 8080), "port to bind")
	token := flag.String("token", "", "authentication token")
	noAuth := flag.Bool("no-auth", false, "disable token authentication")
	flag.Parse()

	authEnabled := !*noAuth
	authToken := firstNonEmpty(*token, os.Getenv("PORT_MASTER_TOKEN"))
	generatedToken := false
	if authEnabled && authToken == "" {
		authToken = generateToken()
		generatedToken = true
	}

	config := api.ServerConfig{
		Auth: api.AuthConfig{
			Enabled: authEnabled,
			Token:   authToken,
		},
	}

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(config),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	printStartup(*host, *port, authEnabled, generatedToken, authToken)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func printStartup(host string, port int, authEnabled bool, generatedToken bool, token string) {
	displayHost := host
	if host == "0.0.0.0" || host == "::" {
		displayHost = "127.0.0.1"
	}
	fmt.Printf("Port Master listening on http://%s\n", net.JoinHostPort(displayHost, strconv.Itoa(port)))
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

func envString(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
