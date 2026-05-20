package auth

import (
	"context"
	"encoding/json"
	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
	"net/http"
	"strings"
)

type authService interface {
	Login(ctx context.Context, request LoginRequest) (*LoginResponse, error)
	Register(ctx context.Context, request RegisterRequest, ip, userAgent string) error
	GetMe(ctx context.Context, isu string) (*MeResponse, error)
}

type AuthHandler struct {
	authService authService
}

func NewAuthHandler(authService authService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login godoc
// @Summary      Аутентификация пользователя
// @Description  Проверяет ISU и пароль, возвращает JWT access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body auth.LoginRequest true "Данные для входа"
// @Success      200 {object} auth.LoginResponse
// @Failure      400 {object} response.ErrorResponse "Некорректный JSON"
// @Failure      401 {object} response.ErrorResponse "Неверные учетные данные"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Router       /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// Register godoc
// @Summary      Регистрация нового пользователя
// @Description  Создаёт пользователя с ролью student, хеширует пароль, привязывает к группе
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body auth.RegisterRequest true "Данные для регистрации"
// @Success      201 {object} map[string]bool
// @Failure      400 {object} response.ErrorResponse "Некорректные данные"
// @Failure      409 {object} response.ErrorResponse "Пользователь уже существует"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Router       /api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad request")
		return
	}

	err := h.authService.Register(r.Context(), req, httputil.ClientIP(r), r.UserAgent())
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			response.WriteError(w, http.StatusConflict, "пользователь с таким ISU уже существует")
			return
		}
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusCreated, map[string]bool{"success": true})
}

// Me godoc
// @Summary      Профиль текущего пользователя
// @Description  Возвращает данные пользователя по JWT-токену
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} auth.MeResponse
// @Failure      401 {object} response.ErrorResponse "Не авторизован"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Router       /api/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok || userID == "" {
		response.WriteError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	me, err := h.authService.GetMe(r.Context(), userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "user not found")
		return
	}

	// Используем роль из JWT-claims (текущая сессия), а не из БД
	role, roleOk := middleware.Role(r.Context())
	if roleOk && role != "" {
		me.Role = role
	}

	response.WriteJSON(w, http.StatusOK, me)
}
