package service

import (
	"context"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/repository/postgres"
)
import http "monitoring_backend/internal/http/handlers/user"

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

	err := s.userRepo.Create(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func (s *userService) AddUserFaces(ctx context.Context, request http.AddUserFacesRequest) error {
	panic("implement me")
}
