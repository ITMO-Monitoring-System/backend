package subject

type CreateSubjectRequest struct {
	ID   int64  `json:"id,omitempty" validate:"omitempty,gt=0"`
	Name string `json:"name" validate:"required"`
}

type GetSubjectByIDRequest struct {
	ID int64 `validate:"required,gt=0"`
}

type GetSubjectByNameRequest struct {
	Name string `validate:"required"`
}

type ListSubjectsRequest struct {
	Limit  int `validate:"omitempty,gte=1,lte=200"`
	Offset int `validate:"omitempty,gte=0"`
}

type SubjectResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
