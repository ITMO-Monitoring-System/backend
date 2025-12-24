package lecture

import "time"

type CreateLectureRequest struct {
	Date      time.Time `json:"date" validate:"required"`
	SubjectID int64     `json:"subject_id" validate:"required,gt=0"`
	TeacherID string    `json:"teacher_id" validate:"required"`
	GroupIDs  []string  `json:"group_ids" validate:"required,min=1"`
}

type GetLectureByIDRequest struct {
	ID int64 `validate:"required,gt=0"`
}

type ListLecturesByTeacherRequest struct {
	TeacherID string    `validate:"required"`
	From      time.Time `validate:"required"`
	To        time.Time `validate:"required"`
}

type ListLecturesBySubjectRequest struct {
	SubjectID int64     `validate:"required,gt=0"`
	From      time.Time `validate:"required"`
	To        time.Time `validate:"required"`
}

type ListLecturesByGroupRequest struct {
	GroupCode string    `validate:"required"`
	From      time.Time `validate:"required"`
	To        time.Time `validate:"required"`
}

type LectureResponse struct {
	ID        int64     `json:"id"`
	Date      time.Time `json:"date"`
	SubjectID int64     `json:"subject_id"`
	TeacherID string    `json:"teacher_id"`
	GroupIDs  []string  `json:"group_ids"`
}

type LectureListItemResponse struct {
	ID        int64     `json:"id"`
	Date      time.Time `json:"date"`
	SubjectID int64     `json:"subject_id"`
	TeacherID string    `json:"teacher_id"`
}
