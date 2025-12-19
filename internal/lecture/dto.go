package lecture

type StartLectureRequest struct {
	LectureID int64  `json:"lecture_id"`
	Queue     string `json:"queue"`
}

type StopLectureRequest struct {
	LectureID int64 `json:"lecture_id"`
}
