package auth

type LoginRequest struct {
	ISU      string `json:"isu"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

type RegisterRequest struct {
	ISU        string  `json:"isu"`
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
	Patronymic *string `json:"patronymic,omitempty"`
	Password   string  `json:"password"`
	GroupCode  string  `json:"group_code,omitempty"`
}

type MeResponse struct {
	ISU        string  `json:"isu"`
	Role       string  `json:"role"`
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
	Patronymic *string `json:"patronymic,omitempty"`
	Group      string  `json:"group,omitempty"`
	HasPhotos  bool    `json:"has_photos"`
}
