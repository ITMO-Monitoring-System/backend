package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/domain"
	groupdto "monitoring_backend/internal/http/handlers/group"
	postgres "monitoring_backend/internal/repository/postgres"
)

type GroupService struct {
	repo postgres.GroupRepository
}

func NewGroupService(db *pgxpool.Pool) *GroupService {
	return &GroupService{repo: postgres.NewGroupRepository(db)}
}

func (s *GroupService) GetByCode(ctx context.Context, req groupdto.GetGroupByCodeRequest) (groupdto.GroupResponse, error) {
	g, err := s.repo.GetByCode(ctx, req.Code)
	if err != nil {
		return groupdto.GroupResponse{}, err
	}
	return mapGroup(g), nil
}

func (s *GroupService) ListByDepartment(ctx context.Context, req groupdto.ListGroupsByDepartmentRequest) ([]groupdto.GroupResponse, error) {
	gs, err := s.repo.ListByDepartment(ctx, req.DepartmentID)
	if err != nil {
		return nil, err
	}
	out := make([]groupdto.GroupResponse, 0, len(gs))
	for _, g := range gs {
		out = append(out, mapGroup(g))
	}
	return out, nil
}

func mapGroup(g domain.Group) groupdto.GroupResponse {
	return groupdto.GroupResponse{
		Code:         g.Code,
		DepartmentID: g.DepartmentID,
	}
}
