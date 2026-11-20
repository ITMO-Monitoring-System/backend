package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"monitoring_backend/internal/http/handlers/visits"
	"monitoring_backend/internal/rabbit"
	"monitoring_backend/internal/repository/postgres"
	"monitoring_backend/internal/service"
	"monitoring_backend/internal/ws"
)

// Run against an isolated PostgreSQL database with the app migrations applied.
func TestAttendanceThroughRabbitMQ(t *testing.T) {
	dsn, broker := os.Getenv("ITMO_TEST_ATTENDANCE_DSN"), os.Getenv("ITMO_TEST_ATTENDANCE_AMQP")
	if dsn == "" || broker == "" {
		t.Skip("set ITMO_TEST_ATTENDANCE_DSN and ITMO_TEST_ATTENDANCE_AMQP")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, dsn)
	requireNoError(t, err)
	defer db.Close()

	// The event time is deterministic and the replay uses only relative timestamps.
	start := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	_, err = db.Exec(ctx, `
		INSERT INTO cores.users VALUES
			('attendance-teacher', 'Test', 'Teacher', NULL),
			('attendance-student', 'Test', 'Student', NULL),
			('attendance-other', 'Test', 'Single', NULL);
		INSERT INTO universities_data.departments (id, code, name) VALUES (9000, 'AT', 'Attendance Test');
		INSERT INTO universities_data.groups VALUES ('AT01', 9000);
		INSERT INTO universities_data.students_groups VALUES ('attendance-student', 'AT01'), ('attendance-other', 'AT01');
		INSERT INTO universities_data.subjects VALUES (9000, 'Attendance Test');
		INSERT INTO universities_data.lectures VALUES
			(9000, now() - interval '1 hour', 9000, 'attendance-teacher'),
			(9001, now() - interval '1 day', 9000, 'attendance-teacher');
		INSERT INTO universities_data.lectures_groups (lecture_id, group_id) VALUES (9000, 'AT01'), (9001, 'AT01');
		INSERT INTO visits.lectures_visiting (lecture_id, user_id, date) VALUES
			(9001, 'attendance-student', now() - interval '1 day'),
			(9001, 'attendance-student', now() - interval '1 day' + interval '20 seconds');

		CREATE SEQUENCE visits.attendance_test_attempts;
		CREATE FUNCTION visits.attendance_test_fail_once() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.request_id = 'replay-0' THEN
				IF nextval('visits.attendance_test_attempts') = 1 THEN
					RAISE EXCEPTION 'simulated transient error' USING ERRCODE = '40001';
				END IF;
			END IF;
			RETURN NEW;
		END
		$$;
		CREATE TRIGGER attendance_test_fail_once BEFORE INSERT ON visits.lecture_observations
			FOR EACH ROW EXECUTE FUNCTION visits.attendance_test_fail_once();
	`)
	requireNoError(t, err)
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := db.Exec(cleanupCtx, `
			DROP TRIGGER attendance_test_fail_once ON visits.lecture_observations;
			DROP FUNCTION visits.attendance_test_fail_once();
			DROP SEQUENCE visits.attendance_test_attempts;
			DELETE FROM visits.lectures_visiting WHERE lecture_id IN (9000, 9001);
			DELETE FROM universities_data.lectures_groups WHERE lecture_id IN (9000, 9001);
			DELETE FROM universities_data.lectures WHERE id IN (9000, 9001);
			DELETE FROM universities_data.subjects WHERE id = 9000;
			DELETE FROM universities_data.students_groups WHERE group_code = 'AT01';
			DELETE FROM cores.users WHERE isu IN ('attendance-student', 'attendance-other', 'attendance-teacher');
			DELETE FROM universities_data.groups WHERE code = 'AT01';
			DELETE FROM universities_data.departments WHERE id = 9000;
		`)
		if err != nil {
			t.Errorf("clean attendance fixtures: %v", err)
		}
	}()

	amqpConn, err := amqp.Dial(broker)
	requireNoError(t, err)
	defer amqpConn.Close()
	ch, err := amqpConn.Channel()
	requireNoError(t, err)
	defer ch.Close()
	queue, err := ch.QueueDeclare("", false, false, false, false, nil)
	requireNoError(t, err)
	defer ch.QueueDelete(queue.Name, false, false, false)

	repo := postgres.NewLectureVisitsRepository(db)
	hub := ws.NewHub(service.NewVisitService(repo))
	server := httptest.NewServer(ws.Handler(hub))
	defer server.Close()
	socket, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	requireNoError(t, err)
	defer socket.Close()
	updates := make(chan ws.UserVisitsLectureResponse, 100)
	subscribed := make(chan struct{})
	socket.SetPongHandler(func(string) error { close(subscribed); return nil })
	go func() {
		defer close(updates)
		for {
			var update ws.UserVisitsLectureResponse
			if err := socket.ReadJSON(&update); err != nil {
				return
			}
			updates <- update
		}
	}()
	requireNoError(t, socket.WriteJSON(map[string]string{"action": "subscribe", "lecture_id": "9000"}))
	requireNoError(t, socket.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)))
	select {
	case <-subscribed:
	case <-ctx.Done():
		t.Fatal("WebSocket did not acknowledge the ping after subscription")
	}
	consumerCtx, stopConsumer := context.WithCancel(ctx)
	defer stopConsumer()
	go rabbit.StartConsumer(consumerCtx, broker, queue.Name, 9000, hub)

	const (
		run1 = "3527f097-3231-4513-abbf-4abdd9d25a5b"
		run2 = "9de3546b-8e32-46e5-a5ab-76d97d0f89a0"
	)
	publish := func(run, request, person string, offsetMs int) {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"lecture_id": 9000, "run_id": run, "request_id": request, "person_id": person,
			"observed_at": start.Add(time.Duration(offsetMs) * time.Millisecond),
		})
		requireNoError(t, err)
		requireNoError(t, ch.PublishWithContext(ctx, "", queue.Name, false, false, amqp.Publishing{Body: body}))
	}
	offsets := []int{0, 5550, 9480, 14650, 24840, 28330, 30670, 39350, 51090, 52090, 55890, 58790,
		61860, 62030, 62560, 62870, 63130, 64340, 65270, 66430, 66990, 71340, 72560,
		72830, 73720, 74840, 75330, 75640, 76010, 76900, 78860, 82780, 86020, 99640,
		100480, 103400, 105140, 107530, 111370, 113080, 117640, 118410, 118780, 119760,
		120260, 120590, 121880, 122860, 123630}
	for index, offset := range offsets {
		publish(run1, fmt.Sprintf("replay-%d", index), "attendance-student", offset)
	}
	publish(run1, "replay-1", "attendance-student", 125_000)
	publish(run1, "replay-2", "attendance-other", 125_000)
	publish(run2, "replay-1", "attendance-student", 160_000)
	publish(run2, "single", "attendance-other", 180_000)
	publish(run2, "last", "attendance-student", 190_000)

	eventCount := 0
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				t.Fatal("WebSocket closed before the last update")
			}
			eventCount++
			if update.ObservedAt.Equal(start.Add(190 * time.Second)) {
				if update.RunID != run2 || eventCount != len(offsets)+3 {
					t.Fatalf("last update: run=%s, events=%d", update.RunID, eventCount)
				}
				goto saved
			}
		case <-ctx.Done():
			t.Fatalf("timed out after %d WebSocket updates", eventCount)
		}
	}

saved:
	var visitsCount, observations, attempts int
	requireNoError(t, db.QueryRow(ctx, `SELECT count(*) FROM visits.lectures_visiting WHERE lecture_id = 9000`).Scan(&visitsCount))
	requireNoError(t, db.QueryRow(ctx, `SELECT count(*) FROM visits.lecture_observations WHERE run_id IN ($1, $2)`, run1, run2).Scan(&observations))
	requireNoError(t, db.QueryRow(ctx, `SELECT last_value FROM visits.attendance_test_attempts`).Scan(&attempts))
	if visitsCount != 3 || observations != 52 || attempts != 2 {
		t.Fatalf("visits=%d, observations=%d, attempts=%d", visitsCount, observations, attempts)
	}
	items, _, err := repo.ListStudentLecturesBySubject(ctx, "attendance-student", 9000, visits.GetLecturesFilter{
		GapSeconds: 120, PageSize: 10, Page: 1,
	})
	requireNoError(t, err)
	if len(items) != 2 || items[0].PresentSeconds != 154 || items[1].PresentSeconds != 20 {
		t.Fatalf("student history: %+v", items)
	}
	group, _, err := repo.ListLectureGroupStudents(ctx, "attendance-teacher", 9000, "AT01", 1, 10, 120)
	requireNoError(t, err)
	if len(group) != 2 || group[1].ISU != "attendance-student" || group[1].PresentSeconds != 154 {
		t.Fatalf("lecture group history: %+v", group)
	}
	report, err := repo.GetTeacherReport(ctx, "attendance-teacher", 9000, visits.TeacherReportFilter{
		DateFrom: start.Add(-48 * time.Hour), DateTo: start.Add(time.Hour), GapSeconds: 120,
	})
	requireNoError(t, err)
	for _, row := range report.Items {
		switch {
		case row.LectureID == 9000 && row.ISU == "attendance-student" && (row.PresentSeconds != 154 || row.DetectionCount != 51),
			row.LectureID == 9000 && row.ISU == "attendance-other" && (row.PresentSeconds != 0 || row.DetectionCount != 1):
			t.Errorf("teacher report row: %+v", row)
		}
	}
	if len(report.Items) != 4 {
		t.Fatalf("teacher report: %+v", report)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
