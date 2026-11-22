package service_test

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(ctx, `
      INSERT INTO cores.users VALUES ('catalog-test','Catalog','Test',NULL);
      INSERT INTO universities_data.departments(id,code,name) VALUES (9801,'T1','Catalog');
      INSERT INTO universities_data.groups VALUES ('T1-A',9801),('T1-B',9801);
      INSERT INTO universities_data.subjects VALUES (9801,'Subject');`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, err := db.Exec(ctx, `
       DELETE FROM universities_data.lectures_groups WHERE lecture_id IN (SELECT id FROM universities_data.lectures WHERE teacher_id='catalog-test');
       DELETE FROM universities_data.lectures WHERE teacher_id='catalog-test';
       DELETE FROM universities_data.groups WHERE department_id=9801;
       DELETE FROM universities_data.departments WHERE id=9801;
       DELETE FROM universities_data.subjects WHERE id=9801;
       DELETE FROM cores.users WHERE isu='catalog-test';`)
		if err != nil {
			t.Error(err)
		}
	}()
	svc := service.NewLectureService(db)
	repo := postgres.NewLectureRepository(db)
	now := time.Now().UTC().Truncate(time.Second)
	req := dto.CreateLectureRequest{Date: now, TeacherID: "catalog-test", SubjectID: 9801, GroupIDs: []string{"T1-B", "T1-A"}}
	item, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != "scheduled" || item.SubjectName != "Subject" || !slices.Equal(item.DepartmentCodes, []string{"T1"}) {
		t.Fatal(item)
	}
	if err = repo.SetRunFinished(ctx, item.ID, "r1", false); err != nil {
		t.Fatal(err)
	}
	byGroup, err := svc.ListByGroup(ctx, dto.ListLecturesByGroupRequest{GroupCode: "T1-B", From: now, To: now})
	if err != nil || len(byGroup) != 1 || byGroup[0].Status != "running" || !slices.Equal(byGroup[0].GroupIDs, []string{"T1-A", "T1-B"}) {
		t.Fatalf("group list: %v, %v", byGroup, err)
	}
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
}
