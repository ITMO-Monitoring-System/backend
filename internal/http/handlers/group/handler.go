package group

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/response"
)

type GroupService interface {
	GetByCode(ctx context.Context, req GetGroupByCodeRequest) (GroupResponse, error)
	ListByDepartment(ctx context.Context, req ListGroupsByDepartmentRequest) ([]GroupResponse, error)
	Create(ctx context.Context, req CreateGroupRequest) (GroupResponse, error)
	Delete(ctx context.Context, code string) error
}

type GroupHandler struct {
	service GroupService
}

func NewGroupHandler(service GroupService) *GroupHandler {
	return &GroupHandler{service: service}
}

// GetGroupByCode godoc
// @Summary      Get group by code
// @Tags         groups
// @Produce      json
// @Param        code  path      string  true  "Group code"
// @Success      200  {object}  group.GroupResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/groups/{code} [get]
func (h *GroupHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code, err := httputil.PathString("code", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.GetByCode(r.Context(), GetGroupByCodeRequest{Code: code})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// ListGroupsByDepartment godoc
// @Summary      List groups by department
// @Tags         groups
// @Produce      json
// @Param        department_id  path  int  true  "Department ID"
// @Success      200  {array}   group.GroupResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/departments/{department_id}/groups [get]
func (h *GroupHandler) ListByDepartment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deptID, err := httputil.PathInt64(r, "department_id", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.ListByDepartment(r.Context(), ListGroupsByDepartmentRequest{DepartmentID: deptID})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// CreateGroup godoc
// @Summary      Create group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        request  body      group.CreateGroupRequest  true  "Group payload"
// @Success      201  {object}  group.GroupResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/groups [post]
func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Code == "" || req.DepartmentID == 0 {
		response.WriteError(w, http.StatusBadRequest, "code and department_id are required")
		return
	}

	resp, err := h.service.Create(r.Context(), req)
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, resp)
}

// DeleteGroup godoc
// @Summary      Delete group
// @Tags         groups
// @Param        code  path  string  true  "Group code"
// @Success      204
// @Failure      400  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/groups/{code} [delete]
func (h *GroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code, err := httputil.PathString("code", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.Delete(r.Context(), code); err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
