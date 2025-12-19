package ws

import "sync"

type Hub struct {
	mu sync.Mutex

	// lecture_id → clients
	lectures map[int64]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		lectures: make(map[int64]map[*Client]bool),
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

	// 2. отправка БЕЗ mutex
	for _, c := range clients {
		c.send <- data
	}
}
