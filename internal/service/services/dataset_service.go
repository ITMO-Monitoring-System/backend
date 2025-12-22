package services

import (
	"context"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/handlers/service/dataset"
)

type DatasetRepository interface {
	Get(ctx context.Context) ([]domain.UserFaces, error)
}

type datasetService struct {
	repo DatasetRepository
}

func NewDatasetService(repo DatasetRepository) *datasetService {
	return &datasetService{repo: repo}
}

func (d *datasetService) Get(ctx context.Context) ([]dataset.StudentResponse, error) {
	faces, err := d.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	var result []dataset.StudentResponse
	for _, face := range faces {
		result = append(result, dataset.StudentResponse{
			UserID:              face.User.ISU,
			LeftFaceEmbedding:   face.LeftFaceEmbedding,
			CenterFaceEmbedding: face.CenterFaceEmbedding,
			RightFaceEmbedding:  face.RightFaceEmbedding,
		})
	}

	return result, nil
}
