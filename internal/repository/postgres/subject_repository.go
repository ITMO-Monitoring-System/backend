package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"monitoring_backend/internal/domain"
)

type subjectRepository struct {
	db *pgxpool.Pool
}

func NewSubjectRepository(db *pgxpool.Pool) SubjectRepository {
	return &subjectRepository{db: db}
}

func (r *subjectRepository) Create(ctx context.Context, s domain.Subject) error {
	query := `
        INSERT INTO universities_data.subjects (id, name)
        VALUES ($1, $2)
    `

	_, err := r.db.Exec(ctx, query, s.ID, s.Name)
	return err
}

func (r *subjectRepository) GetByID(ctx context.Context, id int64) (domain.Subject, error) {
	query := `
        SELECT id, name
        FROM universities_data.subjects
        WHERE id = $1
    `

	var subj domain.Subject
	err := r.db.QueryRow(ctx, query, id).Scan(
		&subj.ID,
		&subj.Name,
	)

	return subj, err
}

func (r *subjectRepository) GetByName(ctx context.Context, name string) (domain.Subject, error) {
	query := `
        SELECT id, name
        FROM universities_data.subjects
        WHERE name = $1
    `

	var subj domain.Subject
	err := r.db.QueryRow(ctx, query, name).Scan(
		&subj.ID,
		&subj.Name,
	)

	return subj, err
}

func (r *subjectRepository) List(ctx context.Context, limit, offset int) ([]domain.Subject, error) {
	query := `
        SELECT id, name
        FROM universities_data.subjects
        ORDER BY id
        LIMIT $1 OFFSET $2
    `

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []domain.Subject
	for rows.Next() {
		var subj domain.Subject
		if err := rows.Scan(&subj.ID, &subj.Name); err != nil {
			return nil, err
		}
		subjects = append(subjects, subj)
	}

	return subjects, rows.Err()
}
