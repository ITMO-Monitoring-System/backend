package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"monitoring_backend/internal/domain"
	deptdto "monitoring_backend/internal/http/handlers/department"
	userdto "monitoring_backend/internal/http/handlers/user"
	"monitoring_backend/internal/http/handlers/visits"
	"monitoring_backend/internal/repository/postgres"
	"monitoring_backend/internal/service"
)

// Each test uses a new temporary database, never an existing application's schemas.
func testDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	dbName := fmt.Sprintf("fizon_upgrade_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{dbName}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
		admin.Close(ctx)
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(ctx, "DROP DATABASE "+quoted+" WITH (FORCE)"); _ = admin.Close(ctx) })
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = dbName
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	migrations, err := filepath.Glob("../../../migrations/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("no migrations found")
	}
	for _, migration := range migrations {
		sql, err := os.ReadFile(migration)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migration %s: %v", migration, err)
		}
	}
	return pool
}

func execSQL(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}
func ptr[T any](value T) *T { return &value }

func TestCreateUserAtomicAndDepartmentUpdate(t *testing.T) {
	ctx := context.Background()
	pool := testDatabase(t)
	depts := postgres.NewDepartmentRepository(pool)
	d, err := depts.Create(ctx, domain.Department{Code: "A", Name: "Направление", Alias: ptr("Старый")})
	if err != nil {
		t.Fatal(err)
	}
	other, err := depts.Create(ctx, domain.Department{Code: "B", Name: "Направление"})
	if err != nil {
		t.Fatal(err)
	}
	execSQL(t, pool, `INSERT INTO universities_data.groups (code, department_id) VALUES ('P1', $1)`, d.ID)
	repo := postgres.NewUserRepository(pool)
	users := service.NewUserService(repo, nil)
	req := userdto.AddUserRequest{ISU: "100", Name: " Иван ", LastName: "Иванов", Password: "test-password", Roles: []string{"student", "teacher", "student"}, DepartmentID: &d.ID, GroupCode: ptr("P1")}
	if err := users.AddUser(ctx, req); err != nil {
		t.Fatal(err)
	}
	roles, err := repo.GetRoles(ctx, "100")
	if err != nil || len(roles) != 2 {
		t.Fatalf("roles: %v, %v", roles, err)
	}
	hash, err := repo.GetUserPassword(ctx, "100")
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		t.Fatal("password was not saved as a valid hash")
	}
	var group string
	if err := pool.QueryRow(ctx, `SELECT group_code FROM universities_data.students_groups WHERE user_id='100'`).Scan(&group); err != nil || group != "P1" {
		t.Fatalf("group: %q %v", group, err)
	}
	// Fail after the user, password and first role were inserted: all must roll back.
	err = repo.CreateWithAccess(ctx, domain.User{ISU: "rollback", FirstName: "A", LastName: "B", Roles: []string{"student", "student"}, GroupCode: ptr("P1")}, hash, &d.ID)
	if err == nil {
		t.Fatal("expected duplicate role failure")
	}
	for _, table := range []string{"cores.users", "cores.users_passwords", "cores.users_roles"} {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE isu='rollback'").Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial account in %s: %d %v", table, count, err)
		}
	}
	req.ISU, req.DepartmentID = "wrong-dept", &other.ID
	if err := users.AddUser(ctx, req); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected department validation, got %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM cores.users WHERE isu='wrong-dept'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("invalid group selection created a partial account")
	}
	req.ISU, req.DepartmentID, req.GroupCode = "missing-group", &d.ID, ptr("missing")
	if err := users.AddUser(ctx, req); !errors.Is(err, domain.ErrGroupNotFound) {
		t.Fatalf("expected missing group: %v", err)
	}
	req.ISU, req.DepartmentID, req.GroupCode, req.Roles = "legacy", nil, nil, nil
	if err := users.AddUser(ctx, req); err != nil {
		t.Fatalf("legacy request compatibility: %v", err)
	}
	deptService := service.NewDepartmentService(depts)
	updated, err := deptService.Update(ctx, d.ID, deptdto.CreateDepartmentRequest{Code: " NEW ", Name: " Новое название ", Alias: ptr(" ")})
	if err != nil || updated.ID != d.ID || updated.Code != "NEW" || updated.Alias != nil {
		t.Fatalf("update: %+v %v", updated, err)
	}
	var actualID int64
	if err := pool.QueryRow(ctx, `SELECT department_id FROM universities_data.groups WHERE code='P1'`).Scan(&actualID); err != nil || actualID != d.ID {
		t.Fatal("group link was changed")
	}
	if _, err := deptService.Update(ctx, d.ID, deptdto.CreateDepartmentRequest{Code: "B", Name: "Conflict"}); err == nil {
		t.Fatal("expected duplicate code conflict")
	}
	stable, _ := depts.GetByID(ctx, d.ID)
	if stable.Code != "NEW" {
		t.Fatal("failed edit changed original department")
	}
	if _, err := deptService.Update(ctx, 999999, deptdto.CreateDepartmentRequest{Code: "X", Name: "Missing"}); !errors.Is(err, domain.ErrorDepartmentNotFound) {
		t.Fatalf("missing department: %v", err)
	}
}

func TestTeacherReportScopeAndPresence(t *testing.T) {
	ctx := context.Background()
	pool := testDatabase(t)
	execSQL(t, pool, `INSERT INTO cores.users (isu, first_name, last_name) VALUES ('teacher','T','T'), ('other','O','O'), ('full','F','F'), ('single','S','S'), ('absent','A','A');
 INSERT INTO universities_data.departments(id,code,name) VALUES (1,'D','Dept');
 INSERT INTO universities_data.groups(code,department_id) VALUES ('P1',1);
 INSERT INTO universities_data.students_groups(user_id,group_code) VALUES ('full','P1'), ('single','P1'), ('absent','P1');
 INSERT INTO universities_data.subjects(id,name) VALUES (1,'Subject'), (2,'Other');
 INSERT INTO universities_data.lectures(id,date,subject_id,teacher_id) VALUES
 (1,'2026-01-05T10:00:00Z',1,'teacher'), (2,'2026-01-06T10:00:00Z',1,'other'),
 (3,now()+interval '1 day',1,'teacher'), (4,'2026-01-07T10:00:00Z',2,'teacher'), (5,'2026-01-08T10:00:00Z',1,'teacher');
 INSERT INTO universities_data.lectures_groups(lecture_id,group_id) VALUES (1,'P1'), (2,'P1'), (3,'P1'), (4,'P1');
 INSERT INTO visits.lectures_visiting(lecture_id,user_id,date) VALUES
 (1,'full','2026-01-05T10:00:00Z'), (1,'full','2026-01-05T10:01:00Z'),
 (1,'full','2026-01-05T10:05:00Z'), (1,'full','2026-01-05T10:06:00Z'),
 (1,'single','2026-01-05T10:00:00Z'), (2,'full','2026-01-06T10:00:00Z');`)
	repo := postgres.NewLectureVisitsRepository(pool)
	filter := visits.TeacherReportFilter{DateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), DateTo: time.Now().AddDate(0, 0, 2), GapSeconds: 180}
	report, err := repo.GetTeacherReport(ctx, "teacher", 1, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Lectures) != 2 || len(report.Items) != 3 {
		t.Fatalf("unexpected scope: %+v", report)
	}
	byISU := map[string]visits.AttendanceRow{}
	for _, row := range report.Items {
		byISU[row.ISU] = row
	}
	if byISU["full"].PresentSeconds != 120 || byISU["full"].DetectionCount != 4 {
		t.Fatalf("gap calculation: %+v", byISU["full"])
	}
	if byISU["single"].PresentSeconds != 0 || byISU["single"].DetectionCount != 1 {
		t.Fatal("single detection was lost")
	}
	if byISU["absent"].PresentSeconds != 0 || byISU["absent"].DetectionCount != 0 {
		t.Fatal("absence missing from roster")
	}
	filter.GapSeconds = 300
	relaxed, err := repo.GetTeacherReport(ctx, "teacher", 1, filter)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range relaxed.Items {
		if row.ISU == "full" && row.PresentSeconds != 360 {
			t.Fatalf("gap setting ignored: %+v", row)
		}
	}
	empty, err := repo.GetTeacherReport(ctx, "unrelated", 1, filter)
	if err != nil || len(empty.Items) != 0 || len(empty.Lectures) != 0 {
		t.Fatal("another teacher's records are exposed")
	}
	execSQL(t, pool, `SELECT setval('universities_data.lectures_id_seq', 5)`)
	// No 200-lecture pagination truncation. Empty lectures remain in the report.
	execSQL(t, pool, `INSERT INTO universities_data.lectures(date,subject_id,teacher_id) SELECT '2026-02-01T10:00:00Z'::timestamptz + n*interval '1 minute',1,'teacher' FROM generate_series(1,250) n`)
	many, err := repo.GetTeacherReport(ctx, "teacher", 1, filter)
	if err != nil || len(many.Lectures) != 252 {
		t.Fatalf("lecture truncation: %d %v", len(many.Lectures), err)
	}
	// Large reports fail explicitly instead of exporting an incomplete subset.
	execSQL(t, pool, `INSERT INTO cores.users(isu,first_name,last_name)
 SELECT 'bulk-' || n, 'Bulk', 'Student' FROM generate_series(1,200) n;
 INSERT INTO universities_data.students_groups(user_id,group_code)
 SELECT 'bulk-' || n, 'P1' FROM generate_series(1,200) n;
 INSERT INTO universities_data.lectures_groups(lecture_id,group_id)
 SELECT id, 'P1' FROM universities_data.lectures WHERE teacher_id='teacher' AND subject_id=1 ON CONFLICT DO NOTHING;`)
	if _, err := repo.GetTeacherReport(ctx, "teacher", 1, filter); !errors.Is(err, domain.ErrReportTooLarge) {
		t.Fatalf("expected explicit size error, got %v", err)
	}

}
