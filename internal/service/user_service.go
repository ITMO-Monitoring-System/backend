package service

import (
	"context"
	"fmt"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/repository/postgres"
	"monitoring_backend/internal/service/common"
	"strings"

	http "monitoring_backend/internal/http/handlers/user"
)

type userService struct {
	userRepo postgres.UserRepository
}

func NewUserService(userRepo postgres.UserRepository) *userService {
	return &userService{userRepo: userRepo}
}

func (s *userService) AddUser(ctx context.Context, request http.AddUserRequest) error {
	user := domain.User{
		ISU:        request.ISU,
		FirstName:  request.Name,
		LastName:   request.LastName,
		Patronymic: request.Patronymic,
	}

	err := s.userRepo.Create(ctx, &user)
	if err != nil {
		return err
	}

	password, err := common.HashPassword(request.Password)
	if err != nil {
		return err
	}

	err = s.userRepo.SetPassword(ctx, user.ISU, password)
	if err != nil {
		return err
	}

	return nil
}

func (s *userService) AddUserFaces(ctx context.Context, request http.AddUserFacesRequest) error {
	user := domain.UserFaces{
		User: domain.User{
			ISU: request.ISU,
		},
		LeftFace:   request.LeftFacePhoto,
		RightFace:  request.RightFacePhoto,
		CenterFace: request.CenterFacePhoto,
	}

	err := user.GenerateEmbeddings()
	if err != nil {
		return err
	}

	err = s.userRepo.AddFaceEmbeddings(ctx, &user)
	if err != nil {
		return err
	}

	return nil
}

func (s *userService) GetUserRoles(ctx context.Context, isu string) ([]string, error) {
	isu = strings.TrimSpace(isu)
	if isu == "" {
		return nil, fmt.Errorf("isu is empty")
	}

	roles, err := s.userRepo.GetRoles(ctx, isu)
	if err != nil {
		return nil, err
	}

	return roles, nil
}

func (s *userService) AddUserRole(ctx context.Context, request http.AddUserRoleRequest) error {
	isu := strings.TrimSpace(request.ISU)
	role := strings.TrimSpace(request.Role)

	if isu == "" {
		return fmt.Errorf("isu is empty")
	}
	if role == "" {
		return fmt.Errorf("role is empty")
	}

	if err := s.userRepo.AddRole(ctx, isu, role); err != nil {
		return err
	}

	return nil
}
