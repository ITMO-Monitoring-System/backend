package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/domain"
	prdto "monitoring_backend/internal/http/handlers/practice"
	postgres "monitoring_backend/internal/repository/postgres"
)

type PracticeService struct {
	db         *pgxpool.Pool
	practices  postgres.PracticeRepository
	pracGroups postgres.PracticeGroupRepository
}

func NewPracticeService(db *pgxpool.Pool) *PracticeService {
	return &PracticeService{
		db:         db,
		practices:  postgres.NewPracticeRepository(db),
		pracGroups: postgres.NewPracticeGroupRepository(db),
	}
}

func (s *PracticeService) Create(ctx context.Context, req prdto.CreatePracticeRequest) (prdto.PracticeResponse, error) {
	id := req.ID
	if id == 0 {
		var err error
		id, err = nextID(ctx, s.db, "universities_data.practices_id_seq")
		if err != nil {
			return prdto.PracticeResponse{}, err
		}
	}

	p := domain.Practice{
		ID:        id,
		Date:      req.Date,
		SubjectID: req.SubjectID,
		TeacherID: req.TeacherID,
	}

	if err := s.practices.Create(ctx, p); err != nil {
		return prdto.PracticeResponse{}, err
	}

	for _, g := range uniqueStrings(req.GroupIDs) {
		if err := s.pracGroups.AddGroup(ctx, id, g); err != nil {
			return prdto.PracticeResponse{}, err
		}
	}

	return prdto.PracticeResponse{
		ID:        id,
		Date:      req.Date,
		SubjectID: req.SubjectID,
		TeacherID: req.TeacherID,
		GroupIDs:  uniqueStrings(req.GroupIDs),
	}, nil
}

func (s *PracticeService) GetByID(ctx context.Context, req prdto.GetPracticeByIDRequest) (prdto.PracticeResponse, error) {
	p, err := s.practices.GetByID(ctx, req.ID)
	if err != nil {
		return prdto.PracticeResponse{}, err
	}

	groups, err := s.pracGroups.ListGroups(ctx, req.ID)
	if err != nil {
		return prdto.PracticeResponse{}, err
	}

	return prdto.PracticeResponse{
		ID:        p.ID,
		Date:      p.Date,
		SubjectID: p.SubjectID,
		TeacherID: p.TeacherID,
		GroupIDs:  groups,
	}, nil
}

func (s *PracticeService) ListByTeacher(ctx context.Context, req prdto.ListPracticesByTeacherRequest) ([]prdto.PracticeListItemResponse, error) {
	ps, err := s.practices.ListByTeacher(ctx, req.TeacherID, req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := make([]prdto.PracticeListItemResponse, 0, len(ps))
	for _, p := range ps {
		out = append(out, prdto.PracticeListItemResponse{
			ID: p.ID, Date: p.Date, SubjectID: p.SubjectID, TeacherID: p.TeacherID,
		})
	}
	return out, nil
}

func (s *PracticeService) ListBySubject(ctx context.Context, req prdto.ListPracticesBySubjectRequest) ([]prdto.PracticeListItemResponse, error) {
	ps, err := s.practices.ListBySubject(ctx, req.SubjectID, req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := make([]prdto.PracticeListItemResponse, 0, len(ps))
	for _, p := range ps {
		out = append(out, prdto.PracticeListItemResponse{
			ID: p.ID, Date: p.Date, SubjectID: p.SubjectID, TeacherID: p.TeacherID,
		})
	}
	return out, nil
}

func (s *PracticeService) ListByGroup(ctx context.Context, req prdto.ListPracticesByGroupRequest) ([]prdto.PracticeListItemResponse, error) {
	ps, err := s.pracGroups.ListPracticesByGroup(ctx, req.GroupCode, req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := make([]prdto.PracticeListItemResponse, 0, len(ps))
	for _, p := range ps {
		out = append(out, prdto.PracticeListItemResponse{
			ID: p.ID, Date: p.Date, SubjectID: p.SubjectID, TeacherID: p.TeacherID,
		})
	}
	return out, nil
}
