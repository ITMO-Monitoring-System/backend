package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"monitoring_backend/internal/domain"
	"time"
)

type practiceGroupRepository struct {
	db *pgxpool.Pool
}

func NewPracticeGroupRepository(db *pgxpool.Pool) PracticeGroupRepository {
	return &practiceGroupRepository{db: db}
}

func (r *practiceGroupRepository) AddGroup(ctx context.Context, practiceID int64, groupCode string) error {
	query := `
        INSERT INTO universities_data.practices_groups (id, practice_id, group_id)
        VALUES (nextval('practices_groups_id_seq'), $1, $2)
        ON CONFLICT (practice_id, group_id) DO NOTHING
    `

	_, err := r.db.Exec(ctx, query, practiceID, groupCode)
	return err
}

func (r *practiceGroupRepository) RemoveGroup(ctx context.Context, practiceID int64, groupCode string) error {
	query := `
        DELETE FROM universities_data.practices_groups 
        WHERE practice_id = $1 AND group_id = $2
    `

	_, err := r.db.Exec(ctx, query, practiceID, groupCode)
	return err
}

func (r *practiceGroupRepository) ListGroups(ctx context.Context, practiceID int64) ([]string, error) {
	query := `
        SELECT group_id
        FROM universities_data.practices_groups
        WHERE practice_id = $1
        ORDER BY group_id
    `

	rows, err := r.db.Query(ctx, query, practiceID)
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

	return groups, rows.Err()
}

func (r *practiceGroupRepository) ListPracticesByGroup(ctx context.Context, groupCode string, from, to time.Time) ([]domain.Practice, error) {
	query := `
        SELECT p.id, p.date, p.subject_id, p.teacher_id
        FROM universities_data.practices p
        INNER JOIN universities_data.practices_groups pg ON p.id = pg.practice_id
        WHERE pg.group_id = $1 
          AND p.date >= $2 
          AND p.date <= $3
        ORDER BY p.date
    `

	rows, err := r.db.Query(ctx, query, groupCode, from, to)
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
