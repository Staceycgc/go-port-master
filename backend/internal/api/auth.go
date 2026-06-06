package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

type AuthConfig struct {
	Enabled bool
	Token   string
}

type AuthHandler struct {
	config AuthConfig
}

func NewAuthHandler(config AuthConfig) *AuthHandler {
	return &AuthHandler{config: config}
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, map[string]interface{}{
		"authRequired": h.config.Enabled,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.config.Enabled {
		WriteSuccess(w, map[string]interface{}{
			"authenticated": true,
			"authRequired":  false,
		})
		return
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid login request")
		return
	}

	if !h.validToken(body.Token) {
		WriteError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"authenticated": true,
		"authRequired":  true,
	})
}

func (h *AuthHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		token := bearerToken(r.Header.Get("Authorization"))
		if !h.validToken(token) {
			WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *AuthHandler) validToken(token string) bool {
	if !h.config.Enabled {
		return true
	}
	if h.config.Token == "" || token == "" {
		return false
	}
	if len(token) != len(h.config.Token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(h.config.Token)) == 1
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
