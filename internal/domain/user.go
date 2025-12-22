package domain

import (
	"sync"
)

type User struct {
	ISU        string
	FirstName  string
	LastName   string
	Patronymic *string
}

type UserFaces struct {
	mu                  sync.Mutex
	User                User
	LeftFace            []byte
	RightFace           []byte
	CenterFace          []byte
	LeftFaceEmbedding   []float32
	RightFaceEmbedding  []float32
	CenterFaceEmbedding []float32
}

func (f *UserFaces) GenerateEmbeddings() error {
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	generate := func(vector *[]float32, photo []byte) {
		defer wg.Done()
		// запрос на пайтон сервис за ембеддингом
		f.mu.Lock()
		defer f.mu.Unlock()

		*vector = []float32{1} // затычка
	}

	if f.LeftFace != nil {
		wg.Add(1)
		go generate(&f.LeftFaceEmbedding, f.LeftFace)
	}

	if f.RightFace != nil {
		wg.Add(1)
		go generate(&f.RightFaceEmbedding, f.RightFace)
	}

	if f.CenterFace != nil {
		wg.Add(1)
		go generate(&f.CenterFaceEmbedding, f.CenterFace)
	}

	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}
