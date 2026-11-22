package domain

import "errors"

var ErrSubjectNotFound = errors.New("subject not found")

type Subject struct {
	ID   int64
	Name string
}
