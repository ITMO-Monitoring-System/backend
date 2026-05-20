package consent

import (
	"context"
	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
	"net/http"
	"strings"
)

type ConsentService interface {
	GiveBiometricConsent(ctx context.Context, isu, ip, userAgent string) error
	RevokeBiometricConsent(ctx context.Context, isu string) error
	ListConsents(ctx context.Context, isu string) ([]ConsentItem, error)
}

type ConsentHandler struct {
	service ConsentService
}

func NewConsentHandler(service ConsentService) *ConsentHandler {
	return &ConsentHandler{service: service}
}

// GiveBiometric godoc
// @Summary      Согласие на обработку биометрических данных
// @Description  Фиксирует согласие текущего пользователя на обработку биометрии
// @Tags         consents
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]bool
// @Failure      401 {object} response.ErrorResponse "Не авторизован"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Router       /api/user/consents/biometric [post]
func (h *ConsentHandler) GiveBiometric(w http.ResponseWriter, r *http.Request) {
	isu, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(isu) == "" {
		response.WriteError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	if err := h.service.GiveBiometricConsent(r.Context(), isu, httputil.ClientIP(r), r.UserAgent()); err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// RevokeBiometric godoc
// @Summary      Отзыв согласия на обработку биометрии
// @Description  Отзывает согласие и удаляет биометрические данные пользователя
// @Tags         consents
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]bool
// @Failure      401 {object} response.ErrorResponse "Не авторизован"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Router       /api/user/consents/biometric [delete]
func (h *ConsentHandler) RevokeBiometric(w http.ResponseWriter, r *http.Request) {
	isu, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(isu) == "" {
		response.WriteError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	if err := h.service.RevokeBiometricConsent(r.Context(), isu); err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// List godoc
// @Summary      Журнал согласий пользователя
// @Description  Возвращает все согласия текущего пользователя и их статус
// @Tags         consents
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} consent.ConsentsResponse
// @Failure      401 {object} response.ErrorResponse "Не авторизован"
// @Failure      500 {object} response.ErrorResponse "Внутренняя ошибка"
// @Router       /api/user/consents [get]
func (h *ConsentHandler) List(w http.ResponseWriter, r *http.Request) {
	isu, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(isu) == "" {
		response.WriteError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	items, err := h.service.ListConsents(r.Context(), isu)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.WriteJSON(w, http.StatusOK, ConsentsResponse{ISU: isu, Consents: items})
}
