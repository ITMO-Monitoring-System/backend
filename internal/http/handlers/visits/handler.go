package visits

import (
	"context"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
	"net/http"
)

type visitsService interface {
	GetVisitedSubjectsByISU(ctx context.Context, isu string) ([]domain.Subject, error)
}

type VisitsHandler struct {
	visitsService visitsService
}

func NewVisitsHandler(visitsService visitsService) *VisitsHandler {
	return &VisitsHandler{visitsService: visitsService}
}

// GetVisitedSubjects godoc
// @Summary      Получить предметы, по которым студент посещал лекции
// @Description  Возвращает уникальный список предметов (subjects), по которым есть записи в visits.lectures_visiting для указанного isu.
// @Tags         visits
// @Accept       json
// @Produce      json
// @Param Authorization header string true "Bearer <JWT>"
// @Success      200 {object} visits.GetVisitedSubjectsResponse
// @Failure      400 {object} response.ErrorResponse "isu is required"
// @Failure      500 {object} response.ErrorResponse "internal error"
// @Security     BearerAuth
// @Router       /api/visits/lectures/subjects [get]
func (h *VisitsHandler) GetVisitedSubjects(w http.ResponseWriter, r *http.Request) {
	isu, ok := middleware.UserID(r.Context())
	if isu == "" || !ok {
		response.WriteError(w, http.StatusBadRequest, "isu is required")
		return
	}

	subjects, err := h.visitsService.GetVisitedSubjectsByISU(r.Context(), isu)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]SubjectDTO, 0, len(subjects))
	for _, s := range subjects {
		out = append(out, SubjectDTO{
			ID:   s.ID,
			Name: s.Name,
		})
	}

	resp := GetVisitedSubjectsResponse{
		ISU:      isu,
		Subjects: out,
	}

	response.WriteJSON(w, http.StatusOK, resp)
}
