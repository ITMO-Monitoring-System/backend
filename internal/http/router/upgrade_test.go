package router_test

import (
	"monitoring_backend/internal/auth"
	"monitoring_backend/internal/http/handlers/department"
	"monitoring_backend/internal/http/handlers/user"
	"monitoring_backend/internal/http/handlers/visits"
	"monitoring_backend/internal/http/router"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUpgradeRoutesRejectUnauthorizedAndMalformedRequests(t *testing.T) {
	jwt := auth.NewJWTManager("test-only", time.Hour)
	r := router.New(router.Dependencies{JWTManager: jwt, Department: department.NewDepartmentHandler(nil), User: user.NewUserHandler(nil), VisitsHandler: visits.NewVisitsHandler(nil)})
	for _, c := range []struct {
		method, path, role, body string
		want                     int
	}{
		{"PUT", "/api/departments/1", "", "{}", 401},
		{"PUT", "/api/departments/1", "student", "{}", 403},
		{"PUT", "/api/departments/1", "teacher", "{}", 403},
		{"PUT", "/api/departments/1", "admin", "{", 400},
		{"POST", "/api/user/admin/create", "teacher", "{}", 403},
		{"POST", "/api/user/admin/create", "admin", "{", 400},
		{"GET", "/api/visits/teacher/1/report", "", "", 401},
		{"GET", "/api/visits/teacher/1/report", "student", "", 403},
		{"GET", "/api/visits/teacher/0/report", "teacher", "", 400},
		{"GET", "/api/visits/teacher/1/report?date_from=invalid", "teacher", "", 400},
	} {
		t.Run(c.method+c.path+c.role, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			if c.role != "" {
				token, err := jwt.Generate("100", c.role)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
			}
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, req)
			if recorder.Code != c.want {
				t.Fatalf("got %d: %s; want %d", recorder.Code, recorder.Body.String(), c.want)
			}
		})
	}
}
