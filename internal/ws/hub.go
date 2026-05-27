package ws

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

type VisitsService interface {
	AddUserVisitsLecture(ctx context.Context, userID string, lectureID int64) (*UserVisitsLectureResponse, error)
}

type lectureUserKey struct {
	lectureID int64
	userID    string
}

// dedupWindow — окно дедупликации одного и того же (lecture_id, user_id).
// Если в течение окна пришло повторное распознавание — INSERT и broadcast пропускаются.
// Управляется env VISITS_DEDUP_WINDOW_SEC (по умолчанию 60). 0 = выключить.
func dedupWindow() time.Duration {
	v := os.Getenv("VISITS_DEDUP_WINDOW_SEC")
	if v == "" {
		return 60 * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 60 * time.Second
	}
	return time.Duration(n) * time.Second
}

type Hub struct {
	mu   sync.Mutex
	serv VisitsService
	// lecture_id → clients
	lectures map[int64]map[*Client]bool

	dedupMu     sync.Mutex
	lastSeen    map[lectureUserKey]time.Time
	dedupWindow time.Duration
}

func NewHub(serv VisitsService) *Hub {
	return &Hub{
		lectures:    make(map[int64]map[*Client]bool),
		serv:        serv,
		lastSeen:    make(map[lectureUserKey]time.Time),
		dedupWindow: dedupWindow(),
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

// shouldDedup возвращает true, если этот (lecture_id, user_id) уже отмечался
// внутри окна дедупликации. При false дополнительно обновляет lastSeen.
func (h *Hub) shouldDedup(lectureID int64, userID string) bool {
	if h.dedupWindow <= 0 || userID == "" {
		return false
	}
	key := lectureUserKey{lectureID: lectureID, userID: userID}
	now := time.Now()
	h.dedupMu.Lock()
	defer h.dedupMu.Unlock()
	if ts, ok := h.lastSeen[key]; ok && now.Sub(ts) < h.dedupWindow {
		// Обновляем lastSeen, чтобы окно "скользило" пока человек продолжает видеться.
		h.lastSeen[key] = now
		return true
	}
	h.lastSeen[key] = now
	return false
}

// pruneDedup чистит lastSeen от записей старше окна — вызывается периодически из background-горутины.
func (h *Hub) pruneDedup() {
	if h.dedupWindow <= 0 {
		return
	}
	cutoff := time.Now().Add(-h.dedupWindow * 2)
	h.dedupMu.Lock()
	defer h.dedupMu.Unlock()
	for k, ts := range h.lastSeen {
		if ts.Before(cutoff) {
			delete(h.lastSeen, k)
		}
	}
}

// StartDedupGC запускает фоновую горутину чистки lastSeen — вызывать один раз при старте app.
func (h *Hub) StartDedupGC(ctx context.Context) {
	if h.dedupWindow <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				h.pruneDedup()
			}
		}
	}()
}

func (h *Hub) Broadcast(lectureID int64, data []byte) {
	request := struct {
		LectureID int64  `json:"lecture_id"`
		PersonID  string `json:"person_id"`
	}{}

	if err := json.Unmarshal(data, &request); err != nil {
		return
	}

	// Пустой PersonID = face-recognizing не нашёл совпадения. Раньше шёл дальше и
	// падал на FK-ограничении в visits.lectures_visiting. Просто выходим — это
	// нормальный сигнал, не ошибка.
	if request.PersonID == "" {
		return
	}

	// Dedup: если этот же студент уже отмечен в текущем окне — пропускаем INSERT
	// и не рассылаем повторное событие. lastSeen скользит дальше.
	if h.shouldDedup(request.LectureID, request.PersonID) {
		return
	}

	// 1. Snapshot клиентов
	h.mu.Lock()
	clientsMap := h.lectures[lectureID]
	clients := make([]*Client, 0, len(clientsMap))
	for c := range clientsMap {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	// 2. INSERT и fan-out — в отдельной горутине, чтобы consumer RabbitMQ не блокировался
	// на БД-операциях и сетевой отправке. На случай большой нагрузки goroutine остаётся
	// дешёвой; для жёсткого ограничения числа inflight операций можно добавить семафор.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()

		user, err := h.serv.AddUserVisitsLecture(ctx, request.PersonID, request.LectureID)
		if err != nil {
			log.Printf("ERROR: apply time of visit for %s - %v", request.PersonID, err)
			return
		}

		payload, err := json.Marshal(user)
		if err != nil {
			return
		}

		for _, c := range clients {
			// Non-blocking send: если у клиента полный буфер — пропускаем, чтобы один
			// зависший подписчик не блокировал рассылку остальным.
			select {
			case c.send <- payload:
			default:
				log.Printf("WARN: ws client %s buffer full, dropping message", c.conn.RemoteAddr().String())
			}
		}
	}()
}
