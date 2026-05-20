package postgres

import (
	"context"
	"monitoring_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type consentRepository struct {
	db *pgxpool.Pool
}

func NewConsentRepository(db *pgxpool.Pool) *consentRepository {
	return &consentRepository{db: db}
}

// Record добавляет новую запись о согласии в журнал.
func (r *consentRepository) Record(ctx context.Context, c domain.Consent) error {
	const insertQuery = `
		INSERT INTO cores.user_consents (isu, consent_type, doc_version, ip_address, user_agent)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''));
	`
	_, err := r.db.Exec(ctx, insertQuery, c.ISU, c.Type, c.DocVersion, c.IPAddress, c.UserAgent)
	return err
}

// HasActive проверяет, есть ли у пользователя действующее (не отозванное)
// согласие указанного типа.
func (r *consentRepository) HasActive(ctx context.Context, isu, consentType string) (bool, error) {
	var exists bool
	const selectQuery = `
		SELECT EXISTS(
			SELECT 1 FROM cores.user_consents
			WHERE isu = $1 AND consent_type = $2 AND revoked_at IS NULL
		);
	`
	err := r.db.QueryRow(ctx, selectQuery, isu, consentType).Scan(&exists)
	return exists, err
}

// Revoke помечает все действующие согласия указанного типа как отозванные.
func (r *consentRepository) Revoke(ctx context.Context, isu, consentType string) error {
	const updateQuery = `
		UPDATE cores.user_consents
		SET revoked_at = now()
		WHERE isu = $1 AND consent_type = $2 AND revoked_at IS NULL;
	`
	_, err := r.db.Exec(ctx, updateQuery, isu, consentType)
	return err
}

// ListByISU возвращает все записи журнала согласий пользователя.
func (r *consentRepository) ListByISU(ctx context.Context, isu string) ([]domain.Consent, error) {
	const selectQuery = `
		SELECT id, isu, consent_type, doc_version, accepted_at, revoked_at,
		       COALESCE(ip_address, ''), COALESCE(user_agent, '')
		FROM cores.user_consents
		WHERE isu = $1
		ORDER BY accepted_at DESC, id DESC;
	`
	rows, err := r.db.Query(ctx, selectQuery, isu)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []domain.Consent
	for rows.Next() {
		var c domain.Consent
		if err := rows.Scan(
			&c.ID, &c.ISU, &c.Type, &c.DocVersion,
			&c.AcceptedAt, &c.RevokedAt, &c.IPAddress, &c.UserAgent,
		); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return consents, rows.Err()
}
