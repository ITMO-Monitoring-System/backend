package postgres

import (
	"context"
	"monitoring_backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByISU(ctx context.Context, isu string) (*domain.User, error)
	List(ctx context.Context, limit, offset int, role string) ([]domain.User, error)
	Update(ctx context.Context, user domain.User) error
	Delete(ctx context.Context, isu string) error
	SetPassword(ctx context.Context, isu, password string) error
	GetUserPassword(ctx context.Context, isu string) (string, error)

	AddFaceEmbeddings(ctx context.Context, userFaces *domain.UserFaces) error
	HasFaceImages(ctx context.Context, isu string) (bool, error)

	AddRole(ctx context.Context, isu, role string) error
	GetRoles(ctx context.Context, isu string) ([]string, error)
}

// type FaceImagesRepository interface {
// 	Upsert(ctx context.Context, img domain.FaceImages) error
// 	GetByStudentID(ctx context.Context, studentID string) (domain.FaceImages, error)
// 	Delete(ctx context.Context, studentID string) error
// }
