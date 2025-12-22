package ws

import (
	"encoding/json"
	"log"

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
		log.Printf("INFO: recv from conn: %s", msg)

		var cmd struct {
			Action    string `json:"action"`
			LectureID int64  `json:"lecture_id"`
		}

		if json.Unmarshal(msg, &cmd) != nil {
			continue
		}

		if cmd.Action == "subscribe" {
			c.hub.Subscribe(c, cmd.LectureID)
			log.Println("INFO: subscribed", cmd.LectureID)
		}

		if cmd.Action == "unsubscribe" {
			c.hub.Unsubscribe(c, cmd.LectureID)
			log.Println("INFO: unsubscribed", cmd.LectureID)
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
		log.Println("INFO: sent", string(msg))
	}
}
