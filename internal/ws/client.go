package ws

import (
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
	hub  *Hub
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  hub,
	}
}

func (c *Client) Read() {
	defer func() {
		c.hub.RemoveClient(c)
		_ = c.conn.Close()
	}()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var cmd struct {
			Action    string `json:"action"`
			LectureID string `json:"lecture_id"`
		}

		if json.Unmarshal(msg, &cmd) != nil {
			continue
		}

		lectureID, err := strconv.ParseInt(cmd.LectureID, 10, 64)
		if err != nil {
			slog.Warn("ws_invalid_lecture_id", "service", "backend", "event", "ws_invalid_lecture_id")
			continue
		}

		if cmd.Action == "subscribe" {
			c.hub.Subscribe(c, lectureID)
			slog.Info("ws_subscribed", "service", "backend", "event", "ws_subscribed", "lecture_id", lectureID)
		}

		if cmd.Action == "unsubscribe" {
			c.hub.Unsubscribe(c, lectureID)
			slog.Info("ws_unsubscribed", "service", "backend", "event", "ws_unsubscribed", "lecture_id", lectureID)
		}
	}
}

func (c *Client) Write() {
	defer func() {
		_ = c.conn.Close()
	}()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
