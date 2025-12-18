package domain

import "time"

type Lecture struct {
	ID        int64
	Date      time.Time
	SubjectID int64
	TeacherID string
}
