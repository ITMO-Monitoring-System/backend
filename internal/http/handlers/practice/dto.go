package practice

import "time"

type CreatePracticeRequest struct {
	ID        int64     `json:"id,omitempty" validate:"omitempty,gt=0"`
	Date      time.Time `json:"date" validate:"required"`
	SubjectID int64     `json:"subject_id" validate:"required,gt=0"`
	TeacherID string    `json:"teacher_id" validate:"required"`
	GroupIDs  []string  `json:"group_ids" validate:"required,min=1"`
}

type GetPracticeByIDRequest struct {
	ID int64 `validate:"required,gt=0"`
}

type ListPracticesByTeacherRequest struct {
	TeacherID string    `validate:"required"`
	From      time.Time `validate:"required"`
	To        time.Time `validate:"required"`
}

type ListPracticesBySubjectRequest struct {
	SubjectID int64     `validate:"required,gt=0"`
	From      time.Time `validate:"required"`
	To        time.Time `validate:"required"`
}

type ListPracticesByGroupRequest struct {
	GroupCode string    `validate:"required"`
	From      time.Time `validate:"required"`
	To        time.Time `validate:"required"`
}

type PracticeResponse struct {
	ID        int64     `json:"id"`
	Date      time.Time `json:"date"`
	SubjectID int64     `json:"subject_id"`
	TeacherID string    `json:"teacher_id"`
	GroupIDs  []string  `json:"group_ids"`
}

type PracticeListItemResponse struct {
	ID        int64     `json:"id"`
	Date      time.Time `json:"date"`
	SubjectID int64     `json:"subject_id"`
	TeacherID string    `json:"teacher_id"`
}
