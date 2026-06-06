package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"port-master/backend/internal/ports"
)

func registerPortRoutes(r chi.Router, service *ports.Service) {
	r.Get("/ports/scan", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.ScanAllPorts(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/query/range", func(w http.ResponseWriter, r *http.Request) {
		start, err := queryInt(r, "start", 0)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		end, err := queryInt(r, "end", 0)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := service.QueryByRange(r.Context(), start, end)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/query/process", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.QueryByProcessName(r.Context(), r.URL.Query().Get("name"))
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/query/pid/{pid}", func(w http.ResponseWriter, r *http.Request) {
		pid, err := pathInt64(r, "pid")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := service.QueryByPID(r.Context(), pid)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/query/{port}", func(w http.ResponseWriter, r *http.Request) {
		port, err := pathInt(r, "port")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := service.QueryByPort(r.Context(), port)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/query", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.QueryByPorts(r.Context(), r.URL.Query().Get("ports"))
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/free", func(w http.ResponseWriter, r *http.Request) {
		start, err := queryInt(r, "start", 8080)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		count, err := queryInt(r, "count", 5)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := service.GenerateFreePorts(r.Context(), start, count)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/conflicts", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.DetectConflicts(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/summary", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Summary(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/ports/probe", func(w http.ResponseWriter, r *http.Request) {
		port, err := queryInt(r, "port", 0)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		timeout, err := queryInt(r, "timeout", 3000)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		WriteSuccess(w, ports.ProbeTCP(r.URL.Query().Get("host"), port, timeout))
	})

	r.Post("/ports/probe/batch", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Host    string `json:"host"`
			Ports   []int  `json:"ports"`
			Timeout int    `json:"timeout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid probe request")
			return
		}
		if body.Timeout == 0 {
			body.Timeout = 3000
		}
		results := make([]ports.PortProbeResult, 0, len(body.Ports))
		for _, port := range body.Ports {
			results = append(results, ports.ProbeTCP(body.Host, port, body.Timeout))
		}
		WriteSuccess(w, results)
	})

	r.Post("/ports/monitor", func(w http.ResponseWriter, r *http.Request) {
		var body ports.PortMonitorRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid monitor request")
			return
		}
		result, err := service.MonitorPorts(r.Context(), body)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})
}

func queryInt(r *http.Request, key string, defaultValue int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(raw)
}

func pathInt(r *http.Request, key string) (int, error) {
	return strconv.Atoi(chi.URLParam(r, key))
}

func pathInt64(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, key), 10, 64)
}
