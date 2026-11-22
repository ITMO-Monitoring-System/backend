package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"monitoring_backend/internal/domain"
	groupdto "monitoring_backend/internal/http/handlers/group"
	postgres "monitoring_backend/internal/repository/postgres"
)

type GroupService struct {
	repo postgres.GroupRepository
}

func NewGroupService(repo postgres.GroupRepository) *GroupService {
	return &GroupService{repo: repo}
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

func (s *GroupService) Create(ctx context.Context, req groupdto.CreateGroupRequest) (groupdto.GroupResponse, error) {
	g := domain.Group{Code: req.Code, DepartmentID: req.DepartmentID}
	created, err := s.repo.Create(ctx, g)
	if err != nil {
		return groupdto.GroupResponse{}, err
	}
	return mapGroup(created), nil
}

// BulkCreate creates each requested group independently and reports which
// codes were created and which were skipped (and why), instead of failing
// the whole batch for one bad code.
func (s *GroupService) BulkCreate(ctx context.Context, req groupdto.BulkCreateGroupsRequest) (groupdto.BulkCreateGroupsResult, error) {
	result := groupdto.BulkCreateGroupsResult{
		Created: make([]groupdto.GroupResponse, 0, len(req.Codes)),
		Skipped: make([]groupdto.BulkGroupSkipped, 0),
	}

	seen := map[string]bool{}
	for _, raw := range req.Codes {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		if seen[code] {
			result.Skipped = append(result.Skipped, groupdto.BulkGroupSkipped{Code: code, Reason: "дублируется в запросе"})
			continue
		}
		seen[code] = true

		created, err := s.repo.Create(ctx, domain.Group{Code: code, DepartmentID: req.DepartmentID})
		if err != nil {
			result.Skipped = append(result.Skipped, groupdto.BulkGroupSkipped{Code: code, Reason: bulkCreateSkipReason(err)})
			continue
		}
		result.Created = append(result.Created, mapGroup(created))
	}

	return result, nil
}

func bulkCreateSkipReason(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return "группа с таким кодом уже существует"
		case "23503":
			return "направление не найдено"
		}
	}
	return err.Error()
}

func (s *GroupService) Delete(ctx context.Context, code string) error {
	return s.repo.Delete(ctx, code)
}

func mapGroup(g domain.Group) groupdto.GroupResponse {
	return groupdto.GroupResponse{
		Code:         g.Code,
		DepartmentID: g.DepartmentID,
	}
}
