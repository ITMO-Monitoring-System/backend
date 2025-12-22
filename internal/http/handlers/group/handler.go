package group

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/response"
)

type GroupService interface {
	GetByCode(ctx context.Context, req GetGroupByCodeRequest) (GroupResponse, error)
	ListByDepartment(ctx context.Context, req ListGroupsByDepartmentRequest) ([]GroupResponse, error)
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
