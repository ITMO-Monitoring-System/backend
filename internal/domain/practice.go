package domain

import "time"

type Practice struct {
	ID        int64
	Date      time.Time
	SubjectID int64
	TeacherID string
}
