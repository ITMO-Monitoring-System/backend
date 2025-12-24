package service

import (
	"context"
	"errors"
	"monitoring_backend/internal/auth"
	"monitoring_backend/internal/domain"
	http "monitoring_backend/internal/http/handlers/auth"
	"monitoring_backend/internal/service/common"
	"strings"

	"slices"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type userRepository interface {
	GetByISU(ctx context.Context, isu string) (*domain.User, error)
	GetUserPassword(ctx context.Context, isu string) (string, error)
}

type AuthService struct {
	repo userRepository
	jwt  *auth.JWTManager
}

func NewAuthService(userRepo userRepository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{
		jwt:  jwt,
		repo: userRepo,
	}
}

func (s *AuthService) Login(ctx context.Context, request http.LoginRequest) (*http.LoginResponse, error) {
	user, err := s.repo.GetByISU(ctx, request.ISU)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	role := strings.ToLower(request.Role)

	if !slices.Contains(user.Roles, role) {
		return nil, ErrInvalidCredentials
	}

	passwordHash, err := s.repo.GetUserPassword(ctx, request.ISU)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	reqPass, err := common.HashPassword(request.Password)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(reqPass),
	)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.jwt.Generate(user.ISU, role)
	if err != nil {
		return nil, err
	}

	response := http.LoginResponse{
		AccessToken: token,
	}

	return &response, nil
}
