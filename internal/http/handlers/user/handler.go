package user

import (
	"context"
	"encoding/json"
	"io"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"fmt"
)

type UserService interface {
	AddUser(ctx context.Context, request AddUserRequest) error
	AddUserFaces(ctx context.Context, request AddUserFacesRequest) error
	AddUserRole(ctx context.Context, request AddUserRoleRequest) error
	GetUserRoles(ctx context.Context, isu string) ([]string, error)
}

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// AddUser godoc
// @Summary      Добавление нового пользователя
// @Description  Создаёт нового пользователя с ISU, именем, фамилией и факультативным отчеством.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param Authorization header string true "Bearer <JWT>"
// @Param        user  body      AddUserRequest  true  "Пользователь для добавления"
// @Success      201   {string}  string               "ok"
// @Failure      400   {object}  response.ErrorResponse      "Некорректный JSON или обязательные поля отсутствуют"
// @Failure      500   {object}  response.ErrorResponse      "Ошибка сервиса при добавлении пользователя"
// @Security     BearerAuth
// @Router       /api/user/admin/create [post]
func (h *UserHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	role, ok := middleware.Role(r.Context())
	if !ok || role != "admin" {
		response.WriteError(w, http.StatusBadRequest, "Invalid role")
		return
	}

	var request AddUserRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
	}

	if err := h.userService.AddUser(r.Context(), request); err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusCreated, "ok")
}

// UploadFaces godoc
// @Summary      Загрузка фотографий лица студента
// @Description  Загружает три фотографии (левая, правая, фронтальная) студента и сохраняет их в сервисе.
// @Tags         users
// @Accept       multipart/form-data
// @Produce      json
// @Param        isu          path      string                 true  "ISU студента"
// @Param        left_face    formData  file                   true  "Фотография левой стороны лица"
// @Param        right_face   formData  file                   true  "Фотография правой стороны лица"
// @Param        center_face  formData  file                   true  "Фотография фронтальной стороны лица"
// @Success      200          {string}  string                 "ok"
// @Failure      400          {object}  response.ErrorResponse        "Некорректный ISU или отсутствуют файлы"
// @Failure      500          {object}  response.ErrorResponse        "Ошибка сервиса при добавлении фотографий"
// @Router       /api/user/upload/faces/{isu} [post]
func (h *UserHandler) UploadFaces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isu := vars["isu"]

	if isu == "" {
		response.WriteError(w, http.StatusBadRequest, "Incorrect isu in path")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.WriteError(w, http.StatusBadRequest, "cannot parse multipart form: "+err.Error())
		return
	}

	photos := [3]string{"left_face", "right_face", "center_face"}
	photosBytes := make(map[string][]byte)

	for _, key := range photos {
		file, _, err := r.FormFile(key)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, fmt.Sprintf("missing file: %s", key))
			return
		}

		data, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, fmt.Sprintf("cannot read file: %s", key))
			return
		}

		photosBytes[key] = data

	}

	request := AddUserFacesRequest{
		ISU:             isu,
		LeftFacePhoto:   photosBytes["left_face"],
		RightFacePhoto:  photosBytes["right_face"],
		CenterFacePhoto: photosBytes["center_face"],
	}

	err := h.userService.AddUserFaces(r.Context(), request)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, "ok")
}

// AddRole godoc
// @Summary      Добавить роль пользователю
// @Description  Назначает роль пользователю. Принимает ISU и role в JSON.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param Authorization header string true "Bearer <JWT>"
// @Param        request body AddUserRoleRequest true "ISU и роль для добавления"
// @Success      201 {string} string "ok"
// @Failure      400 {object} response.ErrorResponse "Некорректный запрос"
// @Failure      404 {object} response.ErrorResponse "Пользователь не найден"
// @Failure      409 {object} response.ErrorResponse "Роль уже назначена"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Security     BearerAuth
// @Router       /api/user/admin/roles [post]
func (h *UserHandler) AddRole(w http.ResponseWriter, r *http.Request) {
	role, ok := middleware.Role(r.Context())
	if !ok || role != "admin" {
		response.WriteError(w, http.StatusBadRequest, "Invalid role")
		return
	}

	var req AddUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.ISU = strings.TrimSpace(req.ISU)
	req.Role = strings.TrimSpace(req.Role)

	if req.ISU == "" || req.Role == "" {
		response.WriteError(w, http.StatusBadRequest, "isu and role are required")
		return
	}

	if err := h.userService.AddUserRole(r.Context(), req); err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusCreated, "ok")
}

// GetRoles godoc
// @Summary      Получить роли пользователя
// @Description  Возвращает список ролей пользователя. ISU передаётся query-параметром ?isu=...
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        isu query string true "ISU пользователя"
// @Success      200 {object} user.GetUserRolesResponse
// @Failure      400 {object} response.ErrorResponse "Некорректный ISU"
// @Failure      404 {object} response.ErrorResponse "Пользователь не найден"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Router       /api/user/roles [get]
func (h *UserHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	isu := strings.TrimSpace(r.URL.Query().Get("isu"))
	if isu == "" {
		response.WriteError(w, http.StatusBadRequest, "isu is required")
		return
	}

	roles, err := h.userService.GetUserRoles(r.Context(), isu)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := GetUserRolesResponse{
		ISU:   isu,
		Roles: roles,
	}
	response.WriteJSON(w, http.StatusOK, resp)
}
