package department

type GetDepartmentByIDRequest struct {
	ID int64 `validate:"required,gt=0"`
}

type GetDepartmentByCodeRequest struct {
	Code string `validate:"required"`
}

type ListDepartmentsRequest struct {
	Limit  int `validate:"omitempty,gte=1,lte=200"`
	Offset int `validate:"omitempty,gte=0"`
}

type DepartmentResponse struct {
	ID    int64   `json:"id"`
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Alias *string `json:"alias,omitempty"`
}

type ListDepartmentsResponse struct {
	Departments []DepartmentResponse `json:"departments"`
	HasMore     bool                 `json:"has_more"`
}
