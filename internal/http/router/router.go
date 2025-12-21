package router

import (
	"monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/handlers/user"
	"monitoring_backend/internal/http/response"
	"monitoring_backend/internal/lecture"
	"monitoring_backend/internal/ws"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func New(h *handlers.Handler, wsHub *ws.Hub, lectureManager *lecture.Manager, userHandler *user.UserHandler) *mux.Router {
	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, http.StatusNotFound, "not_found")
	})

	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/health", h.Health).Methods(http.MethodGet)
	api.HandleFunc("/ws", ws.Handler(wsHub))

	// lectures
	lectureGroup := api.PathPrefix("/lecture").Subrouter()
	lectureGroup.HandleFunc("/start", lectureManager.StartLecture).Methods(http.MethodPost)
	lectureGroup.HandleFunc("/stop", lectureManager.StopLecture).Methods(http.MethodPost)

	// cores
	userGroup := api.PathPrefix("/user").Subrouter()
	userGroup.HandleFunc("/create", userHandler.AddUser).Methods(http.MethodPost)

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return r
}
