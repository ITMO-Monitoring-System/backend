package auth

import (
	"context"
	"encoding/json"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
	"net/http"
)

type authService interface {
	Login(ctx context.Context, request LoginRequest) (*LoginResponse, error)
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
