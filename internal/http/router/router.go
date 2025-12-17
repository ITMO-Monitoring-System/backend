package router

import (
	"monitoring_backend/internal/config"
	"monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/response"
	"net/http"

	"github.com/gorilla/mux"
)

func New(cfg *config.Config, h *handlers.Handler) *mux.Router {
	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, http.StatusNotFound, "not_found")
	})

	r.HandleFunc("/health", h.Health).Methods(http.MethodGet)

	// TODO: add endpoints
	api := r.PathPrefix("/api").Subrouter()
	_ = api

	return r
}
