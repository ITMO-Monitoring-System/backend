package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/domain"
	subjdto "monitoring_backend/internal/http/handlers/subject"
	postgres "monitoring_backend/internal/repository/postgres"
)

type SubjectService struct {
	db   *pgxpool.Pool
	repo postgres.SubjectRepository
}

func NewSubjectService(db *pgxpool.Pool, repo postgres.SubjectRepository) *SubjectService {
	return &SubjectService{
		db:   db,
		repo: repo,
	}
}

func (s *SubjectService) Create(ctx context.Context, req subjdto.CreateSubjectRequest) (subjdto.SubjectResponse, error) {
	id := req.ID
	if id == 0 {
		var err error
		id, err = nextID(ctx, s.db, "universities_data.subjects_id_seq")
		if err != nil {
			return subjdto.SubjectResponse{}, err
		}
	}

	subj := domain.Subject{ID: id, Name: req.Name}
	if err := s.repo.Create(ctx, subj); err != nil {
		return subjdto.SubjectResponse{}, err
	}
	return subjdto.SubjectResponse{ID: id, Name: req.Name}, nil
}

func (s *SubjectService) GetByID(ctx context.Context, req subjdto.GetSubjectByIDRequest) (subjdto.SubjectResponse, error) {
	subj, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		return subjdto.SubjectResponse{}, err
	}
	return mapSubject(subj), nil
}

func (s *SubjectService) GetByName(ctx context.Context, req subjdto.GetSubjectByNameRequest) (subjdto.SubjectResponse, error) {
	subj, err := s.repo.GetByName(ctx, req.Name)
	if err != nil {
		return subjdto.SubjectResponse{}, err
	}
	return mapSubject(subj), nil
}

func (s *SubjectService) List(ctx context.Context, req subjdto.ListSubjectsRequest) ([]subjdto.SubjectResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	subjs, err := s.repo.List(ctx, limit, req.Offset)
	if err != nil {
		return nil, err
	}

	out := make([]subjdto.SubjectResponse, 0, len(subjs))
	for _, x := range subjs {
		out = append(out, mapSubject(x))
	}
	return out, nil
}

func mapSubject(s domain.Subject) subjdto.SubjectResponse {
	return subjdto.SubjectResponse{ID: s.ID, Name: s.Name}
}

func nextID(ctx context.Context, db *pgxpool.Pool, seq string) (int64, error) {
	var id int64
	q := fmt.Sprintf("SELECT nextval('%s')", seq)
	if err := db.QueryRow(ctx, q).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
