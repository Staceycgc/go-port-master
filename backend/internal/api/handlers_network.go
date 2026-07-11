package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"port-master/backend/internal/network"
)

func registerNetworkRoutes(r chi.Router, service *network.Service) {
	r.Get("/network/interfaces", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.ListInterfaces(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "failed to list network interfaces")
			return
		}
		WriteSuccess(w, result)
	})
}
