package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"monitoring_backend/internal/domain"
	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type UserService interface {
	AddUser(ctx context.Context, request AddUserRequest) error
	AddUserFaces(ctx context.Context, request AddUserFacesRequest) error
	AddUserRole(ctx context.Context, request AddUserRoleRequest) error
	GetUserRoles(ctx context.Context, isu string) ([]string, error)
	ListUsers(ctx context.Context, limit, offset int, role string) ([]UserResponse, error)
	DeleteUser(ctx context.Context, isu string) error
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
// @Description  Загружает три фотографии (левая, правая, фронтальная) для авторизованного студента.
// @Tags         users
// @Accept       multipart/form-data
// @Produce      json
// @Param        left_face    formData  file                   true  "Фотография левой стороны лица"
// @Param        right_face   formData  file                   true  "Фотография правой стороны лица"
// @Param        center_face  formData  file                   true  "Фотография фронтальной стороны лица"
// @Success      200          {string}  string                 "ok"
// @Failure      400          {object}  response.ErrorResponse        "Некорректный запрос или отсутствуют файлы"
// @Failure      500          {object}  response.ErrorResponse        "Ошибка сервиса при добавлении фотографий"
// @Security     BearerAuth
// @Router       /api/user/upload/faces [post]
func (h *UserHandler) UploadMyFaces(w http.ResponseWriter, r *http.Request) {
	isu, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(isu) == "" {
		response.WriteError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	h.uploadFacesForISU(w, r, isu)
}

func (h *UserHandler) UploadFaces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isu := vars["isu"]

	if isu == "" {
		response.WriteError(w, http.StatusBadRequest, "Incorrect isu in path")
		return
	}

	role, _ := middleware.Role(r.Context())
	userID, hasUserID := middleware.UserID(r.Context())
	if !hasUserID || userID == "" {
		response.WriteError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	// Только админ может загружать фото за другого пользователя
	if role != "admin" && strings.TrimSpace(userID) != strings.TrimSpace(isu) {
		response.WriteError(w, http.StatusForbidden, "можно загружать только свои фото")
		return
	}

	h.uploadFacesForISU(w, r, isu)
}

func (h *UserHandler) uploadFacesForISU(w http.ResponseWriter, r *http.Request, isu string) {
	request, err := parseAddUserFacesRequest(r, isu)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.userService.AddUserFaces(r.Context(), request); err != nil {
		writeAddFacesError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, "ok")
}

func parseAddUserFacesRequest(r *http.Request, isu string) (AddUserFacesRequest, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return AddUserFacesRequest{}, fmt.Errorf("cannot parse multipart form: %w", err)
	}

	photos := [3]string{"left_face", "right_face", "center_face"}
	photosBytes := make(map[string][]byte)

	for _, key := range photos {
		file, _, err := r.FormFile(key)
		if err != nil {
			return AddUserFacesRequest{}, fmt.Errorf("missing file: %s", key)
		}

		data, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			return AddUserFacesRequest{}, fmt.Errorf("cannot read file: %s", key)
		}

		photosBytes[key] = data

	}

	return AddUserFacesRequest{
		ISU:             isu,
		LeftFacePhoto:   photosBytes["left_face"],
		RightFacePhoto:  photosBytes["right_face"],
		CenterFacePhoto: photosBytes["center_face"],
	}, nil
}

func writeAddFacesError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrBiometricConsentRequired):
		response.WriteError(w, http.StatusForbidden, "нет действующего согласия на обработку биометрических персональных данных")
		return
	case errors.Is(err, domain.ErrEmbeddingServiceUnavailable):
		response.WriteError(w, http.StatusServiceUnavailable, "embedding service unavailable")
		return
	case errors.Is(err, domain.ErrFaceNotDetected):
		response.WriteError(w, http.StatusUnprocessableEntity, "face was not detected in one or more photos")
		return
	default:
		httputil.WriteServiceError(w, err)
	}
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

// ListUsers godoc
// @Summary      Список пользователей
// @Description  Возвращает список пользователей с пагинацией. Опциональный фильтр по роли.
// @Tags         users
// @Produce      json
// @Param Authorization header string true "Bearer <JWT>"
// @Param        limit  query int    false "Кол-во записей (default 50)"
// @Param        offset query int    false "Смещение (default 0)"
// @Param        role   query string false "Фильтр по роли (admin, teacher, student)"
// @Success      200 {object} user.ListUsersResponse
// @Failure      403 {object} response.ErrorResponse "Нет прав"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Security     BearerAuth
// @Router       /api/users [get]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	role, ok := middleware.Role(r.Context())
	if !ok || role != "admin" {
		response.WriteError(w, http.StatusForbidden, "admin only")
		return
	}

	limit, err := httputil.QueryInt(r, "limit", 50)
	if err != nil {
		limit = 50
	}
	offset, err := httputil.QueryInt(r, "offset", 0)
	if err != nil {
		offset = 0
	}
	filterRole := strings.TrimSpace(r.URL.Query().Get("role"))

	users, err := h.userService.ListUsers(r.Context(), limit, offset, filterRole)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, ListUsersResponse{Users: users})
}

// DeleteUser godoc
// @Summary      Удалить пользователя
// @Description  Удаляет пользователя по ISU.
// @Tags         users
// @Produce      json
// @Param Authorization header string true "Bearer <JWT>"
// @Param        isu path string true "ISU пользователя"
// @Success      200 {string} string "ok"
// @Failure      400 {object} response.ErrorResponse "ISU не указан"
// @Failure      403 {object} response.ErrorResponse "Нет прав"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Security     BearerAuth
// @Router       /api/users/{isu} [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	role, ok := middleware.Role(r.Context())
	if !ok || role != "admin" {
		response.WriteError(w, http.StatusForbidden, "admin only")
		return
	}

	isu := strings.TrimSpace(mux.Vars(r)["isu"])
	if isu == "" {
		response.WriteError(w, http.StatusBadRequest, "isu is required")
		return
	}

	if err := h.userService.DeleteUser(r.Context(), isu); err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, "ok")
}
