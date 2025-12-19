package domain

import "errors"

var ErrorDepartmentNotFound = errors.New("department not found")
var ErrorDepartmentsNotFound = errors.New("departments not found")

type Department struct {
	ID    int64
	Code  string
	Name  string
	Alias *string
}

type Departments struct {
	Departments []Department
	HasMore     bool
}
