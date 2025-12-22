package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type VisitsService interface {
	AddUserVisitsLecture(ctx context.Context, userID string, lectureID int64) (*UserVisitsLectureResponse, error)
}

type Hub struct {
	mu   sync.Mutex
	serv VisitsService
	// lecture_id → clients
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

func (h *Hub) Broadcast(lectureID int64, data []byte) {
	// 1. snapshot клиентов
	h.mu.Lock()
	clientsMap := h.lectures[lectureID]

	clients := make([]*Client, 0, len(clientsMap))
	for c := range clientsMap {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	request := struct {
		LectureID int64  `json:"lecture_id"`
		PersonID  string `json:"person_id"`
	}{}

	err := json.Unmarshal(data, &request)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	user, err := h.serv.AddUserVisitsLecture(ctx, request.PersonID, request.LectureID)
	if err != nil {
		return
	}

	data, err = json.Marshal(user)
	if err != nil {
		return
	}

	// 2. отправка БЕЗ mutex
	for _, c := range clients {
		c.send <- data
	}
}
