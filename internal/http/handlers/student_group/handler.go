package student_group

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/response"
)

type StudentGroupService interface {
	SetUserGroup(ctx context.Context, req SetUserGroupRequest) error
	GetUserGroup(ctx context.Context, req GetUserGroupRequest) (StudentGroupResponse, error)
	RemoveUserGroup(ctx context.Context, req RemoveUserGroupRequest) error
	ListUsersByGroup(ctx context.Context, req ListUsersByGroupRequest) (ListUsersByGroupResponse, error)
}

type StudentGroupHandler struct {
	service StudentGroupService
}

func NewStudentGroupHandler(service StudentGroupService) *StudentGroupHandler {
	return &StudentGroupHandler{service: service}
}

type setGroupBody struct {
	GroupCode string `json:"group_code"`
}

// SetUserGroup godoc
// @Summary      Set student's group
// @Description  Привязывает студента (ISU) к группе. В твоей схеме у студента может быть только одна группа.
// @Tags         student-groups
// @Accept       json
// @Produce      json
// @Param        isu    path      string  true  "Student ISU"
// @Param        body   body      student_group.SetUserGroupRequest  true  "Group binding payload"
// @Success      200  {string}  string  "ok"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/students/{isu}/group [put]
func (h *StudentGroupHandler) SetUserGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isu, err := httputil.PathString("isu", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body setGroupBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.GroupCode == "" {
		response.WriteError(w, http.StatusBadRequest, "group_code is required")
		return
	}

	if err := h.service.SetUserGroup(r.Context(), SetUserGroupRequest{
		UserID:    isu,
		GroupCode: body.GroupCode,
	}); err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, "ok")
}

// GetUserGroup godoc
// @Summary      Get student's group
// @Tags         student-groups
// @Produce      json
// @Param        isu  path      string  true  "Student ISU"
// @Success      200  {object}  student_group.StudentGroupResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/students/{isu}/group [get]
func (h *StudentGroupHandler) GetUserGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isu, err := httputil.PathString("isu", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.GetUserGroup(r.Context(), GetUserGroupRequest{UserID: isu})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// RemoveUserGroup godoc
// @Summary      Remove student's group binding
// @Tags         student-groups
// @Produce      json
// @Param        isu  path      string  true  "Student ISU"
// @Success      200  {string}  string  "ok"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/students/{isu}/group [delete]
func (h *StudentGroupHandler) RemoveUserGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isu, err := httputil.PathString("isu", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.RemoveUserGroup(r.Context(), RemoveUserGroupRequest{UserID: isu}); err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, "ok")
}

// ListUsersByGroup godoc
// @Summary      List students by group
// @Tags         student-groups
// @Produce      json
// @Param        code  path      string  true  "Group code"
// @Success      200  {object}  student_group.ListUsersByGroupResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/groups/{code}/students [get]
func (h *StudentGroupHandler) ListUsersByGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code, err := httputil.PathString("code", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.ListUsersByGroup(r.Context(), ListUsersByGroupRequest{GroupCode: code})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}
