package visits

import (
	"context"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type GetLecturesFilter struct {
	DateFrom   *time.Time
	DateTo     *time.Time
	Order      string // "asc"|"desc"
	Page       int
	PageSize   int
	GapSeconds int // для склейки снапшотов
}

type LectureAttendance struct {
	LectureID      int64
	Date           time.Time
	TeacherISU     string
	PresentSeconds int64
}

type visitsService interface {
	GetVisitedSubjectsByISU(ctx context.Context, isu string) ([]domain.Subject, error)
	GetStudentLecturesBySubject(ctx context.Context, isu string, subjectID int64, filter GetLecturesFilter) (items []LectureAttendance, total int, err error)

	GetTeacherLecturesBySubject(
		ctx context.Context,
		teacherISU string,
		subjectID int64,
		filter TeacherLecturesFilter,
	) (items []TeacherLecture, total int, err error)

	GetLectureGroups(
		ctx context.Context,
		teacherISU string,
		lectureID int64,
	) ([]string, error)

	GetLectureGroupStudents(
		ctx context.Context,
		teacherISU string,
		lectureID int64,
		groupCode string,
		page int,
		pageSize int,
		gapSeconds int,
	) (items []StudentOnLecture, total int, err error)

	GetTeacherSubjects(ctx context.Context, teacherISU string) ([]SubjectDTO, error)
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

// GetStudentLecturesBySubject godoc
// @Summary      Лекции студента по предмету
// @Description  Возвращает лекции по предмету (сортировка по дате) и время присутствия студента на каждой лекции (секунды). ISU берётся из JWT.
// @Tags         visits
// @Accept       json
// @Produce      json
// @Param        subject_id path int true "ID предмета"
// @Param        date_from  query string false "Начало периода (RFC3339 или YYYY-MM-DD)"
// @Param        date_to    query string false "Конец периода (RFC3339 или YYYY-MM-DD)"
// @Param        order      query string false "Сортировка по дате: asc или desc (по умолчанию desc)"
// @Param        page       query int false "Страница (по умолчанию 1)"
// @Param        page_size  query int false "Размер страницы (по умолчанию 20)"
// @Param        gap_seconds query int false "Максимальный разрыв между снапшотами для склейки (сек), по умолчанию 120"
// @Success      200 {object} visits.GetStudentLecturesBySubjectResponse
// @Failure      401 {object} response.ErrorResponse "Unauthorized"
// @Failure      400 {object} response.ErrorResponse "Bad request"
// @Failure      500 {object} response.ErrorResponse "Internal error"
// @Security     BearerAuth
// @Router       /api/visits/lectures/{subject_id} [get]
func (h *VisitsHandler) GetStudentLecturesBySubject(w http.ResponseWriter, r *http.Request) {
	isu, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(isu) == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, ok := middleware.Role(r.Context())
	if !ok || role != "student" {
		response.WriteError(w, http.StatusUnauthorized, "Access denied")
	}

	vars := mux.Vars(r)
	subjectStr := strings.TrimSpace(vars["subject_id"])
	subjectID, err := strconv.ParseInt(subjectStr, 10, 64)
	if err != nil || subjectID <= 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid subject_id")
		return
	}

	filter, err := parseLecturesFilter(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	items, total, err := h.visitsService.GetStudentLecturesBySubject(r.Context(), isu, subjectID, filter)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]LectureAttendanceItem, 0, len(items))
	for _, it := range items {
		out = append(out, LectureAttendanceItem{
			LectureID:      it.LectureID,
			Date:           it.Date.UTC().Format(time.RFC3339),
			TeacherISU:     it.TeacherISU,
			PresentSeconds: it.PresentSeconds,
		})
	}

	resp := GetStudentLecturesBySubjectResponse{
		SubjectID: subjectID,
		ISU:       isu,
		Items:     out,
		Meta: PageMeta{
			Page:     filter.Page,
			PageSize: filter.PageSize,
			Total:    total,
		},
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

func parseLecturesFilter(r *http.Request) (GetLecturesFilter, error) {
	q := r.URL.Query()

	order := strings.ToLower(strings.TrimSpace(q.Get("order")))
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		return GetLecturesFilter{}, httpError("order must be asc or desc")
	}

	page := intFromQuery(q.Get("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := intFromQuery(q.Get("page_size"), 20)
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	gapSeconds := intFromQuery(q.Get("gap_seconds"), 120)
	if gapSeconds < 1 {
		gapSeconds = 120
	}

	var dateFrom *time.Time
	if s := strings.TrimSpace(q.Get("date_from")); s != "" {
		t, err := parseDate(s)
		if err != nil {
			return GetLecturesFilter{}, httpError("invalid date_from")
		}
		dateFrom = &t
	}

	var dateTo *time.Time
	if s := strings.TrimSpace(q.Get("date_to")); s != "" {
		t, err := parseDate(s)
		if err != nil {
			return GetLecturesFilter{}, httpError("invalid date_to")
		}
		dateTo = &t
	}

	if dateFrom != nil && dateTo != nil && dateTo.Before(*dateFrom) {
		return GetLecturesFilter{}, httpError("date_to must be >= date_from")
	}

	return GetLecturesFilter{
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Order:      order,
		Page:       page,
		PageSize:   pageSize,
		GapSeconds: gapSeconds,
	}, nil
}

func parseDate(s string) (time.Time, error) {
	// RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// YYYY-MM-DD
	return time.Parse("2006-01-02", s)
}

func intFromQuery(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

type TeacherLecturesFilter struct {
	DateFrom *time.Time
	DateTo   *time.Time
	Order    string // "asc"|"desc"
	Page     int
	PageSize int
}

type TeacherLecture struct {
	LectureID int64
	Date      time.Time
}

type StudentOnLecture struct {
	ISU            string
	FirstName      string
	LastName       string
	Patronymic     *string
	PresentSeconds int64
}

// GetTeacherLecturesBySubject godoc
// @Summary      Лекции преподавателя по предмету
// @Description  Возвращает лекции по предмету для текущего преподавателя (ISU из JWT). Период опционально. Сортировка asc/desc. Есть пагинация.
// @Tags         visits
// @Accept       json
// @Produce      json
// @Param        subject_id path int true "ID предмета"
// @Param        date_from  query string false "Начало периода (RFC3339 или YYYY-MM-DD)"
// @Param        date_to    query string false "Конец периода (RFC3339 или YYYY-MM-DD)"
// @Param        order      query string false "Сортировка по дате: asc или desc (по умолчанию desc)"
// @Param        page       query int false "Страница (по умолчанию 1)"
// @Param        page_size  query int false "Размер страницы (по умолчанию 20)"
// @Success      200 {object} visits.GetTeacherLecturesResponse
// @Failure      401 {object} response.ErrorResponse "Unauthorized"
// @Failure      400 {object} response.ErrorResponse "Bad request"
// @Failure      500 {object} response.ErrorResponse "Internal error"
// @Security     BearerAuth
// @Router       /api/visits/teacher/{subject_id}/lectures [get]
func (h *VisitsHandler) GetTeacherLecturesBySubject(w http.ResponseWriter, r *http.Request) {
	teacherISU, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(teacherISU) == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subjectID, err := parseIDPath(mux.Vars(r), "subject_id")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid subject_id")
		return
	}

	filter, err := parseTeacherLecturesFilter(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	items, total, err := h.visitsService.GetTeacherLecturesBySubject(r.Context(), teacherISU, subjectID, filter)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]TeacherLectureItem, 0, len(items))
	for _, it := range items {
		out = append(out, TeacherLectureItem{
			LectureID: it.LectureID,
			Date:      it.Date.UTC().Format(time.RFC3339),
		})
	}

	resp := GetTeacherLecturesResponse{
		SubjectID:  subjectID,
		TeacherISU: teacherISU,
		Items:      out,
		Meta: PageMeta{
			Page:     filter.Page,
			PageSize: filter.PageSize,
			Total:    total,
		},
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// GetLectureGroups godoc
// @Summary      Группы на лекции
// @Description  Возвращает список групп, привязанных к лекции (без пагинации). Доступ только преподавателю (ISU из JWT) для его лекции.
// @Tags         visits
// @Accept       json
// @Produce      json
// @Param        lecture_id path int true "ID лекции"
// @Success      200 {object} visits.GetLectureGroupsResponse
// @Failure      401 {object} response.ErrorResponse "Unauthorized"
// @Failure      400 {object} response.ErrorResponse "Bad request"
// @Failure      500 {object} response.ErrorResponse "Internal error"
// @Security     BearerAuth
// @Router       /api/visits/teacher/{lecture_id}/groups [get]
func (h *VisitsHandler) GetLectureGroups(w http.ResponseWriter, r *http.Request) {
	teacherISU, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(teacherISU) == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	lectureID, err := parseIDPath(mux.Vars(r), "lecture_id")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid lecture_id")
		return
	}

	groups, err := h.visitsService.GetLectureGroups(r.Context(), teacherISU, lectureID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]GroupItem, 0, len(groups))
	for _, g := range groups {
		out = append(out, GroupItem{GroupCode: g})
	}

	resp := GetLectureGroupsResponse{
		LectureID: lectureID,
		Groups:    out,
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

// GetLectureGroupStudents godoc
// @Summary      Студенты группы на лекции и время присутствия
// @Description  Возвращает студентов выбранной группы на выбранной лекции и сколько секунд каждый присутствовал. Пагинация есть, фильтров/сортировок нет.
// @Tags         visits
// @Accept       json
// @Produce      json
// @Param        lecture_id path int true "ID лекции"
// @Param        group_code path string true "Код группы"
// @Param        page       query int false "Страница (по умолчанию 1)"
// @Param        page_size  query int false "Размер страницы (по умолчанию 50)"
// @Param        gap_seconds query int false "Максимальный разрыв между снапшотами для склейки (сек), по умолчанию 120"
// @Success      200 {object} visits.GetLectureGroupStudentsResponse
// @Failure      401 {object} response.ErrorResponse "Unauthorized"
// @Failure      400 {object} response.ErrorResponse "Bad request"
// @Failure      500 {object} response.ErrorResponse "Internal error"
// @Security     BearerAuth
// @Router       /api/visits/teacher/{lecture_id}/{group_code}/students [get]
func (h *VisitsHandler) GetLectureGroupStudents(w http.ResponseWriter, r *http.Request) {
	teacherISU, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(teacherISU) == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	lectureID, err := parseIDPath(vars, "lecture_id")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid lecture_id")
		return
	}

	groupCode := strings.TrimSpace(vars["group_code"])
	if groupCode == "" {
		response.WriteError(w, http.StatusBadRequest, "invalid group_code")
		return
	}

	page := intFromQuery(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := intFromQuery(r.URL.Query().Get("page_size"), 50)
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	gapSeconds := intFromQuery(r.URL.Query().Get("gap_seconds"), 120)
	if gapSeconds < 1 {
		gapSeconds = 120
	}

	items, total, err := h.visitsService.GetLectureGroupStudents(r.Context(), teacherISU, lectureID, groupCode, page, pageSize, gapSeconds)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]StudentOnLectureItem, 0, len(items))
	for _, it := range items {
		out = append(out, StudentOnLectureItem{
			ISU:            it.ISU,
			FirstName:      it.FirstName,
			LastName:       it.LastName,
			Patronymic:     it.Patronymic,
			PresentSeconds: it.PresentSeconds,
		})
	}

	resp := GetLectureGroupStudentsResponse{
		LectureID: lectureID,
		GroupCode: groupCode,
		Items:     out,
		Meta: PageMeta{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

func parseTeacherLecturesFilter(r *http.Request) (TeacherLecturesFilter, error) {
	q := r.URL.Query()

	order := strings.ToLower(strings.TrimSpace(q.Get("order")))
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		return TeacherLecturesFilter{}, httpError("order must be asc or desc")
	}

	page := intFromQuery(q.Get("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := intFromQuery(q.Get("page_size"), 20)
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	var dateFrom *time.Time
	if s := strings.TrimSpace(q.Get("date_from")); s != "" {
		t, err := parseDate(s)
		if err != nil {
			return TeacherLecturesFilter{}, httpError("invalid date_from")
		}
		dateFrom = &t
	}

	var dateTo *time.Time
	if s := strings.TrimSpace(q.Get("date_to")); s != "" {
		t, err := parseDate(s)
		if err != nil {
			return TeacherLecturesFilter{}, httpError("invalid date_to")
		}
		dateTo = &t
	}

	if dateFrom != nil && dateTo != nil && dateTo.Before(*dateFrom) {
		return TeacherLecturesFilter{}, httpError("date_to must be >= date_from")
	}

	return TeacherLecturesFilter{
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Order:    order,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetTeacherSubjects godoc
// @Summary      Предметы преподавателя
// @Description  Возвращает список предметов, по которым у текущего преподавателя есть лекции. ISU берётся из JWT.
// @Tags         visits
// @Accept       json
// @Produce      json
// @Success      200 {object} visits.GetTeacherSubjectsResponse
// @Failure      401 {object} response.ErrorResponse "Unauthorized"
// @Failure      500 {object} response.ErrorResponse "Internal error"
// @Security     BearerAuth
// @Router       /api/visits/teacher/subjects [get]
func (h *VisitsHandler) GetTeacherSubjects(w http.ResponseWriter, r *http.Request) {
	teacherISU, ok := middleware.UserID(r.Context())
	if !ok || strings.TrimSpace(teacherISU) == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subjects, err := h.visitsService.GetTeacherSubjects(r.Context(), teacherISU)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := GetTeacherSubjectsResponse{
		TeacherISU: teacherISU,
		Subjects:   subjects,
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

func parseIDPath(vars map[string]string, key string) (int64, error) {
	s := strings.TrimSpace(vars[key])
	return strconv.ParseInt(s, 10, 64)
}

type httpError string

func (e httpError) Error() string { return string(e) }
