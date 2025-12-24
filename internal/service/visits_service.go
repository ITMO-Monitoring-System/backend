package service

import (
	"context"
	"fmt"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/handlers/visits"
	"monitoring_backend/internal/repository/postgres"
	"monitoring_backend/internal/ws"
	"strings"
	"time"
)

type visitService struct {
	repo postgres.LectureVisitRepository
}

func NewVisitService(repo postgres.LectureVisitRepository) *visitService {
	return &visitService{repo: repo}
}

func (v *visitService) AddUserVisitsLecture(ctx context.Context, userID string, lectureID int64) (*ws.UserVisitsLectureResponse, error) {
	user, err := v.repo.Add(ctx, domain.LectureVisit{UserID: userID, LectureID: lectureID, Date: time.Now()})
	if err != nil {
		return nil, err
	}

	response := ws.UserVisitsLectureResponse{
		LectureID: lectureID,
		User: ws.UserResponse{
			ISU:        user.ISU,
			Name:       user.FirstName,
			LastName:   user.LastName,
			Patronymic: user.Patronymic,
		},
		Group: user.GroupCode,
	}

	return &response, nil
}

func (s *visitService) GetVisitedSubjectsByISU(ctx context.Context, isu string) ([]domain.Subject, error) {
	isu = strings.TrimSpace(isu)
	if isu == "" {
		return nil, fmt.Errorf("isu is empty")
	}

	return s.repo.ListVisitedSubjectsByISU(ctx, isu)
}

func (s *visitService) GetStudentLecturesBySubject(ctx context.Context, isu string, subjectID int64, filter visits.GetLecturesFilter) ([]visits.LectureAttendance, int, error) {
	isu = strings.TrimSpace(isu)
	if isu == "" {
		return nil, 0, fmt.Errorf("isu is empty")
	}
	if subjectID <= 0 {
		return nil, 0, fmt.Errorf("invalid subject_id")
	}
	return s.repo.ListStudentLecturesBySubject(ctx, isu, subjectID, filter)
}
