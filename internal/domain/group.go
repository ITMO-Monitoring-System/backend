package domain

import "errors"

var ErrGroupNotFound = errors.New("group not found")
var ErrGroupsNotFound = errors.New("groups not found")

type Group struct {
	Code         string
	DepartmentID int64
}
