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
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	const insertQuery = `
  		INSERT INTO cores.users (isu, first_name, last_name, patronymic) VALUES ($1, $2, $3, $4)
 	`

	_, err = tx.Exec(ctx, insertQuery, user.ISU, user.FirstName, user.LastName, user.Patronymic)
	if err != nil {
		return err
	}

	return nil
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
