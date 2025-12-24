package auth

import (
	"context"
	"encoding/json"
	"monitoring_backend/internal/http/response"
	"net/http"
)

type authService interface {
	Login(ctx context.Context, request LoginRequest) (*LoginResponse, error)
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
