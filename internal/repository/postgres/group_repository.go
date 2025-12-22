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

func (r *groupRepository) GetByCode(ctx context.Context, code string) (domain.Group, error) {
	query := `
        SELECT code, department_id
        FROM universities_data.groups
        WHERE code = $1
    `

	var group domain.Group
	err := r.db.QueryRow(ctx, query, code).Scan(
		&group.Code,
		&group.DepartmentID,
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
	query := `
        SELECT code, department_id
        FROM universities_data.groups
        WHERE department_id = $1
        ORDER BY code
    `

	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var group domain.Group
		if err := rows.Scan(&group.Code, &group.DepartmentID); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrGroupsNotFound
	}

	return groups, nil
}
