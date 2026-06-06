package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"port-master/backend/internal/processes"
)

func registerProcessRoutes(r chi.Router, service *processes.Service) {
	r.Get("/process/list", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.ListAllProcesses(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Post("/process/kill/batch", func(w http.ResponseWriter, r *http.Request) {
		var body processes.KillProcessRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid kill request")
			return
		}
		WriteSuccess(w, service.KillProcesses(body.PIDs, body.Force))
	})

	r.Delete("/process/by-port/{port}", func(w http.ResponseWriter, r *http.Request) {
		port, err := pathInt(r, "port")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		force := r.URL.Query().Get("force") == "true"
		result, err := service.KillByPort(r.Context(), port, force)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Delete("/process/by-port/{port}/force", func(w http.ResponseWriter, r *http.Request) {
		port, err := pathInt(r, "port")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := service.KillByPort(r.Context(), port, true)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/process/{pid}", func(w http.ResponseWriter, r *http.Request) {
		pid, err := pathInt64(r, "pid")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := service.Detail(r.Context(), pid)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Delete("/process/{pid}", func(w http.ResponseWriter, r *http.Request) {
		pid, err := pathInt64(r, "pid")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := processes.KillProcess(pid, false); err != nil {
			WriteError(w, http.StatusInternalServerError, fmt.Sprintf("process %d termination failed: %s", pid, err.Error()))
			return
		}
		WriteSuccess(w, fmt.Sprintf("process %d terminated", pid))
	})

	r.Delete("/process/{pid}/force", func(w http.ResponseWriter, r *http.Request) {
		pid, err := pathInt64(r, "pid")
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := processes.KillProcess(pid, true); err != nil {
			WriteError(w, http.StatusInternalServerError, fmt.Sprintf("process %d force termination failed: %s", pid, err.Error()))
			return
		}
		WriteSuccess(w, fmt.Sprintf("process %d force terminated", pid))
	})
}
