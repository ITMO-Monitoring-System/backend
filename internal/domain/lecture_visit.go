package domain

import "time"

type LectureVisit struct {
	LectureID int64
	UserID    string
	Date      time.Time
}
