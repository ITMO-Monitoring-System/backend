package postgres

import (
	"context"
	"monitoring_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *userRepository {
	return &userRepository{
		db: db,
	}
}

func (u *userRepository) Create(ctx context.Context, user domain.User) error {
	//TODO implement me
	panic("implement me")
}

func (u *userRepository) GetByISU(ctx context.Context, isu string) (domain.User, error) {
	//TODO implement me
	panic("implement me")
}

func (u *userRepository) Update(ctx context.Context, user domain.User) error {
	//TODO implement me
	panic("implement me")
}

func (u *userRepository) Delete(ctx context.Context, isu string) error {
	//TODO implement me
	panic("implement me")
}
