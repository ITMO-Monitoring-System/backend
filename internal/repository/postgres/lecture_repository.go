package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/domain"
)

type lectureRepository struct {
	db *pgxpool.Pool
}

// TODO допроверять
func NewLectureRepository(db *pgxpool.Pool) LectureRepository {
	return &lectureRepository{db: db}
}

func (r *lectureRepository) Create(ctx context.Context, l domain.Lecture) (int64, error) {
	query := `
		INSERT INTO universities_data.lectures (date, subject_id, teacher_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRow(ctx, query, l.Date, l.SubjectID, l.TeacherID).Scan(&id)
	return id, err
}

func (r *lectureRepository) GetByID(ctx context.Context, id int64) (domain.Lecture, error) {
	query := `
        SELECT id, date, subject_id, teacher_id
        FROM universities_data.lectures
        WHERE id = $1::bigint
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
        ORDER BY date DESC, id DESC
    `

	rows, err := r.db.Query(ctx, query, teacherID)
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
        ORDER BY date DESC, id DESC
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

func (r *lectureRepository) SetRunFinished(ctx context.Context, id int64, run string, finished bool) error {
	if finished {
		tag, err := r.db.Exec(ctx, `UPDATE universities_data.lecture_runs
            SET finished = true
            WHERE lecture_id = $1 AND run_id = $2`, id, run)
		if err == nil && tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return err
	}
	_, err := r.db.Exec(ctx, `INSERT INTO universities_data.lecture_runs (lecture_id, run_id)
        VALUES ($1, $2)
        ON CONFLICT (lecture_id, run_id) DO UPDATE SET finished = false`, id, run)
	return err
}
