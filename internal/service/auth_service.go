package service

import (
	"context"
	"errors"
	"fmt"
	"monitoring_backend/internal/auth"
	"monitoring_backend/internal/domain"
	http "monitoring_backend/internal/http/handlers/auth"
	"monitoring_backend/internal/service/common"
	"strings"

	"slices"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

type userRepository interface {
	GetByISU(ctx context.Context, isu string) (*domain.User, error)
	GetUserPassword(ctx context.Context, isu string) (string, error)
	Create(ctx context.Context, user *domain.User) error
	SetPassword(ctx context.Context, isu, password string) error
	AddRole(ctx context.Context, isu, role string) error
	HasFaceImages(ctx context.Context, isu string) (bool, error)
}

type studentGroupRepository interface {
	SetUserGroup(ctx context.Context, userID, groupCode string) error
	GetUserGroup(ctx context.Context, userID string) (domain.StudentGroup, error)
}

type consentRecorder interface {
	Record(ctx context.Context, c domain.Consent) error
}

type AuthService struct {
	repo        userRepository
	sgRepo      studentGroupRepository
	consentRepo consentRecorder
	jwt         *auth.JWTManager
}

func NewAuthService(userRepo userRepository, jwt *auth.JWTManager, sgRepo studentGroupRepository, consentRepo consentRecorder) *AuthService {
	return &AuthService{
		jwt:         jwt,
		repo:        userRepo,
		sgRepo:      sgRepo,
		consentRepo: consentRepo,
	}
}

func (s *AuthService) Register(ctx context.Context, req http.RegisterRequest, ip, userAgent string) error {
	isu := strings.TrimSpace(req.ISU)
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	password := req.Password
	groupCode := strings.TrimSpace(req.GroupCode)

	if isu == "" {
		return fmt.Errorf("ISU обязателен")
	}
	if firstName == "" || lastName == "" {
		return fmt.Errorf("имя и фамилия обязательны")
	}
	if len(password) < 6 {
		return fmt.Errorf("пароль должен быть не менее 6 символов")
	}
	if !req.PDConsentAccepted {
		return fmt.Errorf("необходимо согласие на обработку персональных данных")
	}

	user := &domain.User{
		ISU:        isu,
		FirstName:  firstName,
		LastName:   lastName,
		Patronymic: req.Patronymic,
	}

	err := s.repo.Create(ctx, user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserAlreadyExists
		}
		return err
	}

	hashedPassword, err := common.HashPassword(password)
	if err != nil {
		return err
	}

	if err := s.repo.SetPassword(ctx, isu, hashedPassword); err != nil {
		return err
	}

	if err := s.repo.AddRole(ctx, isu, "student"); err != nil {
		return err
	}

	if err := s.consentRepo.Record(ctx, domain.Consent{
		ISU:        isu,
		Type:       domain.ConsentPersonalData,
		DocVersion: domain.PersonalDataConsentVersion,
		IPAddress:  ip,
		UserAgent:  userAgent,
	}); err != nil {
		return err
	}

	if groupCode != "" && s.sgRepo != nil {
		_ = s.sgRepo.SetUserGroup(ctx, isu, groupCode)
	}

	return nil
}

func (s *AuthService) GetMe(ctx context.Context, isu string) (*http.MeResponse, error) {
	user, err := s.repo.GetByISU(ctx, isu)
	if err != nil {
		return nil, err
	}

	role := ""
	if len(user.Roles) > 0 {
		role = user.Roles[0]
	}

	hasPhotos, _ := s.repo.HasFaceImages(ctx, isu)

	groupCode := ""
	if s.sgRepo != nil {
		sg, err := s.sgRepo.GetUserGroup(ctx, isu)
		if err == nil {
			groupCode = sg.GroupCode
		}
	}

	return &http.MeResponse{
		ISU:        user.ISU,
		Role:       role,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Patronymic: user.Patronymic,
		Group:      groupCode,
		HasPhotos:  hasPhotos,
	}, nil
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

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(request.Password),
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
