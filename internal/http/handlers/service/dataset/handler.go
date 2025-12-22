package dataset

import (
	"context"
	response2 "monitoring_backend/internal/http/response"
	"net/http"
)

type DatasetService interface {
	Get(ctx context.Context) ([]StudentResponse, error)
}

type DatasetHandler struct {
	serv DatasetService
}

func NewDatasetHandler(serv DatasetService) *DatasetHandler {
	return &DatasetHandler{serv}
}

// Get godoc
// @Summary      Получить датасет эмбеддингов лиц
// @Description  Возвращает список пользователей с эмбеддингами лиц (левый, правый и центральный ракурс).
// @Tags         dataset
// @Produce      json
// @Success      200 {object} dataset.DatasetResponse "Датасет успешно получен"
// @Failure      500 {object} response.ErrorResponse "Ошибка при получении датасета"
// @Router       /api/service/dataset [get]
func (h *DatasetHandler) Get(w http.ResponseWriter, r *http.Request) {
	response, err := h.serv.Get(r.Context())
	if err != nil {
		response2.WriteError(w, http.StatusInternalServerError, "failed to get dataset")
		return
	}

	response2.WriteJSON(w, http.StatusOK, DatasetResponse{UsersData: response})
}
