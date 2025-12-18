package domain

import "time"

type PracticeVisit struct {
	ID         int64
	PracticeID int64
	UserID     string
	Date       time.Time
}
