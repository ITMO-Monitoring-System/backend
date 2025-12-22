package student_group

type SetUserGroupRequest struct {
	UserID    string `json:"user_id" validate:"required"`
	GroupCode string `json:"group_code" validate:"required"`
}

type GetUserGroupRequest struct {
	UserID string `validate:"required"`
}

type RemoveUserGroupRequest struct {
	UserID string `validate:"required"`
}

type ListUsersByGroupRequest struct {
	GroupCode string `validate:"required"`
}

type StudentGroupResponse struct {
	UserID    string `json:"user_id"`
	GroupCode string `json:"group_code"`
}

type ListUsersByGroupResponse struct {
	UserIDs []string `json:"user_ids"`
}
