package api

import (
	"net/http"
	"strings"
)

func (h *AuthHandler) WebSocketMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		if !h.validToken(token) {
			WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *AuthHandler) ValidToken(token string) bool {
	return h.validToken(token)
}
