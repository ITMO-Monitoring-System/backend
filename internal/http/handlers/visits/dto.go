package visits

type SubjectDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type GetVisitedSubjectsResponse struct {
	ISU      string       `json:"isu"`
	Subjects []SubjectDTO `json:"subjects"`
}
