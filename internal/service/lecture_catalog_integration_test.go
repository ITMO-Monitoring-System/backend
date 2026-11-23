package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gorilla/mux"
	"monitoring_backend/internal/auth"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/middleware"

	groupdto "monitoring_backend/internal/http/handlers/group"
	dto "monitoring_backend/internal/http/handlers/lecture"
	"monitoring_backend/internal/repository/postgres"
	"monitoring_backend/internal/service"
)

// Use a disposable, migrated database.
func TestLectureCatalog(t *testing.T) {
	dsn := os.Getenv("ITMO_TEST_CATALOG_DSN")
	if dsn == "" {
		t.Skip("set ITMO_TEST_CATALOG_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(ctx, `
      INSERT INTO cores.users VALUES ('catalog-test','Catalog','Test',NULL), ('catalog-student','Catalog','Student',NULL);
      INSERT INTO universities_data.departments(id,code,name) VALUES (9801,'T1','Catalog'), (9802,'T2','Catalog 2');
      INSERT INTO universities_data.groups VALUES ('T1-A',9801), ('T1-B',9801), ('T2-A',9802);
      INSERT INTO universities_data.students_groups VALUES ('catalog-student','T1-A');
      INSERT INTO universities_data.subjects VALUES (9801,'Subject');`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, err := db.Exec(ctx, `
       DELETE FROM universities_data.lectures_groups WHERE lecture_id IN (SELECT id FROM universities_data.lectures WHERE teacher_id='catalog-test');
       DELETE FROM universities_data.lectures WHERE teacher_id='catalog-test';
       DELETE FROM universities_data.students_groups WHERE user_id='catalog-student';
       DELETE FROM universities_data.groups WHERE department_id IN (9801,9802);
       DELETE FROM universities_data.departments WHERE id IN (9801,9802);
       DELETE FROM universities_data.subjects WHERE id=9801;
       DELETE FROM cores.users WHERE isu IN ('catalog-test','catalog-student');`)
		if err != nil {
			t.Error(err)
		}
	}()
	svc := service.NewLectureService(db)
	repo := postgres.NewLectureRepository(db)
	handler := dto.NewLectureHandler(svc)
	router := mux.NewRouter()
	router.HandleFunc("/api/lectures", handler.Create).Methods(http.MethodPost)
	groupHandler := groupdto.NewGroupHandler(service.NewGroupService(postgres.NewGroupRepository(db)))
	router.HandleFunc("/api/departments/{department_id:[0-9]+}/groups", groupHandler.ListByDepartment).Methods(http.MethodGet)
	t.Run("department groups include enrolled student counts", func(t *testing.T) {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/departments/9801/groups", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("got HTTP %d: %s", response.Code, response.Body)
		}
		var groups []groupdto.GroupResponse
		if err := json.Unmarshal(response.Body.Bytes(), &groups); err != nil {
			t.Fatal(err)
		}
		want := []groupdto.GroupResponse{
			{Code: "T1-A", DepartmentID: 9801, StudentCount: 1},
			{Code: "T1-B", DepartmentID: 9801, StudentCount: 0},
		}
		if !slices.Equal(groups, want) {
			t.Fatalf("got groups %+v, want %+v", groups, want)
		}
	})
	now := time.Now().UTC().Truncate(time.Second)
	req := dto.CreateLectureRequest{Date: now, TeacherID: "catalog-test", SubjectID: 9801, GroupIDs: []string{"T2-A", "T1-A"}}
	wantGroupIDs := []string{"T1-A", "T2-A"}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/lectures", bytes.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("creating a lecture across departments: got HTTP %d: %s", response.Code, response.Body)
	}
	var item dto.LectureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.Status != "scheduled" || item.SubjectName != "Subject" ||
		!slices.Equal(item.GroupIDs, wantGroupIDs) || !slices.Equal(item.DepartmentIDs, []int64{9801, 9802}) ||
		!slices.Equal(item.DepartmentCodes, []string{"T1", "T2"}) {
		t.Fatalf("unexpected created lecture: %+v", item)
	}
	if err = repo.SetRunFinished(ctx, item.ID, "r1", false); err != nil {
		t.Fatal(err)
	}
	for _, groupCode := range wantGroupIDs {
		byGroup, err := svc.ListByGroup(ctx, dto.ListLecturesByGroupRequest{GroupCode: groupCode, From: now, To: now})
		if err != nil || len(byGroup) != 1 || byGroup[0].ID != item.ID || byGroup[0].Status != "running" || !slices.Equal(byGroup[0].GroupIDs, wantGroupIDs) {
			t.Fatalf("group %s list: %v, %v", groupCode, byGroup, err)
		}
	}
	t.Run("running lecture update returns HTTP 409 and changes nothing", func(t *testing.T) {
		manager := auth.NewJWTManager("catalog-test", time.Minute)
		token, err := manager.Generate("catalog-test", "teacher")
		if err != nil {
			t.Fatal(err)
		}
		router.Handle("/api/lectures/{id}", middleware.JWT(manager)(http.HandlerFunc(handler.Update))).Methods(http.MethodPut)
		payload := req
		payload.Date = now.Add(2 * time.Hour)
		payload.GroupIDs = []string{"T1-A"}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/lectures/%d", item.ID), bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusConflict {
			t.Fatalf("got HTTP %d: %s", response.Code, response.Body)
		}
		current, err := svc.GetByID(ctx, dto.GetLectureByIDRequest{ID: item.ID})
		if err != nil || !current.Date.Equal(now) || !slices.Equal(current.GroupIDs, wantGroupIDs) {
			t.Fatalf("running lecture changed: %+v, %v", current, err)
		}
	})

	if err = repo.SetRunFinished(ctx, item.ID, "r1", true); err != nil {
		t.Fatal(err)
	}
	req.GroupIDs = []string{"T1-A"}
	req.Date = now.Add(time.Hour)
	if _, err = svc.Update(ctx, item.ID, req); err != nil {
		t.Fatal(err)
	}
	req.TeacherID = "someone-else"
	if _, err = svc.Update(ctx, item.ID, req); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("other teacher update: %v", err)
	}
	req.TeacherID = "catalog-test"
	req.GroupIDs = []string{"no-such-group"}
	if _, err = svc.Update(ctx, item.ID, req); err == nil {
		t.Fatal("invalid group accepted")
	}
	byTeacher, err := svc.ListByTeacher(ctx, dto.ListLecturesByTeacherRequest{TeacherID: req.TeacherID})
	if err != nil || len(byTeacher) != 1 || byTeacher[0].Status != "completed" || !slices.Equal(byTeacher[0].GroupIDs, []string{"T1-A"}) {
		t.Fatalf("teacher list: %v, %v", byTeacher, err)
	}
	bySubject, err := svc.ListBySubject(ctx, dto.ListLecturesBySubjectRequest{SubjectID: 9801, From: now, To: now})
	if err != nil || len(bySubject) != 0 {
		t.Fatalf("old date after edit: %v, %v", bySubject, err)
	}
	t.Run("edit waits for concurrent start and sees its committed state", func(t *testing.T) {
		req.GroupIDs = []string{"T1-A"}
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		var blocker int32
		if err := tx.QueryRow(ctx, `SELECT pg_backend_pid() FROM universities_data.lectures WHERE id=$1 FOR UPDATE`, item.ID).Scan(&blocker); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO universities_data.lecture_runs(lecture_id,run_id) VALUES($1,'concurrent-start')`, item.ID); err != nil {
			t.Fatal(err)
		}
		pending := make(chan error, 1)
		go func() { _, err := svc.Update(ctx, item.ID, req); pending <- err }()
		waitCatalogLock(t, ctx, db, blocker)
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if err := <-pending; !errors.Is(err, domain.ErrLectureRunning) {
			t.Fatalf("concurrent start bypassed edit protection: %v", err)
		}
		if err := repo.SetRunFinished(ctx, item.ID, "concurrent-start", true); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("run registration waits for an edit transaction", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		var blocker int32
		if err := tx.QueryRow(ctx, `SELECT pg_backend_pid() FROM universities_data.lectures WHERE id=$1 FOR UPDATE`, item.ID).Scan(&blocker); err != nil {
			t.Fatal(err)
		}
		pending := make(chan error, 1)
		go func() { pending <- repo.SetRunFinished(ctx, item.ID, "after-edit", false) }()
		waitCatalogLock(t, ctx, db, blocker)
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if err := <-pending; err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Update(ctx, item.ID, req); !errors.Is(err, domain.ErrLectureRunning) {
			t.Fatalf("registered start was not visible: %v", err)
		}
	})

}

func waitCatalogLock(t *testing.T, ctx context.Context, db *pgxpool.Pool, blocker int32) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))`, blocker).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("concurrent operation did not wait for the lecture row lock")
}
