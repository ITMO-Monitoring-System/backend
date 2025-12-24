package postgres

import (
	"context"
	"monitoring_backend/internal/http/handlers/visits"
	"time"

	"monitoring_backend/internal/domain"
)

type LectureVisitRepository interface {
	Add(ctx context.Context, v domain.LectureVisit) (*domain.User, error)
	Exists(ctx context.Context, lectureID int64, userID string) (bool, error)
	ListByLecture(ctx context.Context, lectureID int64) ([]domain.LectureVisit, error)
	ListByUser(ctx context.Context, userID string, from, to time.Time) ([]domain.LectureVisit, error)
	ListVisitedSubjectsByISU(ctx context.Context, isu string) ([]domain.Subject, error)
	ListStudentLecturesBySubject(ctx context.Context, isu string, subjectID int64, filter visits.GetLecturesFilter) ([]visits.LectureAttendance, int, error)
}

type PracticeVisitRepository interface {
	Add(ctx context.Context, v domain.PracticeVisit) error
	Exists(ctx context.Context, practiceID int64, userID string) (bool, error)
	ListByPractice(ctx context.Context, practiceID int64) ([]domain.PracticeVisit, error)
	ListByUser(ctx context.Context, userID string, from, to time.Time) ([]domain.PracticeVisit, error)
}
