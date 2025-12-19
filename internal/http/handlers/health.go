package handlers

import "net/http"

// Health godoc
// @Summary Health check
// @Description Проверка доступности сервиса
// @Tags system
// @Produce json
// @Success 200 {object} string
// @Router /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
