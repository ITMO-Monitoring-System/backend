package lecture

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monitoring_backend/internal/domain"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type lookupFunc func(context.Context, int64) (domain.Lecture, error)

func (f lookupFunc) GetByID(ctx context.Context, id int64) (domain.Lecture, error) {
	return f(ctx, id)
}

func TestCheckLectureStatus(t *testing.T) {
	manager := NewManager(nil, "", lookupFunc(func(_ context.Context, id int64) (domain.Lecture, error) {
		switch id {
		case 7:
			return domain.Lecture{ID: id}, nil
		case 8:
			return domain.Lecture{}, pgx.ErrNoRows
		default:
			return domain.Lecture{}, errors.New("database unavailable")
		}
	}))
	router := mux.NewRouter()
	router.HandleFunc("/lecture/{id}/exists", manager.CheckLecture).Methods(http.MethodGet)
	for _, tt := range []struct {
		id   string
		code int
	}{{"7", 204}, {"8", 404}, {"9", 503}, {"0", 400}, {"bad", 400}} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/lecture/"+tt.id+"/exists", nil))
		if w.Code != tt.code {
			t.Errorf("lecture %s: got HTTP %d, want %d", tt.id, w.Code, tt.code)
		}
	}
}

func TestStartLectureRejectsUnknownIDBeforeConsumer(t *testing.T) {
	manager := NewManager(nil, "configured", lookupFunc(func(context.Context, int64) (domain.Lecture, error) {
		return domain.Lecture{}, pgx.ErrNoRows
	}))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/lecture/start",
		strings.NewReader(`{"lecture_id":9999,"queue":"results"}`))
	manager.StartLecture(w, r)
	if w.Code != http.StatusNotFound || len(manager.running) != 0 {
		t.Fatalf("unknown lecture started: HTTP %d, %s", w.Code, w.Body)
	}
}

func TestStoppingEarlierRunKeepsResultsConsumerForNextRun(t *testing.T) {
	manager := NewManager(nil, "configured", funcLookupLecture(7))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.running[7] = resultConsumer{
		queue:  "results",
		runs:   map[string]struct{}{"first": {}},
		cancel: cancel,
	}
	start := httptest.NewRecorder()
	manager.StartLecture(start, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"lecture_id":7,"queue":"results","run_id":"second"}`)))
	if start.Code != http.StatusOK {
		t.Fatalf("start next run: HTTP %d, %s", start.Code, start.Body)
	}
	stop := httptest.NewRecorder()
	manager.StopLecture(stop, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"lecture_id":7,"run_id":"first"}`)))
	if stop.Code != http.StatusOK || ctx.Err() != nil {
		t.Fatalf("first stop canceled the active consumer: HTTP %d, context %v", stop.Code, ctx.Err())
	}
	stop = httptest.NewRecorder()
	manager.StopLecture(stop, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"lecture_id":7,"run_id":"second"}`)))
	if stop.Code != http.StatusOK || ctx.Err() != context.Canceled {
		t.Fatalf("last stop did not cancel the consumer: HTTP %d, context %v", stop.Code, ctx.Err())
	}
}

func funcLookupLecture(id int64) lookupFunc {
	return func(context.Context, int64) (domain.Lecture, error) {
		return domain.Lecture{ID: id}, nil
	}
}
