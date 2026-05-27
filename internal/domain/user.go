package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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

var (
	ErrEmbeddingServiceUnavailable = errors.New("embedding service unavailable")
	ErrFaceNotDetected             = errors.New("face not detected in photo")
	ErrFaceImageNotFound           = errors.New("face image not found")
	ErrUserNotFound                = errors.New("user not found")
	ErrUserHasLectures             = errors.New("user is a teacher with existing lectures")
)

type FaceSlot string

const (
	FaceSlotLeft   FaceSlot = "left"
	FaceSlotCenter FaceSlot = "center"
	FaceSlotRight  FaceSlot = "right"
)

func ParseFaceSlot(s string) (FaceSlot, bool) {
	switch FaceSlot(s) {
	case FaceSlotLeft, FaceSlotCenter, FaceSlotRight:
		return FaceSlot(s), true
	}
	return "", false
}

// Column returns the bytea column name for a given slot.
func (s FaceSlot) Column() string {
	switch s {
	case FaceSlotLeft:
		return "left_face"
	case FaceSlotRight:
		return "right_face"
	case FaceSlotCenter:
		return "full_face"
	}
	return ""
}

const defaultEmbeddingServiceURL = "https://projctviscon.vps.webdock.cloud/recognizing/api/embedding"

func embeddingServiceURL() string {
	if raw := strings.TrimSpace(os.Getenv("EMBEDDING_SERVICE_URL")); raw != "" {
		return raw
	}
	return defaultEmbeddingServiceURL
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
		embeddingServiceURL(),
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
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingServiceUnavailable, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("%w: status %d", ErrEmbeddingServiceUnavailable, resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: status %d", ErrFaceNotDetected, resp.StatusCode)
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
		return nil, ErrFaceNotDetected
	}

	return result.Embedding, nil
}
