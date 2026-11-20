package ws

import "time"

type UserResponse struct {
	ISU        string  `json:"isu"`
	Name       string  `json:"name"`
	LastName   string  `json:"last_name"`
	Patronymic *string `json:"patronymic"`
}

type UserVisitsLectureResponse struct {
	User       UserResponse `json:"user"`
	LectureID  int64        `json:"lecture_id"`
	RunID      string       `json:"run_id,omitempty"`
	ObservedAt time.Time    `json:"observed_at"`
	Group      *string      `json:"group"`
}
