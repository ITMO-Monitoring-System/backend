package domain

import (
	"errors"
	"time"
)

var ErrLectureRunning = errors.New("нельзя редактировать работающую лекцию")
var ErrLectureRunFinished = errors.New("lecture run is already finished")

type LectureRun struct {
	LectureID int64
	RunID     string
	StartedAt time.Time
}
