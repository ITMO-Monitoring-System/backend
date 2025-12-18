package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"monitoring_backend/internal/domain"
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

	return dept, err
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

	return dept, err
}

func (r *departmentRepository) List(ctx context.Context, limit, offset int) ([]domain.Department, error) {
	query := `
        SELECT id, code, name, alias
        FROM universities_data.departments
        ORDER BY id
        LIMIT $1 OFFSET $2
    `

	rows, err := r.db.Query(ctx, query, limit, offset)
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

	return departments, rows.Err()
}
