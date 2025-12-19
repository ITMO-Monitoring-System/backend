package postgres

import (
	"context"
	"errors"
	"monitoring_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type departmentRepository struct {
	db *pgxpool.Pool
}

func NewDepartmentRepository(db *pgxpool.Pool) DepartmentRepository {
	return &departmentRepository{db: db}
}

func (r *departmentRepository) GetByID(ctx context.Context, id int64) (domain.Department, error) {
	query := `
        SELECT id, code, name, alias
        FROM universities_data.departments
        WHERE id = $1
    `

	var dept domain.Department
	err := r.db.QueryRow(ctx, query, id).Scan(
		&dept.ID,
		&dept.Code,
		&dept.Name,
		&dept.Alias,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return dept, domain.ErrorDepartmentNotFound
	}

	if err != nil {
		return dept, err
	}

	return dept, nil
}

func (r *departmentRepository) GetByCode(ctx context.Context, code string) (domain.Department, error) {
	query := `
        SELECT id, code, name, alias
        FROM universities_data.departments
        WHERE code = $1
    `

	var dept domain.Department
	err := r.db.QueryRow(ctx, query, code).Scan(
		&dept.ID,
		&dept.Code,
		&dept.Name,
		&dept.Alias,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return dept, domain.ErrorDepartmentNotFound
	}

	if err != nil {
		return dept, err
	}

	return dept, nil
}

func (r *departmentRepository) List(ctx context.Context, limit, offset int) (*domain.Departments, error) {
	query := `
        SELECT id, code, name, alias
        FROM universities_data.departments
        ORDER BY id
        LIMIT $1 OFFSET $2
    `

	rows, err := r.db.Query(ctx, query, limit+1, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var departments []domain.Department
	for rows.Next() {
		var dept domain.Department
		if err := rows.Scan(&dept.ID, &dept.Code, &dept.Name, &dept.Alias); err != nil {
			return nil, err
		}
		departments = append(departments, dept)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrorDepartmentsNotFound
	}

	if rows.Err() != nil {
		return nil, err
	}

	var deps domain.Departments
	if len(departments) > limit {
		deps.HasMore = true
		deps.Departments = departments[:limit]
	} else {
		deps.HasMore = false
		deps.Departments = departments
	}

	return &deps, nil
}
