package service

import (
	"context"
	"fmt"
	"strings"

	"monitoring_backend/internal/domain"
	deptdto "monitoring_backend/internal/http/handlers/department"
	postgres "monitoring_backend/internal/repository/postgres"
)

type DepartmentService struct {
	repo postgres.DepartmentRepository
}

func NewDepartmentService(repo postgres.DepartmentRepository) *DepartmentService {
	return &DepartmentService{repo: repo}
}

func (s *DepartmentService) GetByID(ctx context.Context, req deptdto.GetDepartmentByIDRequest) (deptdto.DepartmentResponse, error) {
	dept, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		return deptdto.DepartmentResponse{}, err
	}
	return mapDepartment(dept), nil
}

func (s *DepartmentService) GetByCode(ctx context.Context, req deptdto.GetDepartmentByCodeRequest) (deptdto.DepartmentResponse, error) {
	dept, err := s.repo.GetByCode(ctx, req.Code)
	if err != nil {
		return deptdto.DepartmentResponse{}, err
	}
	return mapDepartment(dept), nil
}

func (s *DepartmentService) List(ctx context.Context, req deptdto.ListDepartmentsRequest) (deptdto.ListDepartmentsResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	deps, err := s.repo.List(ctx, limit, req.Offset)
	if err != nil {
		return deptdto.ListDepartmentsResponse{}, err
	}

	out := deptdto.ListDepartmentsResponse{HasMore: deps.HasMore}
	out.Departments = make([]deptdto.DepartmentResponse, 0, len(deps.Departments))
	for _, d := range deps.Departments {
		out.Departments = append(out.Departments, mapDepartment(d))
	}
	return out, nil
}

func (s *DepartmentService) Create(ctx context.Context, req deptdto.CreateDepartmentRequest) (deptdto.DepartmentResponse, error) {
	d := domain.Department{Code: req.Code, Name: req.Name, Alias: req.Alias}
	created, err := s.repo.Create(ctx, d)
	if err != nil {
		return deptdto.DepartmentResponse{}, err
	}
	return mapDepartment(created), nil
}

func (s *DepartmentService) Update(ctx context.Context, id int64, req deptdto.CreateDepartmentRequest) (deptdto.DepartmentResponse, error) {
	req.Code, req.Name = strings.TrimSpace(req.Code), strings.TrimSpace(req.Name)
	if id <= 0 || req.Code == "" || req.Name == "" || len([]rune(req.Code)) > 125 || len([]rune(req.Name)) > 125 {
		return deptdto.DepartmentResponse{}, fmt.Errorf("%w: Код и название обязательны, не более 125 символов", domain.ErrInvalidInput)
	}
	if req.Alias != nil {
		alias := strings.TrimSpace(*req.Alias)
		if len([]rune(alias)) > 125 {
			return deptdto.DepartmentResponse{}, fmt.Errorf("%w: Псевдоним длиннее 125 символов", domain.ErrInvalidInput)
		}
		req.Alias = &alias
		if alias == "" {
			req.Alias = nil
		}
	}
	d, err := s.repo.Update(ctx, domain.Department{ID: id, Code: req.Code, Name: req.Name, Alias: req.Alias})
	if err != nil {
		return deptdto.DepartmentResponse{}, err
	}
	return mapDepartment(d), nil
}

func (s *DepartmentService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func mapDepartment(d domain.Department) deptdto.DepartmentResponse {
	return deptdto.DepartmentResponse{
		ID:    d.ID,
		Code:  d.Code,
		Name:  d.Name,
		Alias: d.Alias,
	}
}
