package domain

import "time"

type LectureVisit struct {
	ID        int64
	LectureID int64
	UserID    string
	Date      time.Time
}
