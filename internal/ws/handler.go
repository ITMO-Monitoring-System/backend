package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocketHandler godoc
// @Summary WebSocket connection for lecture streaming
// @Description Establishes a WebSocket connection.
// @Description After connection the client sends JSON messages:
// @Description - action=subscribe with lecture_id to start receiving data
// @Description - action=unsubscribe with lecture_id to stop receiving data
// @Description The server sends lecture data only for subscribed lectures.
// @Tags websocket
// @Produce application/json
// @Success 101 {string} string "Switching Protocols"
// @Router /ws [get]
func Handler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := NewClient(conn, hub)

		go client.Read()
		go client.Write()
	}
}
