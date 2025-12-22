package service

import (
	"context"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/repository/postgres"
	"monitoring_backend/internal/ws"
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
	}

	return &response, nil
}
