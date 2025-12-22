package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/domain"
	lectdto "monitoring_backend/internal/http/handlers/lecture"
	postgres "monitoring_backend/internal/repository/postgres"
)

type LectureService struct {
	db        *pgxpool.Pool
	lectures  postgres.LectureRepository
	lecGroups postgres.LectureGroupRepository
}

func NewLectureService(db *pgxpool.Pool, lectures postgres.LectureRepository, lecGroups postgres.LectureGroupRepository) *LectureService {
	return &LectureService{
		db:        db,
		lectures:  lectures,
		lecGroups: lecGroups,
	}
}

func (s *LectureService) Create(ctx context.Context, req lectdto.CreateLectureRequest) (lectdto.LectureResponse, error) {
	id := req.ID
	if id == 0 {
		var err error
		id, err = nextID(ctx, s.db, "universities_data.lectures_id_seq")
		if err != nil {
			return lectdto.LectureResponse{}, err
		}
	}

	l := domain.Lecture{
		ID:        id,
		Date:      req.Date,
		SubjectID: req.SubjectID,
		TeacherID: req.TeacherID,
	}

	if err := s.lectures.Create(ctx, l); err != nil {
		return lectdto.LectureResponse{}, err
	}

	for _, g := range uniqueStrings(req.GroupIDs) {
		if err := s.lecGroups.AddGroup(ctx, id, g); err != nil {
			return lectdto.LectureResponse{}, err
		}
	}

	return lectdto.LectureResponse{
		ID:        id,
		Date:      req.Date,
		SubjectID: req.SubjectID,
		TeacherID: req.TeacherID,
		GroupIDs:  uniqueStrings(req.GroupIDs),
	}, nil
}

func (s *LectureService) GetByID(ctx context.Context, req lectdto.GetLectureByIDRequest) (lectdto.LectureResponse, error) {
	l, err := s.lectures.GetByID(ctx, req.ID)
	if err != nil {
		return lectdto.LectureResponse{}, err
	}

	groups, err := s.lecGroups.ListGroups(ctx, req.ID)
	if err != nil {
		return lectdto.LectureResponse{}, err
	}

	return lectdto.LectureResponse{
		ID:        l.ID,
		Date:      l.Date,
		SubjectID: l.SubjectID,
		TeacherID: l.TeacherID,
		GroupIDs:  groups,
	}, nil
}

func (s *LectureService) ListByTeacher(ctx context.Context, req lectdto.ListLecturesByTeacherRequest) ([]lectdto.LectureListItemResponse, error) {
	ls, err := s.lectures.ListByTeacher(ctx, req.TeacherID, req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := make([]lectdto.LectureListItemResponse, 0, len(ls))
	for _, l := range ls {
		out = append(out, lectdto.LectureListItemResponse{
			ID: l.ID, Date: l.Date, SubjectID: l.SubjectID, TeacherID: l.TeacherID,
		})
	}
	return out, nil
}

func (s *LectureService) ListBySubject(ctx context.Context, req lectdto.ListLecturesBySubjectRequest) ([]lectdto.LectureListItemResponse, error) {
	ls, err := s.lectures.ListBySubject(ctx, req.SubjectID, req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := make([]lectdto.LectureListItemResponse, 0, len(ls))
	for _, l := range ls {
		out = append(out, lectdto.LectureListItemResponse{
			ID: l.ID, Date: l.Date, SubjectID: l.SubjectID, TeacherID: l.TeacherID,
		})
	}
	return out, nil
}

func (s *LectureService) ListByGroup(ctx context.Context, req lectdto.ListLecturesByGroupRequest) ([]lectdto.LectureListItemResponse, error) {
	ls, err := s.lecGroups.ListLecturesByGroup(ctx, req.GroupCode, req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := make([]lectdto.LectureListItemResponse, 0, len(ls))
	for _, l := range ls {
		out = append(out, lectdto.LectureListItemResponse{
			ID: l.ID, Date: l.Date, SubjectID: l.SubjectID, TeacherID: l.TeacherID,
		})
	}
	return out, nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
