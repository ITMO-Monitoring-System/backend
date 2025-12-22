package ws

type UserResponse struct {
	ISU        string  `json:"isu"`
	Name       string  `json:"name"`
	LastName   string  `json:"last_name"`
	Patronymic *string `json:"patronymic"`
}

type UserVisitsLectureResponse struct {
	User      UserResponse `json:"user"`
	LectureID int64        `json:"lecture_id"`
	Group     *string      `json:"group"`
}
