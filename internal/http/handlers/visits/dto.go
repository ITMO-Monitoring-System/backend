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
