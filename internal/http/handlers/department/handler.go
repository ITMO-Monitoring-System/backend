package department

import (
	"context"
	"encoding/json"
	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/response"
	"net/http"

	"github.com/gorilla/mux"
)

type DepartmentService interface {
	GetByID(ctx context.Context, req GetDepartmentByIDRequest) (DepartmentResponse, error)
	GetByCode(ctx context.Context, req GetDepartmentByCodeRequest) (DepartmentResponse, error)
	List(ctx context.Context, req ListDepartmentsRequest) (ListDepartmentsResponse, error)
	Create(ctx context.Context, req CreateDepartmentRequest) (DepartmentResponse, error)
	Delete(ctx context.Context, id int64) error
}

type DepartmentHandler struct {
	service DepartmentService
}

func NewDepartmentHandler(service DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{service: service}
}

// ListDepartments godoc
// @Summary      List departments
// @Description  Возвращает список департаментов с пагинацией.
// @Tags         departments
// @Produce      json
// @Param        limit   query     int  false  "Limit (default 50, max 200)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Success      200  {object}  department.ListDepartmentsResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/departments [get]
func (h *DepartmentHandler) List(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.service.List(r.Context(), ListDepartmentsRequest{Limit: limit, Offset: offset})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// GetDepartmentByID godoc
// @Summary      Get department by ID
// @Tags         departments
// @Produce      json
// @Param        id   path      int  true  "Department ID"
// @Success      200  {object}  department.DepartmentResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/departments/{id} [get]
func (h *DepartmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := httputil.PathInt64(r, "id", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.GetByID(r.Context(), GetDepartmentByIDRequest{ID: id})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// GetDepartmentByCode godoc
// @Summary      Get department by code
// @Tags         departments
// @Produce      json
// @Param        code  path      string  true  "Department code"
// @Success      200  {object}  department.DepartmentResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/departments/code/{code} [get]
func (h *DepartmentHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code, err := httputil.PathString("code", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.GetByCode(r.Context(), GetDepartmentByCodeRequest{Code: code})
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// CreateDepartment godoc
// @Summary      Create department
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        request  body      department.CreateDepartmentRequest  true  "Department payload"
// @Success      201  {object}  department.DepartmentResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/departments [post]
func (h *DepartmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Code == "" || req.Name == "" {
		response.WriteError(w, http.StatusBadRequest, "code and name are required")
		return
	}

	resp, err := h.service.Create(r.Context(), req)
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, resp)
}

// DeleteDepartment godoc
// @Summary      Delete department
// @Tags         departments
// @Param        id  path  int  true  "Department ID"
// @Success      204
// @Failure      400  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /api/departments/{id} [delete]
func (h *DepartmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := httputil.PathInt64(r, "id", vars)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		httputil.WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
