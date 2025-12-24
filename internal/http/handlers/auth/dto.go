package auth

type LoginRequest struct {
	ISU      string `json:"isu"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}
