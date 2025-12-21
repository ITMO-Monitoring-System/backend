package service

import "github.com/jackc/pgx/v5/pgxpool"
import http "monitoring_backend/internal/http/handlers/user"

type UserService struct {
	db *pgxpool.Pool
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

func (s *UserService) AddUser(request http.AddUserRequest) error {
	panic("implement me")
}

func (s *UserService) AddUserFaces(request http.AddUserFacesRequest) error {
	panic("implement me")
}
