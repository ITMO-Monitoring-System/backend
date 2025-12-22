package subject

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/response"
)

type SubjectService interface {
	Create(ctx context.Context, req CreateSubjectRequest) (SubjectResponse, error)
	GetByID(ctx context.Context, req GetSubjectByIDRequest) (SubjectResponse, error)
	GetByName(ctx context.Context, req GetSubjectByNameRequest) (SubjectResponse, error)
	List(ctx context.Context, req ListSubjectsRequest) ([]SubjectResponse, error)
}

type SubjectHandler struct {
	service SubjectService
}

func NewSubjectHandler(service SubjectService) *SubjectHandler {
	return &SubjectHandler{service: service}
}

// CreateSubject godoc
// @Summary      Create subject
// @Tags         subjects
// @Accept       json
// @Produce      json
// @Param        request  body      subject.CreateSubjectRequest  true  "Subject payload"
// @Success      201  {object}  subject.SubjectResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/subjects [post]
func (h *SubjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		response.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := h.service.Create(r.Context(), req)
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, resp)
}

// GetSubjectByID godoc
// @Summary      Get subject by ID
// @Tags         subjects
// @Produce      json
// @Param        id  path      int  true  "Subject ID"
// @Success      200  {object}  subject.SubjectResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/subjects/{id} [get]
func (h *SubjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := httputil.PathInt64(r, "id", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.GetByID(r.Context(), GetSubjectByIDRequest{ID: id})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// GetSubjectByName godoc
// @Summary      Get subject by name
// @Tags         subjects
// @Produce      json
// @Param        name  path      string  true  "Subject name"
// @Success      200  {object}  subject.SubjectResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/subjects/by-name/{name} [get]
func (h *SubjectHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name, err := httputil.PathString("name", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.GetByName(r.Context(), GetSubjectByNameRequest{Name: name})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// ListSubjects godoc
// @Summary      List subjects
// @Tags         subjects
// @Produce      json
// @Param        limit   query     int  false  "Limit (default 50, max 200)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Success      200  {array}   subject.SubjectResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/subjects [get]
func (h *SubjectHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, err := httputil.QueryInt(r, "limit", 50)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	offset, err := httputil.QueryInt(r, "offset", 0)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.List(r.Context(), ListSubjectsRequest{Limit: limit, Offset: offset})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}
