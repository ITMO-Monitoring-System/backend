package domain

type User struct {
	ISU        string
	FirstName  string
	LastName   string
	Patronymic *string
}

type UserFaces struct {
	User                User
	LeftFace            []byte
	RightFace           []byte
	CenterFace          []byte
	LeftFaceEmbedding   []float64
	RightFaceEmbedding  []float64
	CenterFaceEmbedding []float64
}
