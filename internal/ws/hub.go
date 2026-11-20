package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"monitoring_backend/internal/domain"
)

type VisitsService interface {
	AddUserVisitsLecture(context.Context, domain.LectureVisit) (*UserVisitsLectureResponse, error)
}

type Hub struct {
	mu       sync.Mutex
	serv     VisitsService
	lectures map[int64]map[*Client]bool
}

func NewHub(serv VisitsService) *Hub {
	return &Hub{
		lectures: make(map[int64]map[*Client]bool),
		serv:     serv,
	}
}

func (h *Hub) Subscribe(c *Client, lectureID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.lectures[lectureID] == nil {
		h.lectures[lectureID] = make(map[*Client]bool)
	}
	h.lectures[lectureID][c] = true
}

func (h *Hub) RemoveClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, clients := range h.lectures {
		delete(clients, c)
	}
}

func (h *Hub) Unsubscribe(c *Client, lectureID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.lectures[lectureID]; ok {
		delete(clients, c)

		// если клиентов больше нет — чистим мапу
		if len(clients) == 0 {
			delete(h.lectures, lectureID)
		}
	}
}

// RecordObservation commits the visit before notifying WebSocket subscribers.
func (h *Hub) RecordObservation(ctx context.Context, visit domain.LectureVisit) error {
	start := time.Now()
	user, err := h.serv.AddUserVisitsLecture(ctx, visit)
	if err != nil {
		slog.Error("visit_write_failed", "service", "backend", "event", "visit_write_failed",
			"lecture_id", visit.LectureID, "run_id", visit.RunID, "request_id", visit.RequestID, "error_type", fmt.Sprintf("%T", err))
		return err
	}
	if user == nil {
		slog.Info("visit_deduplicated", "service", "backend", "event", "visit_deduplicated",
			"lecture_id", visit.LectureID, "run_id", visit.RunID, "request_id", visit.RequestID)
		return nil
	}
	slog.Info("visit_written", "service", "backend", "event", "visit_written", "lecture_id", visit.LectureID,
		"run_id", visit.RunID, "request_id", visit.RequestID, "person_id", visit.UserID, "duration_ms", time.Since(start).Milliseconds())

	payload, err := json.Marshal(user)
	if err != nil {
		slog.Error("ws_payload_failed", "lecture_id", visit.LectureID, "request_id", visit.RequestID, "error", err)
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.lectures[visit.LectureID]
	queued, dropped := 0, 0
	for client := range clients {
		select {
		case client.send <- payload:
			queued++
		default:
			dropped++
		}
	}
	slog.Info("ws_fanout_summary", "service", "backend", "event", "ws_fanout_summary",
		"lecture_id", visit.LectureID, "run_id", visit.RunID, "request_id", visit.RequestID,
		"person_id", visit.UserID, "subscribers", len(clients), "queued", queued, "dropped", dropped)
	return nil
}
