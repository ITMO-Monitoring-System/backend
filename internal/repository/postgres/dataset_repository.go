package postgres

import (
	"context"
	"monitoring_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type datasetRepository struct {
	db *pgxpool.Pool
}

func NewDatasetRepository(db *pgxpool.Pool) *datasetRepository {
	return &datasetRepository{db}
}

func (d datasetRepository) Get(ctx context.Context) ([]domain.UserFaces, error) {
	const selectQuery = `
		SELECT fi.student_id, fi.left_face_embedding, fi.right_face_embedding, fi.full_face_embedding
		FROM cores.face_images fi
	`

	rows, err := d.db.Query(ctx, selectQuery)
	if err != nil {
		return nil, err
	}

	var users []domain.UserFaces

	defer rows.Close()
	for rows.Next() {
		var user domain.UserFaces
		err = rows.Scan(
			&user.User.ISU,
			&user.LeftFaceEmbedding,
			&user.RightFaceEmbedding,
			&user.CenterFaceEmbedding,
		)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
