package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"port-master/backend/internal/k8s"
)

func registerK8sRoutes(r chi.Router, client *k8s.Client) {
	r.Get("/k8s/available", func(w http.ResponseWriter, r *http.Request) {
		WriteSuccess(w, client.Available())
	})

	r.Get("/k8s/context", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := contextWithTimeout(r, k8s.DefaultTimeout())
		defer cancel()
		contextName, err := client.CurrentContext(ctx)
		if err != nil {
			writeK8sError(w, err)
			return
		}
		WriteSuccess(w, contextName)
	})

	r.Get("/k8s/pods", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.URL.Query().Get("namespace")
		ctx, cancel := contextWithTimeout(r, k8s.DefaultTimeout())
		defer cancel()
		pods, err := client.ListPods(ctx, namespace)
		if err != nil {
			writeK8sError(w, err)
			return
		}
		WriteSuccess(w, pods)
	})

	r.Get("/k8s/services", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.URL.Query().Get("namespace")
		ctx, cancel := contextWithTimeout(r, k8s.DefaultTimeout())
		defer cancel()
		services, err := client.ListServices(ctx, namespace)
		if err != nil {
			writeK8sError(w, err)
			return
		}
		WriteSuccess(w, services)
	})

	r.Get("/k8s/summary", func(w http.ResponseWriter, r *http.Request) {
		namespace := r.URL.Query().Get("namespace")
		ctx, cancel := contextWithTimeout(r, k8s.DefaultTimeout())
		defer cancel()
		summary, err := client.Summary(ctx, namespace)
		if err != nil {
			writeK8sError(w, err)
			return
		}
		WriteSuccess(w, summary)
	})
}
