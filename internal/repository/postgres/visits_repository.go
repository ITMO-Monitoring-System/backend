package postgres

import (
	"context"
	"monitoring_backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type lectureVisitsRepository struct {
	db *pgxpool.Pool
}

func NewLectureVisitsRepository(db *pgxpool.Pool) *lectureVisitsRepository {
	return &lectureVisitsRepository{
		db: db,
	}
}

func (v *lectureVisitsRepository) Add(ctx context.Context, visit domain.LectureVisit) (*domain.User, error) {
	return nil, nil
}

func (v *lectureVisitsRepository) Exists(ctx context.Context, lectureID int64, userID string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (v *lectureVisitsRepository) ListByLecture(ctx context.Context, lectureID int64) ([]domain.LectureVisit, error) {
	//TODO implement me
	panic("implement me")
}

func (v *lectureVisitsRepository) ListByUser(ctx context.Context, userID string, from, to time.Time) ([]domain.LectureVisit, error) {
	//TODO implement me
	panic("implement me")
}
