package practice

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/response"
)

type PracticeService interface {
	Create(ctx context.Context, req CreatePracticeRequest) (PracticeResponse, error)
	GetByID(ctx context.Context, req GetPracticeByIDRequest) (PracticeResponse, error)
	ListByTeacher(ctx context.Context, req ListPracticesByTeacherRequest) ([]PracticeListItemResponse, error)
	ListBySubject(ctx context.Context, req ListPracticesBySubjectRequest) ([]PracticeListItemResponse, error)
	ListByGroup(ctx context.Context, req ListPracticesByGroupRequest) ([]PracticeListItemResponse, error)
}

type PracticeHandler struct {
	service PracticeService
}

func NewPracticeHandler(service PracticeService) *PracticeHandler {
	return &PracticeHandler{service: service}
}

// CreatePractice godoc
// @Summary      Create practice
// @Tags         practices
// @Accept       json
// @Produce      json
// @Param        request  body      practice.CreatePracticeRequest  true  "Practice payload"
// @Success      201  {object}  practice.PracticeResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/practices [post]
func (h *PracticeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePracticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TeacherID == "" || req.SubjectID <= 0 || req.Date.IsZero() || len(req.GroupIDs) == 0 {
		response.WriteError(w, http.StatusBadRequest, "teacher_id, subject_id, date, group_ids are required")
		return
	}

	resp, err := h.service.Create(r.Context(), req)
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, resp)
}

// GetPracticeByID godoc
// @Summary      Get practice by ID
// @Tags         practices
// @Produce      json
// @Param        id  path      int  true  "Practice ID"
// @Success      200  {object}  practice.PracticeResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/practices/{id} [get]
func (h *PracticeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := httputil.PathInt64(r, "id", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.GetByID(r.Context(), GetPracticeByIDRequest{ID: id})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// ListPracticesByTeacher godoc
// @Summary      List practices by teacher
// @Tags         practices
// @Produce      json
// @Param        isu   path   string  true  "Teacher ISU"
// @Param        from  query  string  true  "RFC3339 start time"
// @Param        to    query  string  true  "RFC3339 end time"
// @Success      200  {array}   practice.PracticeListItemResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/teachers/{isu}/practices [get]
func (h *PracticeHandler) ListByTeacher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teacherID, err := httputil.PathString("isu", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	from, err := httputil.QueryTimeRFC3339(r, "from")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	to, err := httputil.QueryTimeRFC3339(r, "to")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.ListByTeacher(r.Context(), ListPracticesByTeacherRequest{
		TeacherID: teacherID,
		From:      from,
		To:        to,
	})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// ListPracticesBySubject godoc
// @Summary      List practices by subject
// @Tags         practices
// @Produce      json
// @Param        id    path   int     true  "Subject ID"
// @Param        from  query  string  true  "RFC3339 start time"
// @Param        to    query  string  true  "RFC3339 end time"
// @Success      200  {array}   practice.PracticeListItemResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/subjects/{id}/practices [get]
func (h *PracticeHandler) ListBySubject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subjectID, err := httputil.PathInt64(r, "id", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	from, err := httputil.QueryTimeRFC3339(r, "from")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	to, err := httputil.QueryTimeRFC3339(r, "to")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.ListBySubject(r.Context(), ListPracticesBySubjectRequest{
		SubjectID: subjectID,
		From:      from,
		To:        to,
	})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// ListPracticesByGroup godoc
// @Summary      List practices by group
// @Tags         practices
// @Produce      json
// @Param        code  path   string  true  "Group code"
// @Param        from  query  string  true  "RFC3339 start time"
// @Param        to    query  string  true  "RFC3339 end time"
// @Success      200  {array}   practice.PracticeListItemResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/groups/{code}/practices [get]
func (h *PracticeHandler) ListByGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code, err := httputil.PathString("code", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	from, err := httputil.QueryTimeRFC3339(r, "from")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	to, err := httputil.QueryTimeRFC3339(r, "to")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.ListByGroup(r.Context(), ListPracticesByGroupRequest{
		GroupCode: code,
		From:      from,
		To:        to,
	})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}
