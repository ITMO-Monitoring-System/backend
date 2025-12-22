package ws

import (
	"encoding/json"
	"log"
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
		log.Printf("INFO: recv from conn: %s", msg)

		var cmd struct {
			Action    string `json:"action"`
			LectureID string `json:"lecture_id"`
		}

		lectureID, err := strconv.ParseInt(cmd.LectureID, 10, 64)
		if err != nil {
			log.Printf("WARN: failed to parse lecture id from conn: %s", msg)
			continue
		}

		if json.Unmarshal(msg, &cmd) != nil {
			continue
		}

		if cmd.Action == "subscribe" {
			c.hub.Subscribe(c, lectureID)
			log.Println("INFO: subscribed", cmd.LectureID)
		}

		if cmd.Action == "unsubscribe" {
			c.hub.Unsubscribe(c, lectureID)
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
