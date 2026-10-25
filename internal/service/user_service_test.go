package service

import (
	"context"
	"errors"
	"monitoring_backend/internal/domain"
	userdto "monitoring_backend/internal/http/handlers/user"
	"strings"
	"testing"
)

func TestInvalidCreateUserDoesNotWrite(t *testing.T) {
	// A nil repository deliberately panics if validation attempts any database write.
	service := NewUserService(nil, nil)
	base := userdto.AddUserRequest{ISU: "100", Name: "A", LastName: "B", Password: "password"}
	for name, mutate := range map[string]func(*userdto.AddUserRequest){
		"missing password":         func(r *userdto.AddUserRequest) { r.Password = " " },
		"oversized password":       func(r *userdto.AddUserRequest) { r.Password = strings.Repeat("я", 37) },
		"unknown role":             func(r *userdto.AddUserRequest) { r.Roles = []string{"root"} },
		"missing name":             func(r *userdto.AddUserRequest) { r.Name = " " },
		"group without student":    func(r *userdto.AddUserRequest) { group := "P1"; r.GroupCode = &group },
		"department without group": func(r *userdto.AddUserRequest) { id := int64(1); r.DepartmentID = &id },
	} {
		t.Run(name, func(t *testing.T) {
			req := base
			mutate(&req)
			if err := service.AddUser(context.Background(), req); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}
