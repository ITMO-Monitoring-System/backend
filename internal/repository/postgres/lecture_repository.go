package postgres

import (
	"context"
	"monitoring_backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type lectureRepository struct {
	db *pgxpool.Pool
}

// TODO допроверять
func NewLectureRepository(db *pgxpool.Pool) LectureRepository {
	return &lectureRepository{db: db}
}

func (r *lectureRepository) Create(ctx context.Context, l domain.Lecture) error {
	query := `
        INSERT INTO universities_data.lectures (id, date, subject_id, teacher_id)
        VALUES ($1, $2, $3, $4)
    `

	_, err := r.db.Exec(ctx, query, l.ID, l.Date, l.SubjectID, l.TeacherID)
	return err
}

func (r *lectureRepository) GetByID(ctx context.Context, id int64) (domain.Lecture, error) {
	query := `
        SELECT id, date, subject_id, teacher_id
        FROM universities_data.lectures
        WHERE id = $1
    `

	var lecture domain.Lecture
	err := r.db.QueryRow(ctx, query, id).Scan(
		&lecture.ID,
		&lecture.Date,
		&lecture.SubjectID,
		&lecture.TeacherID,
	)

	return lecture, err
}

func (r *lectureRepository) ListByTeacher(ctx context.Context, teacherID string, from, to time.Time) ([]domain.Lecture, error) {
	query := `
        SELECT id, date, subject_id, teacher_id
        FROM universities_data.lectures
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

	var lectures []domain.Lecture
	for rows.Next() {
		var lecture domain.Lecture
		if err := rows.Scan(&lecture.ID, &lecture.Date, &lecture.SubjectID, &lecture.TeacherID); err != nil {
			return nil, err
		}
		lectures = append(lectures, lecture)
	}

	return lectures, rows.Err()
}

func (r *lectureRepository) ListBySubject(ctx context.Context, subjectID int64, from, to time.Time) ([]domain.Lecture, error) {
	query := `
        SELECT id, date, subject_id, teacher_id
        FROM universities_data.lectures
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

	var lectures []domain.Lecture
	for rows.Next() {
		var lecture domain.Lecture
		if err := rows.Scan(&lecture.ID, &lecture.Date, &lecture.SubjectID, &lecture.TeacherID); err != nil {
			return nil, err
		}
		lectures = append(lectures, lecture)
	}

	return lectures, rows.Err()
}
