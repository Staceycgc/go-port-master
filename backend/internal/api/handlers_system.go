package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"port-master/backend/internal/system"
)

func registerSystemRoutes(r chi.Router, service *system.Service, authRequired bool) {
	r.Get("/system/stats", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Stats(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, result)
	})

	r.Get("/system/info", func(w http.ResponseWriter, r *http.Request) {
		WriteSuccess(w, system.Info(authRequired))
	})
}
