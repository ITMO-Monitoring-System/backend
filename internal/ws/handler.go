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
// @Description Establishes a WebSocket connection for real-time lecture data streaming.
// @Description
// @Description Connection flow:
// @Description 1. Client opens WebSocket connection to this endpoint.
// @Description 2. After connection client sends control messages to manage subscriptions.
// @Description 3. Server sends data only for lectures the client is subscribed to.
// @Description
// @Description Client control messages:
// @Description - Subscribe to lecture: action=subscribe, lecture_id=<lecture identifier>
// @Description - Unsubscribe from lecture: action=unsubscribe, lecture_id=<lecture identifier>
// @Description
// @Description Subscription rules:
// @Description - One client may subscribe to multiple lectures.
// @Description - Unsubscribed lectures will no longer send data.
// @Description - If the client disconnects, all subscriptions are removed automatically.
// @Description
// @Description Server messages:
// @Description - Server sends lecture data as JSON messages.
// @Description - Message payload corresponds to data received from RabbitMQ.
// @Description - Exact message schema depends on the physical model and may be extended in the future.
// @Description
// @Description Future extensions:
// @Description - Additional message types (errors, control events).
// @Description - Extended payload schemas.
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
