package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"monitoring_backend/internal/domain"
	"time"
)

type practiceRepository struct {
	db *pgxpool.Pool
}

func NewPracticeRepository(db *pgxpool.Pool) PracticeRepository {
	return &practiceRepository{db: db}
}

func (r *practiceRepository) Create(ctx context.Context, p domain.Practice) error {
	query := `
        INSERT INTO universities_data.practices (id, date, subject_id, teacher_id)
        VALUES ($1, $2, $3, $4)
    `

	_, err := r.db.Exec(ctx, query, p.ID, p.Date, p.SubjectID, p.TeacherID)
	return err
}

func (r *practiceRepository) GetByID(ctx context.Context, id int64) (domain.Practice, error) {
	query := `
        SELECT id, date, subject_id, teacher_id
        FROM universities_data.practices
        WHERE id = $1
    `

	var practice domain.Practice
	err := r.db.QueryRow(ctx, query, id).Scan(
		&practice.ID,
		&practice.Date,
		&practice.SubjectID,
		&practice.TeacherID,
	)

	return practice, err
}

func (r *practiceRepository) ListByTeacher(ctx context.Context, teacherID string, from, to time.Time) ([]domain.Practice, error) {
	query := `
        SELECT id, date, subject_id, teacher_id
        FROM universities_data.practices
        WHERE teacher_id = $1 
          AND date >= $2 
          AND date <= $3
        ORDER BY date
    `

	rows, err := r.db.Query(ctx, query, teacherID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var practices []domain.Practice
	for rows.Next() {
		var practice domain.Practice
		if err := rows.Scan(&practice.ID, &practice.Date, &practice.SubjectID, &practice.TeacherID); err != nil {
			return nil, err
		}
		practices = append(practices, practice)
	}

	return practices, rows.Err()
}

func (r *practiceRepository) ListBySubject(ctx context.Context, subjectID int64, from, to time.Time) ([]domain.Practice, error) {
	query := `
        SELECT id, date, subject_id, teacher_id
        FROM universities_data.practices
        WHERE subject_id = $1 
          AND date >= $2 
          AND date <= $3
        ORDER BY date
    `

	rows, err := r.db.Query(ctx, query, subjectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var practices []domain.Practice
	for rows.Next() {
		var practice domain.Practice
		if err := rows.Scan(&practice.ID, &practice.Date, &practice.SubjectID, &practice.TeacherID); err != nil {
			return nil, err
		}
		practices = append(practices, practice)
	}

	return practices, rows.Err()
}
