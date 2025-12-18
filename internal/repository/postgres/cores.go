package postgres

import (
	"context"
	"monitoring_backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User) error
	GetByISU(ctx context.Context, isu string) (domain.User, error)
	Update(ctx context.Context, u domain.User) error
	Delete(ctx context.Context, isu string) error
}

type FaceImagesRepository interface {
	Upsert(ctx context.Context, img domain.FaceImages) error
	GetByStudentID(ctx context.Context, studentID string) (domain.FaceImages, error)
	Delete(ctx context.Context, studentID string) error
}
