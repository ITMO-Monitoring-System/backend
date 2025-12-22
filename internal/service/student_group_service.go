package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/domain"
	sgdto "monitoring_backend/internal/http/handlers/student_group"
	postgres "monitoring_backend/internal/repository/postgres"
)

type StudentGroupService struct {
	repo postgres.StudentGroupRepository
}

func NewStudentGroupService(db *pgxpool.Pool) *StudentGroupService {
	return &StudentGroupService{repo: postgres.NewStudentGroupRepository(db)}
}

func (s *StudentGroupService) SetUserGroup(ctx context.Context, req sgdto.SetUserGroupRequest) error {
	return s.repo.SetUserGroup(ctx, req.UserID, req.GroupCode)
}

func (s *StudentGroupService) GetUserGroup(ctx context.Context, req sgdto.GetUserGroupRequest) (sgdto.StudentGroupResponse, error) {
	sg, err := s.repo.GetUserGroup(ctx, req.UserID)
	if err != nil {
		return sgdto.StudentGroupResponse{}, err
	}
	return mapStudentGroup(sg), nil
}

func (s *StudentGroupService) RemoveUserGroup(ctx context.Context, req sgdto.RemoveUserGroupRequest) error {
	return s.repo.RemoveUserGroup(ctx, req.UserID)
}

func (s *StudentGroupService) ListUsersByGroup(ctx context.Context, req sgdto.ListUsersByGroupRequest) (sgdto.ListUsersByGroupResponse, error) {
	ids, err := s.repo.ListUsersByGroup(ctx, req.GroupCode)
	if err != nil {
		return sgdto.ListUsersByGroupResponse{}, err
	}
	return sgdto.ListUsersByGroupResponse{UserIDs: ids}, nil
}

func mapStudentGroup(sg domain.StudentGroup) sgdto.StudentGroupResponse {
	return sgdto.StudentGroupResponse{UserID: sg.UserID, GroupCode: sg.GroupCode}
}
