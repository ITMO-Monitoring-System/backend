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
	InterruptRun(context.Context, int64, string) error
	ListUnfinishedRuns(context.Context) ([]domain.LectureRun, error)
	SetRunFinished(context.Context, int64, string, bool) error
	GetByID(context.Context, int64) (domain.Lecture, error)
}

type resultConsumer struct {
	queue  string
	runs   map[string]struct{}
	cancel context.CancelFunc
}

type Manager struct {
	mu sync.Mutex

	running  map[int64]resultConsumer
	hub      *ws.Hub
	amqpURL  string
	lectures lectureLookup
	consume  func(context.Context, string, int64)
}

func NewManager(hub *ws.Hub, amqpURL string, lectures lectureLookup) *Manager {
	return &Manager{
		running:  make(map[int64]resultConsumer),
		hub:      hub,
		amqpURL:  amqpURL,
		lectures: lectures,
		consume: func(ctx context.Context, queue string, id int64) {
			rabbit.StartConsumer(ctx, amqpURL, queue, id, hub)
		},
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
	current, ok := m.running[req.LectureID]
	if ok && current.queue != req.Queue {
		m.mu.Unlock()
		response.WriteError(w, http.StatusConflict, "Lecture already consumes a different queue")
		return
	}
	if err := m.lectures.SetRunFinished(r.Context(), req.LectureID, req.RunID, false); err != nil {
		m.mu.Unlock()
		if errors.Is(err, domain.ErrLectureRunFinished) {
			response.WriteJSON(w, http.StatusOK, "Run already finished")
			return
		}
		response.WriteError(w, http.StatusServiceUnavailable, "could not save lecture start")
		return
	}
	if ok {
		current.runs[req.RunID] = struct{}{}
		m.mu.Unlock()
		response.WriteJSON(w, http.StatusOK, "Consumer already running")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.running[req.LectureID] = resultConsumer{
		queue: req.Queue, runs: map[string]struct{}{req.RunID: {}}, cancel: cancel,
	}
	m.mu.Unlock()

	slog.Info("lecture_consumer_started", "service", "backend", "event", "lecture_consumer_started", "lecture_id", req.LectureID, "queue", req.Queue)
	go m.consume(ctx, req.Queue, req.LectureID)

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

	current, ok := m.running[req.LectureID]
	if err := m.lectures.SetRunFinished(r.Context(), req.LectureID, req.RunID, true); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.WriteError(w, http.StatusNotFound, "Lecture run not found")
		} else {
			response.WriteError(w, http.StatusServiceUnavailable, "could not save lecture stop")
		}
		return
	}
	if !ok {
		response.WriteJSON(w, http.StatusOK, "ok")
		return
	}

	delete(current.runs, req.RunID)
	if len(current.runs) == 0 {
		current.cancel()
		delete(m.running, req.LectureID)
		slog.Info("lecture_consumer_stopped", "service", "backend", "event", "lecture_consumer_stopped", "lecture_id", req.LectureID)
	}

	response.WriteJSON(w, http.StatusOK, "ok")
}
