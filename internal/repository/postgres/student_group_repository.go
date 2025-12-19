package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"monitoring_backend/internal/domain"
)

type studentGroupRepository struct {
	db *pgxpool.Pool
}

func NewStudentGroupRepository(db *pgxpool.Pool) StudentGroupRepository {
	return &studentGroupRepository{db: db}
}

func (r *studentGroupRepository) SetUserGroup(ctx context.Context, userID, groupCode string) error {
	query := `
        INSERT INTO universities_data.students_groups (user_id, group_code)
        VALUES ($1, $2)
        ON CONFLICT (user_id) 
        DO UPDATE SET group_code = EXCLUDED.group_code
    `

	_, err := r.db.Exec(ctx, query, userID, groupCode)
	return err
}

func (r *studentGroupRepository) GetUserGroup(ctx context.Context, userID string) (domain.StudentGroup, error) {
	query := `
        SELECT user_id, group_code
        FROM universities_data.students_groups
        WHERE user_id = $1
    `

	var sg domain.StudentGroup
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&sg.UserID,
		&sg.GroupCode,
	)

	return sg, err
}

func (r *studentGroupRepository) RemoveUserGroup(ctx context.Context, userID string) error {
	query := `DELETE FROM universities_data.students_groups WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *studentGroupRepository) ListUsersByGroup(ctx context.Context, groupCode string) ([]string, error) {
	query := `
        SELECT user_id
        FROM universities_data.students_groups
        WHERE group_code = $1
        ORDER BY user_id
    `

	rows, err := r.db.Query(ctx, query, groupCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}
