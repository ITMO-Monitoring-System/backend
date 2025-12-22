package service

import (
	"context"

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

func mapDepartment(d domain.Department) deptdto.DepartmentResponse {
	return deptdto.DepartmentResponse{
		ID:    d.ID,
		Code:  d.Code,
		Name:  d.Name,
		Alias: d.Alias,
	}
}
