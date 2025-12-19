package rabbit

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"monitoring_backend/internal/ws"
)

// StartConsumer читает очередь queue из RabbitMQ и рассылает сообщения всем WS-клиентам lecture_id.
// Реализован reconnect loop: если RabbitMQ временно недоступен — переподключаемся.
func StartConsumer(ctx context.Context, amqpURL string, queue string, lectureID int64, hub *ws.Hub) {
	backoff := 1 * time.Second
	maxBackoff := 20 * time.Second

	for {
		err := consumeOnce(ctx, amqpURL, queue, lectureID, hub)
		log.Printf("rabbit consumer stopped (lecture_id=%d queue=%s): %v", lectureID, queue, err)
		if err == nil {
			log.Printf("rabbit consumer stopped (lecture_id=%d queue=%s)", lectureID, queue)
			return
		}

		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func consumeOnce(ctx context.Context, amqpURL string, queue string, lectureID int64, hub *ws.Hub) error {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// QoS: чтобы не заливать память при ручных ack
	if err := ch.Qos(50, 0, false); err != nil {
		return err
	}

	msgs, err := ch.Consume(
		queue,
		"",    // consumer tag
		false, // autoAck = false (делаем ack сами)
		false, // exclusive
		false, // noLocal (не используется)
		false, // noWait
		nil,
	)
	if err != nil {
		return err
	}

	// Отслеживаем закрытие соединения/канала
	conn_closed := make(chan *amqp.Error, 1)
	ch_closed := make(chan *amqp.Error, 1)
	conn.NotifyClose(conn_closed)
	ch.NotifyClose(ch_closed)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return amqp.ErrClosed
			}

			if isLectureEnd(msg.Body) {
				_ = msg.Ack(false)
				log.Printf("lecture %d is end", lectureID)
				return nil
			}

			// Здесь можно парсить JSON, валидировать, писать в БД.
			// Для минимального рабочего варианта — просто транслируем body как есть.
			hub.Broadcast(lectureID, msg.Body)

			// Ack после успешной передачи в Hub (передача асинхронная, но это достаточный критерий для учебного проекта)
			_ = msg.Ack(false)

		case err := <-conn_closed:
			if err != nil {
				return err
			}
			return amqp.ErrClosed

		case err := <-ch_closed:
			if err != nil {
				return err
			}
			return amqp.ErrClosed
		}
	}
}

func isLectureEnd(body []byte) bool {
	var msg struct {
		End bool `json:"end"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return false
	}
	return msg.End
}
