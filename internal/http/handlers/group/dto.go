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

type CreateGroupRequest struct {
	Code         string `json:"code"`
	DepartmentID int64  `json:"department_id"`
}

type BulkCreateGroupsRequest struct {
	DepartmentID int64    `json:"department_id"`
	Codes        []string `json:"codes"`
}

type BulkGroupSkipped struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

type BulkCreateGroupsResult struct {
	Created []GroupResponse    `json:"created"`
	Skipped []BulkGroupSkipped `json:"skipped"`
}
