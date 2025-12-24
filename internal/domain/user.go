package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type User struct {
	ISU        string
	FirstName  string
	LastName   string
	Patronymic *string
	GroupCode  *string
	Roles      []string
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
		embedding, err := f.requestEmbedding(photo)
		if err != nil {
			log.Printf("ERROR: %v", err)
			errChan <- err
			return
		}
		f.mu.Lock()
		*vector = embedding
		f.mu.Unlock()
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

func (f *UserFaces) requestEmbedding(photo []byte) ([]float32, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		"http://89.111.170.130:8180/api/embedding",
		bytes.NewReader(photo),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{
		Timeout: time.Second * 120,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding service returned %d", resp.StatusCode)
	}

	result := struct {
		Ok        bool      `json:"ok"`
		Embedding []float32 `json:"embedding"`
		BBox      []float64 `json:"bbox"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Ok {
		return nil, errors.New("embedding service returned ok=false")
	}

	return result.Embedding, nil
}
