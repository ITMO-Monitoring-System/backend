package lecture

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/response"
	"monitoring_backend/internal/rabbit"
	"monitoring_backend/internal/ws"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type lectureLookup interface {
	GetByID(context.Context, int64) (domain.Lecture, error)
}

type Manager struct {
	mu sync.Mutex

	started  map[int64]bool // lecture_id → consumer started
	running  map[int64]context.CancelFunc
	hub      *ws.Hub
	amqpURL  string
	lectures lectureLookup
}

func NewManager(hub *ws.Hub, amqpURL string, lectures lectureLookup) *Manager {
	return &Manager{
		started:  make(map[int64]bool),
		running:  make(map[int64]context.CancelFunc),
		hub:      hub,
		amqpURL:  amqpURL,
		lectures: lectures,
	}
}

// CheckLecture is the internal preflight used before tracking opens an archive or queue.
func (m *Manager) CheckLecture(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_lecture_id")
		return
	}
	if m.validateLecture(w, r, id) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (m *Manager) validateLecture(w http.ResponseWriter, r *http.Request, id int64) bool {
	if id <= 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid_lecture_id")
		return false
	}
	_, err := m.lectures.GetByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		response.WriteError(w, http.StatusNotFound, "lecture_not_found")
		return false
	}
	if err != nil {
		slog.Error("lecture lookup failed", "lecture_id", id, "error", err)
		response.WriteError(w, http.StatusServiceUnavailable, "lecture_validation_unavailable")
		return false
	}
	return true
}

// StartLecture godoc
// @Summary Start lecture processing
// @Description Запускает обработку очереди RabbitMQ для лекции
// @Tags lecture
// @Accept json
// @Produce json
// @Param request body StartLectureRequest true "Lecture and RabbitMQ queue"
// @Success 202 {string} string "Consumer started"
// @Success 200 {string} string "Consumer already running"
// @Failure 400 {string} string "Invalid request body"
// @Failure 404 {object} response.ErrorResponse "Lecture not found"
// @Router /api/lecture/start [post]
func (m *Manager) StartLecture(w http.ResponseWriter, r *http.Request) {

	var req StartLectureRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Queue == "" {
		response.WriteError(w, http.StatusBadRequest, "Invalid queue name")
		return
	}
	if !m.validateLecture(w, r, req.LectureID) {
		return
	}
	if strings.TrimSpace(m.amqpURL) == "" {
		response.WriteError(w, http.StatusInternalServerError, "RabbitMQ amqp_url is not configured")
		return
	}

	m.mu.Lock()
	if m.started[req.LectureID] {
		m.mu.Unlock()
		response.WriteJSON(w, http.StatusOK, "Consumer already running")
		return
	}
	m.started[req.LectureID] = true

	ctx, cancel := context.WithCancel(context.Background())
	m.running[req.LectureID] = cancel

	m.mu.Unlock()

	slog.Info("lecture_consumer_started", "service", "backend", "event", "lecture_consumer_started", "lecture_id", req.LectureID, "queue", req.Queue)
	go rabbit.StartConsumer(ctx, m.amqpURL, req.Queue, req.LectureID, m.hub)

	response.WriteJSON(w, http.StatusOK, "Consumer started")
}

// StopLecture godoc
// @Summary Stop lecture processing
// @Description Stops RabbitMQ consumer and WebSocket broadcasting for the specified lecture.
// @Tags lecture
// @Accept json
// @Produce json
// @Param request body StopLectureRequest true "Lecture identifier"
// @Success 200 {string} string "ok"
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 404 {object} map[string]string "Lecture not found"
// @Router /api/lecture/stop [post]
func (m *Manager) StopLecture(w http.ResponseWriter, r *http.Request) {
	var req StopLectureRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.running[req.LectureID]; !ok {
		response.WriteError(w, http.StatusNotFound, "Lecture not found")
		return
	}

	m.running[req.LectureID]()
	delete(m.running, req.LectureID)
	delete(m.started, req.LectureID)
	slog.Info("lecture_consumer_stopped", "service", "backend", "event", "lecture_consumer_stopped", "lecture_id", req.LectureID)

	response.WriteJSON(w, http.StatusOK, "ok")
}
