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
	userRepo    postgres.UserRepository
	consentRepo postgres.ConsentRepository
}

func NewUserService(userRepo postgres.UserRepository, consentRepo postgres.ConsentRepository) *userService {
	return &userService{userRepo: userRepo, consentRepo: consentRepo}
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
	hasConsent, err := s.consentRepo.HasActive(ctx, request.ISU, domain.ConsentBiometric)
	if err != nil {
		return err
	}
	if !hasConsent {
		return domain.ErrBiometricConsentRequired
	}

	user := domain.UserFaces{
		User: domain.User{
			ISU: request.ISU,
		},
		LeftFace:   request.LeftFacePhoto,
		RightFace:  request.RightFacePhoto,
		CenterFace: request.CenterFacePhoto,
	}

	err = user.GenerateEmbeddings()
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

func (s *userService) ListUsers(ctx context.Context, limit, offset int, role string) ([]http.UserResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	users, err := s.userRepo.List(ctx, limit, offset, role)
	if err != nil {
		return nil, err
	}
	out := make([]http.UserResponse, 0, len(users))
	for _, u := range users {
		roles := u.Roles
		if roles == nil {
			roles = []string{}
		}
		out = append(out, http.UserResponse{
			ISU:       u.ISU,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Roles:     roles,
		})
	}
	return out, nil
}

func (s *userService) DeleteUser(ctx context.Context, isu string) error {
	isu = strings.TrimSpace(isu)
	if isu == "" {
		return fmt.Errorf("isu is empty")
	}
	return s.userRepo.Delete(ctx, isu)
}
