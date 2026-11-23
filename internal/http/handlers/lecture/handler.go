package lecture

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
)

type LectureService interface {
	Update(ctx context.Context, id int64, req CreateLectureRequest) (LectureResponse, error)
	Create(ctx context.Context, req CreateLectureRequest) (LectureResponse, error)
	GetByID(ctx context.Context, req GetLectureByIDRequest) (LectureResponse, error)
	ListByTeacher(ctx context.Context, req ListLecturesByTeacherRequest) ([]LectureListItemResponse, error)
	ListBySubject(ctx context.Context, req ListLecturesBySubjectRequest) ([]LectureListItemResponse, error)
	ListByGroup(ctx context.Context, req ListLecturesByGroupRequest) ([]LectureListItemResponse, error)
}

type LectureHandler struct {
	service LectureService
}

func NewLectureHandler(service LectureService) *LectureHandler {
	return &LectureHandler{service: service}
}

// CreateLecture godoc
// @Summary      Create lecture
// @Tags         lectures
// @Accept       json
// @Produce      json
// @Param        request  body      lecture.CreateLectureRequest  true  "Lecture payload"
// @Success      201  {object}  lecture.LectureResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/lectures [post]
func (h *LectureHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateLectureRequest
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

// Update godoc
// @Summary Update a lecture owned by the authenticated teacher
// @Tags lectures
// @Accept json
// @Produce json
// @Param id path int true "Lecture ID"
// @Param Authorization header string true "Bearer token"
// @Param request body CreateLectureRequest true "Lecture details (teacher_id comes from the token)"
// @Success 200 {object} LectureResponse
// @Failure 400,401,403,404,409 {object} response.ErrorResponse
// @Router /api/lectures/{id} [put]
func (h *LectureHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.PathInt64(r, "id", mux.Vars(r))
	if err != nil || id <= 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid lecture id")
		return
	}
	actor, _ := middleware.UserID(r.Context())
	role, _ := middleware.Role(r.Context())
	if role != "teacher" || actor == "" {
		response.WriteError(w, http.StatusForbidden, "teacher required")
		return
	}
	var req CreateLectureRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Date.IsZero() || req.SubjectID <= 0 || len(req.GroupIDs) == 0 {
		response.WriteError(w, http.StatusBadRequest, "date, subject_id and group_ids are required")
		return
	}
	req.TeacherID = actor
	item, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, item)
}

// GetLectureByID godoc
// @Summary      Get lecture by ID
// @Tags         lectures
// @Produce      json
// @Param        id  path      int  true  "Lecture ID"
// @Success      200  {object}  lecture.LectureResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/lectures/{id} [get]
func (h *LectureHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := httputil.PathInt64(r, "id", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.GetByID(r.Context(), GetLectureByIDRequest{ID: id})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// ListLecturesByTeacher godoc
// @Summary      List lectures by teacher
// @Tags         lectures
// @Produce      json
// @Param        isu   path   string  true  "Teacher ISU"
// @Param        from  query  string  false  "RFC3339 start time"
// @Param        to    query  string  false  "RFC3339 end time"
// @Success      200  {array}   lecture.LectureListItemResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/teachers/{isu}/lectures [get]
func (h *LectureHandler) ListByTeacher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teacherID, err := httputil.PathString("isu", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	req := ListLecturesByTeacherRequest{TeacherID: teacherID}
	for _, key := range []string{"from", "to"} {
		if !r.URL.Query().Has(key) {
			continue
		}
		value, err := httputil.QueryTimeRFC3339(r, key)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if key == "from" {
			req.From = value
		} else {
			req.To = value
		}
	}
	if !req.To.IsZero() && req.To.Before(req.From) {
		response.WriteError(w, http.StatusBadRequest, "invalid date range")
		return
	}
	resp, err := h.service.ListByTeacher(r.Context(), req)
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// ListLecturesBySubject godoc
// @Summary      List lectures by subject
// @Tags         lectures
// @Produce      json
// @Param        id    path   int     true  "Subject ID"
// @Param        from  query  string  true  "RFC3339 start time"
// @Param        to    query  string  true  "RFC3339 end time"
// @Success      200  {array}   lecture.LectureListItemResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/subjects/{id}/lectures [get]
func (h *LectureHandler) ListBySubject(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.service.ListBySubject(r.Context(), ListLecturesBySubjectRequest{
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

// ListLecturesByGroup godoc
// @Summary      List lectures by group
// @Tags         lectures
// @Produce      json
// @Param        code  path   string  true  "Group code"
// @Param        from  query  string  true  "RFC3339 start time"
// @Param        to    query  string  true  "RFC3339 end time"
// @Success      200  {array}   lecture.LectureListItemResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/groups/{code}/lectures [get]
func (h *LectureHandler) ListByGroup(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.service.ListByGroup(r.Context(), ListLecturesByGroupRequest{
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
