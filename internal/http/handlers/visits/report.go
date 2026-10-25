package visits

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	httputil "monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
)

type TeacherReportFilter struct {
	DateFrom   time.Time
	DateTo     time.Time // exclusive
	GapSeconds int
}

type ReportLecture struct {
	LectureID int64     `json:"lecture_id"`
	Date      time.Time `json:"date"`
	Groups    []string  `json:"groups"`
}

type AttendanceRow struct {
	LectureID      int64   `json:"lecture_id"`
	GroupCode      string  `json:"group_code"`
	ISU            string  `json:"isu"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	Patronymic     *string `json:"patronymic,omitempty"`
	PresentSeconds int64   `json:"present_seconds"`
	DetectionCount int64   `json:"detection_count"`
}

type TeacherReport struct {
	Lectures []ReportLecture `json:"lectures"`
	Items    []AttendanceRow `json:"items"`
}

// GetTeacherReport godoc
// @Summary Полный отчёт по посещаемости лекций преподавателя
// @Description Только свои начавшиеся лекции и текущий состав привязанных групп. Все строки периода без обрезки; для слишком большого отчёта возвращается 400.
// @Tags visits
// @Produce json
// @Param subject_id path int true "Предмет"
// @Param date_from query string false "Начало периода, RFC3339 или YYYY-MM-DD; по умолчанию 90 дней назад"
// @Param date_to query string false "Конец периода: дата включительно, RFC3339 исключительно; по умолчанию сейчас"
// @Param gap_seconds query int false "Допустимый разрыв между отметками: 5–600 секунд, по умолчанию 180"
// @Success 200 {object} visits.TeacherReport
// @Failure 400,401,403,500 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /api/visits/teacher/{subject_id}/report [get]
func (h *VisitsHandler) GetTeacherReport(w http.ResponseWriter, r *http.Request) {
	isu, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(isu) == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if role, _ := middleware.Role(r.Context()); role != "teacher" {
		response.WriteError(w, http.StatusForbidden, "teacher only")
		return
	}
	subjectID, err := parseIDPath(mux.Vars(r), "subject_id")
	if err != nil || subjectID <= 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid subject_id")
		return
	}
	filter, err := parseTeacherReportFilter(r, time.Now())
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	report, err := h.visitsService.GetTeacherReport(r.Context(), isu, subjectID, filter)
	if err != nil {
		httputil.WriteServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, report)
}

func parseTeacherReportFilter(r *http.Request, now time.Time) (TeacherReportFilter, error) {
	f := TeacherReportFilter{DateFrom: now.AddDate(0, 0, -90), DateTo: now, GapSeconds: 180}
	q := r.URL.Query()
	if raw := strings.TrimSpace(q.Get("date_from")); raw != "" {
		parsed, err := parseDate(raw)
		if err != nil {
			return f, fmt.Errorf("Некорректная дата начала")
		}
		f.DateFrom = parsed
	}
	if raw := strings.TrimSpace(q.Get("date_to")); raw != "" {
		parsed, err := parseDate(raw)
		if err != nil {
			return f, fmt.Errorf("Некорректная дата окончания")
		}
		if len(raw) == 10 {
			parsed = parsed.AddDate(0, 0, 1)
		}
		f.DateTo = parsed
	}
	if !f.DateTo.After(f.DateFrom) || f.DateTo.Sub(f.DateFrom) > 366*24*time.Hour {
		return f, fmt.Errorf("Выберите период от 1 до 366 дней")
	}
	if raw := q.Get("gap_seconds"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 5 || n > 600 {
			return f, fmt.Errorf("Допустимый разрыв — от 5 до 600 секунд")
		}
		f.GapSeconds = n
	}
	return f, nil
}
