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

// SetRunFinished serializes run transitions with catalog edits on the lecture row.
func (r *lectureRepository) SetRunFinished(ctx context.Context, id int64, run string, finished bool) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var lockedID int64
		if err := tx.QueryRow(ctx, `SELECT id FROM universities_data.lectures WHERE id = $1 FOR UPDATE`, id).Scan(&lockedID); err != nil {
			return err
		}
		if finished {
			// Keep a terminal record even if stop arrives before a delayed start.
			_, err := tx.Exec(ctx, `INSERT INTO universities_data.lecture_runs (lecture_id, run_id, finished)
                VALUES ($1, $2, true)
                ON CONFLICT (lecture_id, run_id) DO UPDATE SET finished = true, interrupted = false`, id, run)
			return err
		}
		var active bool
		err := tx.QueryRow(ctx, `INSERT INTO universities_data.lecture_runs (lecture_id, run_id)
            VALUES ($1, $2)
			ON CONFLICT (lecture_id, run_id) DO UPDATE
			SET finished = CASE WHEN EXCLUDED.run_id = '' OR lecture_runs.interrupted THEN false ELSE lecture_runs.finished END,
			    started_at = CASE WHEN (EXCLUDED.run_id = '' OR lecture_runs.interrupted) AND lecture_runs.finished
			                      THEN clock_timestamp() ELSE lecture_runs.started_at END,
			    interrupted = false
			RETURNING NOT finished`, id, run).Scan(&active)
		if err == nil && !active {
			return domain.ErrLectureRunFinished
		}
		return err
	})
}

// InterruptRun records absence from a healthy recognizing snapshot separately
// from an explicit stop. Tracking may reuse this UUID after recognizing restarts.
func (r *lectureRepository) InterruptRun(ctx context.Context, id int64, run string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var lockedID int64
		if err := tx.QueryRow(ctx, `SELECT id FROM universities_data.lectures WHERE id = $1 FOR UPDATE`, id).Scan(&lockedID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE universities_data.lecture_runs
            SET finished = true, interrupted = true
            WHERE lecture_id = $1 AND run_id = $2 AND NOT finished`, id, run)
		return err
	})
}

func (r *lectureRepository) ListUnfinishedRuns(ctx context.Context) ([]domain.LectureRun, error) {
	rows, err := r.db.Query(ctx, `SELECT lecture_id, run_id, started_at
        FROM universities_data.lecture_runs WHERE NOT finished ORDER BY lecture_id, run_id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LectureRun, error) {
		var run domain.LectureRun
		err := row.Scan(&run.LectureID, &run.RunID, &run.StartedAt)
		return run, err
	})
}
