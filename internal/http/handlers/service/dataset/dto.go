package dataset

type DatasetResponse struct {
	UsersData []StudentResponse `json:"users_data"`
}

type StudentResponse struct {
	UserID              string    `json:"user_id"`
	LeftFaceEmbedding   []float32 `json:"left_face_embedding"`
	RightFaceEmbedding  []float32 `json:"right_face_embedding"`
	CenterFaceEmbedding []float32 `json:"center_face_embedding"`
}
