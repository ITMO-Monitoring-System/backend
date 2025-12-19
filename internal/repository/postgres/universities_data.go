package postgres

import (
	"context"
	"time"

	"monitoring_backend/internal/domain"
)

type DepartmentRepository interface {
	GetByID(ctx context.Context, id int64) (domain.Department, error)
	GetByCode(ctx context.Context, code string) (domain.Department, error)
	List(ctx context.Context, limit, offset int) (*domain.Departments, error)
}

type GroupRepository interface {
	GetByCode(ctx context.Context, code string) (domain.Group, error)
	ListByDepartment(ctx context.Context, departmentID int64) ([]domain.Group, error)
}

type StudentGroupRepository interface {
	SetUserGroup(ctx context.Context, userID, groupCode string) error
	GetUserGroup(ctx context.Context, userID string) (domain.StudentGroup, error)
	RemoveUserGroup(ctx context.Context, userID string) error
	ListUsersByGroup(ctx context.Context, groupCode string) ([]string, error)
}

type SubjectRepository interface {
	Create(ctx context.Context, s domain.Subject) error
	GetByID(ctx context.Context, id int64) (domain.Subject, error)
	GetByName(ctx context.Context, name string) (domain.Subject, error)
	List(ctx context.Context, limit, offset int) ([]domain.Subject, error)
}

type LectureRepository interface {
	Create(ctx context.Context, l domain.Lecture) error
	GetByID(ctx context.Context, id int64) (domain.Lecture, error)
	ListByTeacher(ctx context.Context, teacherID string, from, to time.Time) ([]domain.Lecture, error)
	ListBySubject(ctx context.Context, subjectID int64, from, to time.Time) ([]domain.Lecture, error)
}

type LectureGroupRepository interface {
	AddGroup(ctx context.Context, lectureID int64, groupCode string) error
	RemoveGroup(ctx context.Context, lectureID int64, groupCode string) error
	ListGroups(ctx context.Context, lectureID int64) ([]string, error)
	ListLecturesByGroup(ctx context.Context, groupCode string, from, to time.Time) ([]domain.Lecture, error)
}

type PracticeRepository interface {
	Create(ctx context.Context, p domain.Practice) error
	GetByID(ctx context.Context, id int64) (domain.Practice, error)
	ListByTeacher(ctx context.Context, teacherID string, from, to time.Time) ([]domain.Practice, error)
	ListBySubject(ctx context.Context, subjectID int64, from, to time.Time) ([]domain.Practice, error)
}

type PracticeGroupRepository interface {
	AddGroup(ctx context.Context, practiceID int64, groupCode string) error
	RemoveGroup(ctx context.Context, practiceID int64, groupCode string) error
	ListGroups(ctx context.Context, practiceID int64) ([]string, error)
	ListPracticesByGroup(ctx context.Context, groupCode string, from, to time.Time) ([]domain.Practice, error)
}
