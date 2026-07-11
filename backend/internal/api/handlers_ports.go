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
		forceRefresh := r.URL.Query().Get("refresh") == "true"
		result, err := service.ScanAllPortsRefresh(r.Context(), forceRefresh)
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
		if err := ports.ValidateProbePort(port); err != nil {
			writeProbeInputError(w, err)
			return
		}
		host, err := ports.ResolveProbeHost(r.URL.Query().Get("host"))
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		timeoutMs, timeoutExplicit, err := queryProbeTimeout(r)
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		WriteSuccess(w, ports.ProbeTCP(r.Context(), host, port, timeoutMs, timeoutExplicit))
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
		if err := ports.ValidateBatchPorts(body.Ports); err != nil {
			writeProbeInputError(w, err)
			return
		}
		host, err := ports.ResolveProbeHost(body.Host)
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		timeoutExplicit := body.Timeout != 0
		results, err := ports.ProbeTCPBatch(r.Context(), host, body.Ports, body.Timeout, timeoutExplicit)
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		WriteSuccess(w, results)
	})

	r.Get("/ports/probe/tls", func(w http.ResponseWriter, r *http.Request) {
		port, err := queryInt(r, "port", 0)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := ports.ValidateProbePort(port); err != nil {
			writeProbeInputError(w, err)
			return
		}
		host, err := ports.ResolveProbeHost(r.URL.Query().Get("host"))
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		timeoutMs, timeoutExplicit, err := queryProbeTimeout(r)
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		WriteSuccess(w, ports.ProbeTLS(r.Context(), host, port, timeoutMs, timeoutExplicit))
	})

	r.Get("/ports/probe/http", func(w http.ResponseWriter, r *http.Request) {
		port, err := queryInt(r, "port", 0)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := ports.ValidateProbePort(port); err != nil {
			writeProbeInputError(w, err)
			return
		}
		host, err := ports.ResolveProbeHost(r.URL.Query().Get("host"))
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		path, err := probeHTTPPath(r)
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		timeoutMs, timeoutExplicit, err := queryProbeTimeout(r)
		if err != nil {
			writeProbeInputError(w, err)
			return
		}
		WriteSuccess(w, ports.ProbeHTTP(r.Context(), host, port, path, timeoutMs, timeoutExplicit))
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

func queryProbeTimeout(r *http.Request) (timeoutMs int, explicit bool, err error) {
	raw := r.URL.Query().Get("timeout")
	if raw == "" {
		return 0, false, nil
	}
	timeoutMs, err = strconv.Atoi(raw)
	if err != nil {
		return 0, true, ports.ErrInvalidProbeInput
	}
	if _, err := ports.ValidateProbeTimeout(timeoutMs, true); err != nil {
		return 0, true, err
	}
	return timeoutMs, true, nil
}

func probeHTTPPath(r *http.Request) (string, error) {
	path := r.URL.Query().Get("path")
	if path == "" {
		return "/", nil
	}
	normalizedPath, rawQuery, err := ports.ValidateHTTPPath(path)
	if err != nil {
		return "", err
	}
	if rawQuery != "" {
		return normalizedPath + "?" + rawQuery, nil
	}
	return normalizedPath, nil
}
