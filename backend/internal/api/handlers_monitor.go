package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"port-master/backend/internal/monitor"
	"port-master/backend/internal/ports"
)

func registerMonitorRoutes(r chi.Router, registry *monitor.Registry, hub *monitor.Hub) {
	r.Post("/monitor/config", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Enabled bool                    `json:"enabled"`
			Ports   []ports.MonitorPortItem `json:"ports"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid monitor config")
			return
		}

		items, err := monitor.ParseMonitorItems(body.Ports)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		registry.Update(body.Enabled, items)
		if !body.Enabled {
			registry.ClearSnapshots()
		}

		WriteSuccess(w, map[string]interface{}{
			"enabled":       body.Enabled,
			"portCount":     len(items),
			"wsConnections": hub.ConnectionCount(),
		})
	})

	r.Get("/monitor/status", func(w http.ResponseWriter, r *http.Request) {
		WriteSuccess(w, map[string]interface{}{
			"enabled":       registry.Enabled(),
			"portCount":     registry.PortCount(),
			"wsConnections": hub.ConnectionCount(),
		})
	})
}

func writeProbeInputError(w http.ResponseWriter, err error) {
	if errors.Is(err, ports.ErrInvalidProbeInput) {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteError(w, http.StatusBadRequest, "invalid probe request")
}
