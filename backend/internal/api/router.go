package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"port-master/backend/internal/ports"
	"port-master/backend/internal/processes"
	"port-master/backend/internal/system"
	"port-master/backend/internal/web"
)

type ServerConfig struct {
	Auth AuthConfig
}

func NewRouter(config ServerConfig) http.Handler {
	portService := ports.NewService()
	processService := processes.NewService(portService)
	systemService := system.NewService(portService)
	authHandler := NewAuthHandler(config.Auth)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(corsMiddleware)

	router.Route("/api", func(r chi.Router) {
		r.Get("/auth/status", authHandler.Status)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(authHandler.Middleware)
			registerPortRoutes(r, portService)
			registerProcessRoutes(r, processService)
			registerSystemRoutes(r, systemService, config.Auth.Enabled)
			registerRemoteRoutes(r)
		})
	})

	spa := spaHandler()
	router.Get("/", spa.ServeHTTP)
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			WriteError(w, http.StatusNotFound, "api endpoint not found")
			return
		}
		spa.ServeHTTP(w, r)
	})
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			WriteError(w, http.StatusNotFound, "api endpoint not found")
			return
		}
		spa.ServeHTTP(w, r)
	})

	return router
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func spaHandler() http.Handler {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			WriteError(w, http.StatusInternalServerError, "web assets are not available")
		})
	}
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if file, err := dist.Open(path); err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r)
	})
}
