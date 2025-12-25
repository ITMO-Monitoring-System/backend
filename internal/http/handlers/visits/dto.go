package visits

type SubjectDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type GetVisitedSubjectsResponse struct {
	ISU      string       `json:"isu"`
	Subjects []SubjectDTO `json:"subjects"`
}

type PageMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type LectureAttendanceItem struct {
	LectureID      int64  `json:"lecture_id"`
	Date           string `json:"date"` // RFC3339
	TeacherISU     string `json:"teacher_isu"`
	PresentSeconds int64  `json:"present_seconds"`
}

type GetStudentLecturesBySubjectResponse struct {
	SubjectID int64                   `json:"subject_id"`
	ISU       string                  `json:"isu"`
	Items     []LectureAttendanceItem `json:"items"`
	Meta      PageMeta                `json:"meta"`
}

type TeacherLectureItem struct {
	LectureID int64  `json:"lecture_id"`
	Date      string `json:"date"` // RFC3339
}

type GetTeacherLecturesResponse struct {
	SubjectID  int64                `json:"subject_id"`
	TeacherISU string               `json:"teacher_isu"`
	Items      []TeacherLectureItem `json:"items"`
	Meta       PageMeta             `json:"meta"`
}

type GroupItem struct {
	GroupCode string `json:"group_code"`
}

type GetLectureGroupsResponse struct {
	LectureID int64       `json:"lecture_id"`
	Groups    []GroupItem `json:"groups"`
}

type StudentOnLectureItem struct {
	ISU            string  `json:"isu"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	Patronymic     *string `json:"patronymic,omitempty"`
	PresentSeconds int64   `json:"present_seconds"`
}

type GetLectureGroupStudentsResponse struct {
	LectureID int64                  `json:"lecture_id"`
	GroupCode string                 `json:"group_code"`
	Items     []StudentOnLectureItem `json:"items"`
	Meta      PageMeta               `json:"meta"`
}

type GetTeacherSubjectsResponse struct {
	TeacherISU string       `json:"teacher_isu"`
	Subjects   []SubjectDTO `json:"subjects"`
}
