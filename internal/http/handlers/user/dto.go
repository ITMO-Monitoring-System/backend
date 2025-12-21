package user

type AddUserRequest struct {
	ISU        string  `json:"isu" validate:"required"`
	Name       string  `json:"name" validate:"required"`
	LastName   string  `json:"last_name" validate:"required"`
	Patronymic *string `json:"patronymic"`
}

type AddUserFacesRequest struct {
	ISU             string `json:"isu"`
	LeftFacePhoto   []byte `json:"left_face_photo"`
	RightFacePhoto  []byte `json:"right_face_photo"`
	CenterFacePhoto []byte `json:"center_face_photo"`
}
