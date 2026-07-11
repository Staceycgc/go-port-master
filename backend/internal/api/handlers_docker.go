package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"port-master/backend/internal/docker"
	"port-master/backend/internal/k8s"
	"port-master/backend/internal/remote"
)

func registerRemoteRoutes(r chi.Router, service *remote.Service) {
	r.Post("/remote/scan", func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeRemoteRequest(r)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := service.ScanPorts(r.Context(), req)
		if err != nil {
			writeRemoteError(w, err)
			return
		}
		WriteSuccess(w, result)
	})

	r.Post("/remote/kill", func(w http.ResponseWriter, r *http.Request) {
		var body remote.KillRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid remote kill request")
			return
		}
		ok, err := service.KillProcess(r.Context(), body)
		if err != nil {
			writeRemoteError(w, err)
			return
		}
		WriteSuccess(w, ok)
	})

	r.Post("/remote/test", func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeRemoteRequest(r)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		ok, err := service.TestConnection(r.Context(), req)
		if err != nil {
			writeRemoteError(w, err)
			return
		}
		WriteSuccess(w, ok)
	})

	r.Post("/remote/info", func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeRemoteRequest(r)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		info, err := service.SystemInfo(r.Context(), req)
		if err != nil {
			writeRemoteError(w, err)
			return
		}
		WriteSuccess(w, info)
	})
}

func decodeRemoteRequest(r *http.Request) (remote.HostRequest, error) {
	var req remote.HostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, errors.New("invalid remote host request")
	}
	return req, nil
}

func writeRemoteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, remote.ErrInvalidInput):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, remote.ErrAuthFailed):
		WriteError(w, http.StatusUnauthorized, "ssh authentication failed")
	case errors.Is(err, remote.ErrConnect):
		WriteError(w, http.StatusBadGateway, "ssh connection failed")
	default:
		WriteError(w, http.StatusInternalServerError, "remote operation failed")
	}
}

func registerDockerRoutes(r chi.Router, service *docker.Service) {
	r.Get("/docker/available", func(w http.ResponseWriter, r *http.Request) {
		WriteSuccess(w, service.Available())
	})

	r.Get("/docker/containers", func(w http.ResponseWriter, r *http.Request) {
		all := r.URL.Query().Get("all") == "true"
		ctx, cancel := contextWithTimeout(r, docker.DefaultTimeout())
		defer cancel()
		result, err := service.ListContainers(ctx, all)
		if err != nil {
			writeDockerError(w, err)
			return
		}
		WriteSuccess(w, result)
	})

	r.Post("/docker/stop", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ContainerID string `json:"containerId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid docker stop request")
			return
		}
		ctx, cancel := contextWithTimeout(r, docker.DefaultTimeout())
		defer cancel()
		message, err := service.StopContainer(ctx, body.ContainerID)
		if err != nil {
			writeDockerError(w, err)
			return
		}
		WriteSuccess(w, message)
	})

	r.Post("/docker/restart", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ContainerID string `json:"containerId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid docker restart request")
			return
		}
		ctx, cancel := contextWithTimeout(r, docker.DefaultTimeout())
		defer cancel()
		message, err := service.RestartContainer(ctx, body.ContainerID)
		if err != nil {
			writeDockerError(w, err)
			return
		}
		WriteSuccess(w, message)
	})
}

func writeDockerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, docker.ErrUnavailable):
		WriteError(w, http.StatusServiceUnavailable, "docker is not available")
	case errors.Is(err, docker.ErrCommandFailed):
		WriteError(w, http.StatusBadGateway, err.Error())
	default:
		msg := err.Error()
		if msg == "invalid container id" || msg == "container id is required" {
			WriteError(w, http.StatusBadRequest, msg)
			return
		}
		WriteError(w, http.StatusInternalServerError, msg)
	}
}

func contextWithTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := r.Context().Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}
	return context.WithTimeout(r.Context(), timeout)
}

func writeK8sError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, k8s.ErrUnavailable):
		WriteError(w, http.StatusServiceUnavailable, "kubectl is not available")
	case errors.Is(err, k8s.ErrCommandFailed):
		WriteError(w, http.StatusBadGateway, err.Error())
	default:
		msg := err.Error()
		if msg == "invalid namespace" {
			WriteError(w, http.StatusBadRequest, msg)
			return
		}
		WriteError(w, http.StatusInternalServerError, msg)
	}
}
