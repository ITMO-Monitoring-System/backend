package postgres

import (
	"context"
	"errors"
	"monitoring_backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type lectureGroupRepository struct {
	db *pgxpool.Pool
}

func NewLectureGroupRepository(db *pgxpool.Pool) LectureGroupRepository {
	return &lectureGroupRepository{db: db}
}

func (r *lectureGroupRepository) AddGroup(ctx context.Context, lectureID int64, groupCode string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	query := `
        INSERT INTO universities_data.lectures_groups (lecture_id, group_id)
        VALUES ($1, $2)
        ON CONFLICT (lecture_id, group_id) DO NOTHING
    `

	_, err = tx.Exec(ctx, query, lectureID, groupCode)
	return err
}

func (r *lectureGroupRepository) RemoveGroup(ctx context.Context, lectureID int64, groupCode string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	query := `
        DELETE FROM universities_data.lectures_groups 
        WHERE lecture_id = $1 AND group_id = $2
    `

	_, err = tx.Exec(ctx, query, lectureID, groupCode)
	return err
}

func (r *lectureGroupRepository) ListGroups(ctx context.Context, lectureID int64) ([]string, error) {
	query := `
        SELECT group_id
        FROM universities_data.lectures_groups
        WHERE lecture_id = $1
        ORDER BY group_id
    `

	rows, err := r.db.Query(ctx, query, lectureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var groupCode string
		if err := rows.Scan(&groupCode); err != nil {
			return nil, err
		}
		groups = append(groups, groupCode)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrGroupsNotFound
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// TODO как будто тоже лучше с лимитом
func (r *lectureGroupRepository) ListLecturesByGroup(ctx context.Context, groupCode string, from, to time.Time) ([]domain.Lecture, error) {
	query := `
        SELECT l.id, l.date, l.subject_id, l.teacher_id
        FROM universities_data.lectures l
        INNER JOIN universities_data.lectures_groups lg ON l.id = lg.lecture_id
        WHERE lg.group_id = $1 
          AND l.date >= $2 
          AND l.date <= $3
        ORDER BY l.date
    `

	rows, err := r.db.Query(ctx, query, groupCode, from, to)
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
