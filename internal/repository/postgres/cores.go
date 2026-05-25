package postgres

import (
	"context"
	"monitoring_backend/internal/domain"
	"time"
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
	DeleteFaceImages(ctx context.Context, isu string) error
	GetFaceImage(ctx context.Context, isu string, slot domain.FaceSlot) ([]byte, time.Time, error)
	GetFacesUpdatedAt(ctx context.Context, isu string) (time.Time, error)

	AddRole(ctx context.Context, isu, role string) error
	GetRoles(ctx context.Context, isu string) ([]string, error)
}

type ConsentRepository interface {
	Record(ctx context.Context, c domain.Consent) error
	HasActive(ctx context.Context, isu, consentType string) (bool, error)
	Revoke(ctx context.Context, isu, consentType string) error
	ListByISU(ctx context.Context, isu string) ([]domain.Consent, error)
}

// type FaceImagesRepository interface {
// 	Upsert(ctx context.Context, img domain.FaceImages) error
// 	GetByStudentID(ctx context.Context, studentID string) (domain.FaceImages, error)
// 	Delete(ctx context.Context, studentID string) error
// }
