package group

type GetGroupByCodeRequest struct {
	Code string `validate:"required"`
}

type ListGroupsByDepartmentRequest struct {
	DepartmentID int64 `validate:"required,gt=0"`
}

type GroupResponse struct {
	Code         string `json:"code"`
	DepartmentID int64  `json:"department_id"`
}
