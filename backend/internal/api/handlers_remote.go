package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func registerRemoteRoutes(r chi.Router) {
	r.Post("/remote/scan", notImplemented)
	r.Post("/remote/kill", notImplemented)
	r.Post("/remote/test", notImplemented)
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "remote management is not implemented")
}
