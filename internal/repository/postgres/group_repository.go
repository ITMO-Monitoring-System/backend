package postgres

import (
	"context"
	"errors"
	"monitoring_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type groupRepository struct {
	db *pgxpool.Pool
}

func NewGroupRepository(db *pgxpool.Pool) GroupRepository {
	return &groupRepository{db: db}
}

const groupWithStudentCount = `
        SELECT g.code, g.department_id, (
            SELECT count(*) FROM universities_data.students_groups AS sg WHERE sg.group_code = g.code
        )
        FROM universities_data.groups AS g
    `

func (r *groupRepository) GetByCode(ctx context.Context, code string) (domain.Group, error) {
	var group domain.Group
	err := r.db.QueryRow(ctx, groupWithStudentCount+`WHERE g.code = $1`, code).Scan(
		&group.Code,
		&group.DepartmentID,
		&group.StudentCount,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return group, domain.ErrGroupNotFound
	}

	if err != nil {
		return group, err
	}

	return group, nil
}

func (r *groupRepository) ListByDepartment(ctx context.Context, departmentID int64) ([]domain.Group, error) {
	rows, err := r.db.Query(ctx, groupWithStudentCount+`WHERE g.department_id = $1 ORDER BY g.code`, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var group domain.Group
		if err := rows.Scan(&group.Code, &group.DepartmentID, &group.StudentCount); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

func (r *groupRepository) Create(ctx context.Context, g domain.Group) (domain.Group, error) {
	query := `
		INSERT INTO universities_data.groups (code, department_id)
		VALUES ($1, $2)
		RETURNING code, department_id
	`
	var group domain.Group
	err := r.db.QueryRow(ctx, query, g.Code, g.DepartmentID).Scan(
		&group.Code, &group.DepartmentID,
	)
	return group, err
}

func (r *groupRepository) Delete(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM universities_data.groups WHERE code = $1`, code)
	return err
}
