package lecture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/repository/postgres"
)

func TestLectureRecovery(t *testing.T) {
	dsn := os.Getenv("ITMO_TEST_CATALOG_DSN")
	if dsn == "" {
		t.Skip("set ITMO_TEST_CATALOG_DSN to a disposable migrated database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := fmt.Sprintf("lecture_recovery_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	cfg := admin.Config()
	cfg.ConnConfig.Database = name
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+name); err != nil {
			t.Error(err)
		}
	}()
	paths, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil || len(paths) == 0 {
		t.Fatalf("migrations: %v", err)
	}
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, string(contents)); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	// The database is private to this test; recovery checks can advance time safely.
	_, err = db.Exec(ctx, `INSERT INTO cores.users VALUES ('recovery-test','Recovery','Test',NULL);
        INSERT INTO universities_data.subjects VALUES (9851,'Recovery test');
        INSERT INTO universities_data.lectures (id,date,subject_id,teacher_id)
        SELECT id, now(), 9851, 'recovery-test' FROM generate_series(9851,9854) AS id;`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, err := db.Exec(context.Background(), `DELETE FROM universities_data.lectures WHERE teacher_id='recovery-test';
            DELETE FROM universities_data.subjects WHERE id=9851; DELETE FROM cores.users WHERE isu='recovery-test';`)
		if err != nil {
			t.Error(err)
		}
	}()
	repo := postgres.NewLectureRepository(db)
	const oldRun = "00000000-0000-4000-8000-000000000001"
	const liveRun = "00000000-0000-4000-8000-000000000002"
	const missedStart = "00000000-0000-4000-8000-000000000003"
	for _, run := range []string{oldRun, liveRun} {
		if err := repo.SetRunFinished(ctx, 9851, run, false); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.Exec(ctx, `UPDATE universities_data.lecture_runs SET started_at=now()-interval '1 minute' WHERE lecture_id=9851`)
	if err != nil {
		t.Fatal(err)
	}
	manager := NewManager(nil, "configured", repo)
	defer manager.closeConsumers()
	type consumer struct {
		id    int64
		ctx   context.Context
		queue string
	}
	consumers := make(chan consumer, 10)
	manager.consume = func(ctx context.Context, queue string, id int64) {
		consumers <- consumer{id, ctx, queue}
	}
	body := fmt.Sprintf(`{"ok":true,"lectures":[
        {"lecture_id":9851,"running":true,"in_queue":"faces.queue.lecture.9851.%s","out_queue":"results.9851"},
        {"lecture_id":9852,"running":true,"in_queue":"faces.queue.lecture.9852.%s","out_queue":"results.9852"}]}`, liveRun, missedStart)
	statusCode := http.StatusOK
	var responseMu sync.RWMutex
	setResponse := func(code int, value string) {
		responseMu.Lock()
		defer responseMu.Unlock()
		statusCode, body = code, value
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/lecture/status" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		responseMu.RLock()
		defer responseMu.RUnlock()
		w.WriteHeader(statusCode)
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	check := func(when time.Time) error {
		return manager.reconcile(ctx, server.Client(), server.URL+"/api/lecture/status", when)
	}
	isFinished := func(id int64, run string) bool {
		t.Helper()
		var finished bool
		if err := db.QueryRow(ctx, `SELECT finished FROM universities_data.lecture_runs WHERE lecture_id=$1 AND run_id=$2`, id, run).Scan(&finished); err != nil {
			t.Fatal(err)
		}
		return finished
	}
	if err := check(time.Now()); err != nil {
		t.Fatal(err)
	}
	if !isFinished(9851, oldRun) || isFinished(9851, liveRun) || isFinished(9852, missedStart) {
		t.Fatal("stale run was not closed or a live run was lost")
	}
	for range 2 {
		select {
		case c := <-consumers:
			if c.ctx.Err() != nil || c.queue != fmt.Sprintf("results.%d", c.id) {
				t.Fatal(c)
			}
		case <-ctx.Done():
			t.Fatal("live result consumers were not restored")
		}
	}
	if err := check(time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(consumers) != 0 {
		t.Fatal("duplicate consumer after repeated snapshot")
	}
	if err := repo.SetRunFinished(ctx, 9851, liveRun, true); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetRunFinished(ctx, 9851, liveRun, false); !errors.Is(err, domain.ErrLectureRunFinished) {
		t.Fatalf("late start reopened run: %v", err)
	}
	// A stale HTTP snapshot must not resurrect a run stopped after the snapshot.
	if err := check(time.Now()); err != nil {
		t.Fatal(err)
	}
	if !isFinished(9851, liveRun) {
		t.Fatal("stale snapshot reopened completed run")
	}

	if err := repo.SetRunFinished(ctx, 9853, oldRun, false); err != nil {
		t.Fatal(err)
	}
	setResponse(200, `{"ok":true,"lectures":[]}`)
	if err := check(time.Now()); err != nil {
		t.Fatal(err)
	}
	if isFinished(9853, oldRun) {
		t.Fatal("fresh start closed before grace period")
	}
	if err := check(time.Now().Add(runStartGrace + time.Second)); err != nil {
		t.Fatal(err)
	}
	if !isFinished(9853, oldRun) {
		t.Fatal("missed stop was not recovered after grace period")
	}
	// A recognizing restart keeps tracking's binding and UUID. This interrupted
	// run can resume, unlike a run explicitly stopped by the user.
	if err := repo.SetRunFinished(ctx, 9853, oldRun, false); err != nil {
		t.Fatalf("interrupted run could not resume: %v", err)
	}
	if isFinished(9853, oldRun) {
		t.Fatal("resumed run is still finished")
	}
	if err := repo.SetRunFinished(ctx, 9853, oldRun, true); err != nil {
		t.Fatal(err)
	}
	if err := repo.InterruptRun(ctx, 9853, oldRun); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetRunFinished(ctx, 9853, oldRun, false); !errors.Is(err, domain.ErrLectureRunFinished) {
		t.Fatalf("recovery overwrote an explicit stop: %v", err)
	}

	if err := repo.SetRunFinished(ctx, 9854, oldRun, false); err != nil {
		t.Fatal(err)
	}
	for _, response := range []struct {
		code int
		body string
	}{
		{503, `unavailable`}, {200, `{"ok":false,"lectures":[]}`},
		{200, `{"ok":true}`}, {200, `{"ok":true,"lectures":[{"lecture_id":9854}]}`},
		{200, `{`},
	} {
		setResponse(response.code, response.body)
		if err := check(time.Now().Add(time.Minute)); err == nil {
			t.Errorf("invalid response accepted: %s", response.body)
		}
		if isFinished(9854, oldRun) {
			t.Fatal("unavailable or invalid status closed a run")
		}
	}
	// Stop-before-start is terminal too; duplicate notifications are safe.
	if err := repo.SetRunFinished(ctx, 9854, missedStart, true); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetRunFinished(ctx, 9854, missedStart, false); !errors.Is(err, domain.ErrLectureRunFinished) {
		t.Fatal(err)
	}
	var status string
	if err := db.QueryRow(ctx, `SELECT status FROM universities_data.lecture_catalog WHERE id=9851`).Scan(&status); err != nil || status != "completed" {
		t.Fatalf("catalog status %q: %v", status, err)
	}
}

func TestInputQueueRunID(t *testing.T) {
	for queue, want := range map[string]string{
		"faces.queue.lecture.7.00000000-0000-4000-8000-000000000001": "00000000-0000-4000-8000-000000000001",
		"faces.queue.lecture.7": "", "legacy": "", "": "",
	} {
		if got := inputQueueRunID(queue); got != want {
			t.Errorf("%s: got %q, want %q", queue, got, want)
		}
	}
}

func TestInvalidRecognizingSnapshotDoesNotReadDatabase(t *testing.T) {
	manager := NewManager(nil, "", nil)
	for _, body := range []string{`{"ok":false,"lectures":[]}`, `{"ok":true,"lectures":null}`, `{"ok":true,"lectures":[{"lecture_id":7,"running":true}]}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var value any
				_ = json.Unmarshal([]byte(body), &value)
				_ = json.NewEncoder(w).Encode(value)
			}))
			defer server.Close()
			if err := manager.reconcile(context.Background(), server.Client(), server.URL, time.Now()); err == nil {
				t.Fatal("invalid status accepted")
			}
		})
	}
}
