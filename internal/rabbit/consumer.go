package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	amqp "github.com/rabbitmq/amqp091-go"

	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/ws"
)

// StartConsumer читает очередь queue из RabbitMQ и рассылает сообщения всем WS-клиентам lecture_id.
// Реализован reconnect loop: если RabbitMQ временно недоступен — переподключаемся.
func StartConsumer(ctx context.Context, amqpURL string, queue string, lectureID int64, hub *ws.Hub) {
	backoff := 1 * time.Second
	maxBackoff := 20 * time.Second

	for {
		select {
		case <-ctx.Done():
			log.Printf("rabbit consumer context canceled (lecture_id=%d queue=%s)", lectureID, queue)
			return
		default:
		}

		log.Printf("rabbit consumer connecting (lecture_id=%d queue=%s)", lectureID, queue)
		err := consumeOnce(ctx, amqpURL, queue, lectureID, hub)
		if err == nil {
			log.Printf("rabbit consumer stopped (lecture_id=%d queue=%s)", lectureID, queue)
			return
		}
		slog.Warn("consumer_reconnect", "service", "backend", "event", "consumer_reconnect", "lecture_id", lectureID, "error_type", fmt.Sprintf("%T", err))

		select {
		case <-ctx.Done():
			log.Printf("rabbit consumer canceled during backoff (lecture_id=%d queue=%s)", lectureID, queue)
			return
		case <-time.After(backoff):
		}

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
	log.Printf("rabbit consumer started (lecture_id=%d queue=%s)", lectureID, queue)

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
			var result recognitionResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				slog.Warn("recognition_invalid", "lecture_id", lectureID)
				if err := msg.Reject(false); err != nil {
					return err
				}
				continue
			}
			if result.End {
				return msg.Ack(false)
			}
			slog.Info("recognition_received", "service", "backend", "event", "recognition_received",
				"lecture_id", lectureID, "run_id", result.RunID, "request_id", result.RequestID, "person_id", result.PersonID, "bytes", len(msg.Body))
			if err := storeResult(ctx, hub, lectureID, result); err != nil {
				// Closing the channel requeues the unacked delivery. The outer
				// reconnect backoff also limits retries during a database outage.
				return err
			}
			if err := msg.Ack(false); err != nil {
				return err
			}
			slog.Info("recognition_acked", "service", "backend", "event", "recognition_acked",
				"lecture_id", lectureID, "run_id", result.RunID, "request_id", result.RequestID, "person_id", result.PersonID)

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

type recognitionResult struct {
	LectureID  int64     `json:"lecture_id"`
	RunID      string    `json:"run_id"`
	RequestID  string    `json:"request_id"`
	PersonID   string    `json:"person_id"`
	ObservedAt time.Time `json:"observed_at"`
	End        bool      `json:"end"`
}

func storeResult(ctx context.Context, hub *ws.Hub, lectureID int64, result recognitionResult) error {
	if result.PersonID == "" {
		return nil
	}
	var runID pgtype.UUID
	if result.LectureID != lectureID || (result.RunID != "" && (runID.Scan(result.RunID) != nil || !runID.Valid || result.RequestID == "")) {
		slog.Warn("recognition_invalid", "lecture_id", lectureID, "run_id", result.RunID, "request_id", result.RequestID)
		return nil
	}
	if result.ObservedAt.IsZero() {
		result.ObservedAt = time.Now()
	}
	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := hub.RecordObservation(writeCtx, domain.LectureVisit{
		LectureID: lectureID, RunID: result.RunID, UserID: result.PersonID, RequestID: result.RequestID, Date: result.ObservedAt,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		// A deleted lecture or user cannot be recovered by retrying this result.
		slog.Warn("recognition_reference_missing", "lecture_id", lectureID, "request_id", result.RequestID, "constraint", pgErr.ConstraintName)
		return nil
	}
	return err
}
