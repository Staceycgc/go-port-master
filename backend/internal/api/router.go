package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"port-master/backend/internal/config"
	"port-master/backend/internal/docker"
	"port-master/backend/internal/k8s"
	"port-master/backend/internal/monitor"
	"port-master/backend/internal/network"
	"port-master/backend/internal/ports"
	"port-master/backend/internal/processes"
	"port-master/backend/internal/remote"
	"port-master/backend/internal/system"
	"port-master/backend/internal/web"
)

type ServerConfig struct {
	Auth AuthConfig
	App  config.Config
}

type Server struct {
	router    http.Handler
	scheduler *monitor.Scheduler
	hub       *monitor.Hub
}

func NewServer(serverConfig ServerConfig) (*Server, error) {
	appConfig, err := serverConfig.App.Validate()
	if err != nil {
		return nil, err
	}

	portService := ports.NewServiceWithOptions(
		ports.GopsutilScanner{},
		appConfig.ScanCacheTTL(),
	)
	processService := processes.NewService(portService)
	systemService := system.NewService(portService)
	authHandler := NewAuthHandler(serverConfig.Auth)

	remoteService := remote.NewService(appConfig)
	dockerService := docker.NewService()
	k8sClient := k8s.NewClient()
	networkService := network.NewService()

	monitorRegistry := monitor.NewRegistry()
	monitorHub := monitor.NewHub()
	monitorScheduler := monitor.NewScheduler(
		monitorRegistry,
		monitorHub,
		portService,
		appConfig.MonitorPollInterval(),
		monitor.DefaultPollTimeout,
	)
	monitorScheduler.Start()
	monitorWS := monitor.NewWSHandler(monitorHub)

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
			registerSystemRoutes(r, systemService, serverConfig.Auth.Enabled, appConfig)
			registerRemoteRoutes(r, remoteService)
			registerDockerRoutes(r, dockerService)
			registerK8sRoutes(r, k8sClient)
			registerNetworkRoutes(r, networkService)
			registerMonitorRoutes(r, monitorRegistry, monitorHub)
		})
	})

	router.Group(func(r chi.Router) {
		r.Use(authHandler.WebSocketMiddleware)
		r.Get("/ws/monitor", monitorWS.ServeHTTP)
	})

	spa := spaHandler()
	router.Get("/", spa.ServeHTTP)
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			WriteError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		spa.ServeHTTP(w, r)
	})
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			WriteError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		spa.ServeHTTP(w, r)
	})

	return &Server{router: router, scheduler: monitorScheduler, hub: monitorHub}, nil
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) Close() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
	if s.hub != nil {
		s.hub.Shutdown()
	}
}

func NewRouter(serverConfig ServerConfig) (http.Handler, error) {
	server, err := NewServer(serverConfig)
	if err != nil {
		return nil, err
	}
	return server.Handler(), nil
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
