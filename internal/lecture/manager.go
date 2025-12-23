package lecture

import (
	"context"
	"encoding/json"
	"monitoring_backend/internal/http/response"
	"net/http"
	"sync"

	"monitoring_backend/internal/rabbit"
	"monitoring_backend/internal/ws"
)

type Manager struct {
	mu sync.Mutex

	started map[int64]bool // lecture_id → consumer started
	running map[int64]context.CancelFunc
	hub     *ws.Hub
	amqpURL string
}

func NewManager(hub *ws.Hub, amqpURL string) *Manager {
	return &Manager{
		started: make(map[int64]bool),
		running: make(map[int64]context.CancelFunc),
		hub:     hub,
		amqpURL: amqpURL,
	}
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

	response.WriteJSON(w, http.StatusOK, "ok")
}
