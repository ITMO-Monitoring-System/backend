package service

import (
	"context"
	"monitoring_backend/internal/domain"
	consenthttp "monitoring_backend/internal/http/handlers/consent"
	"time"
)

type consentRepository interface {
	Record(ctx context.Context, c domain.Consent) error
	HasActive(ctx context.Context, isu, consentType string) (bool, error)
	Revoke(ctx context.Context, isu, consentType string) error
	ListByISU(ctx context.Context, isu string) ([]domain.Consent, error)
}

// faceDeleter нужен для удаления биометрических данных при отзыве согласия.
type faceDeleter interface {
	DeleteFaceImages(ctx context.Context, isu string) error
}

type ConsentService struct {
	repo  consentRepository
	faces faceDeleter
}

func NewConsentService(repo consentRepository, faces faceDeleter) *ConsentService {
	return &ConsentService{repo: repo, faces: faces}
}

// GiveBiometricConsent фиксирует согласие на обработку биометрии.
// Идемпотентно: если действующее согласие уже есть, повторная запись не создаётся.
func (s *ConsentService) GiveBiometricConsent(ctx context.Context, isu, ip, userAgent string) error {
	active, err := s.repo.HasActive(ctx, isu, domain.ConsentBiometric)
	if err != nil {
		return err
	}
	if active {
		return nil
	}

	return s.repo.Record(ctx, domain.Consent{
		ISU:        isu,
		Type:       domain.ConsentBiometric,
		DocVersion: domain.BiometricConsentVersion,
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// RevokeBiometricConsent отзывает согласие на биометрию и удаляет
// биометрические данные пользователя (ст. 9 152-ФЗ — прекращение обработки).
func (s *ConsentService) RevokeBiometricConsent(ctx context.Context, isu string) error {
	if err := s.repo.Revoke(ctx, isu, domain.ConsentBiometric); err != nil {
		return err
	}
	return s.faces.DeleteFaceImages(ctx, isu)
}

func (s *ConsentService) ListConsents(ctx context.Context, isu string) ([]consenthttp.ConsentItem, error) {
	consents, err := s.repo.ListByISU(ctx, isu)
	if err != nil {
		return nil, err
	}

	items := make([]consenthttp.ConsentItem, 0, len(consents))
	for _, c := range consents {
		item := consenthttp.ConsentItem{
			Type:       c.Type,
			DocVersion: c.DocVersion,
			AcceptedAt: c.AcceptedAt.Format(time.RFC3339),
			Active:     c.RevokedAt == nil,
		}
		if c.RevokedAt != nil {
			revoked := c.RevokedAt.Format(time.RFC3339)
			item.RevokedAt = &revoked
		}
		items = append(items, item)
	}
	return items, nil
}
