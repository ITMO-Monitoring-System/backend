package postgres

import (
	"context"
	"fmt"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/handlers/visits"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type lectureVisitsRepository struct {
	db *pgxpool.Pool
}

func NewLectureVisitsRepository(db *pgxpool.Pool) *lectureVisitsRepository {
	return &lectureVisitsRepository{
		db: db,
	}
}

func (v *lectureVisitsRepository) Add(ctx context.Context, visit domain.LectureVisit) (user *domain.User, err error) {
	tx, err := v.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	const insertQuery = `
		INSERT INTO
			visits.lectures_visiting(lecture_id, user_id, date) 
		VALUES($1, $2, $3);
	`

	_, err = tx.Exec(ctx, insertQuery, visit.LectureID, visit.UserID, visit.Date.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	user = &domain.User{
		ISU: visit.UserID,
	}

	const selectQuery = `
		SELECT
			u.last_name,
			u.first_name,
			u.patronymic,
			sg.group_code
		FROM cores.users u
		JOIN universities_data.students_groups sg on u.isu = sg.user_id
		WHERE u.isu = $1
		LIMIT 1;
	`

	err = tx.QueryRow(ctx, selectQuery, user.ISU).Scan(&user.LastName, &user.FirstName, &user.Patronymic, &user.GroupCode)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (v *lectureVisitsRepository) Exists(ctx context.Context, lectureID int64, userID string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (v *lectureVisitsRepository) ListByLecture(ctx context.Context, lectureID int64) ([]domain.LectureVisit, error) {
	//TODO implement me
	panic("implement me")
}

func (v *lectureVisitsRepository) ListByUser(ctx context.Context, userID string, from, to time.Time) ([]domain.LectureVisit, error) {
	//TODO implement me
	panic("implement me")
}

func (r *lectureVisitsRepository) ListVisitedSubjectsByISU(ctx context.Context, isu string) ([]domain.Subject, error) {
	const q = `
		SELECT DISTINCT
			s.id,
			s.name
		FROM visits.lectures_visiting lv
		JOIN universities_data.lectures l
			ON l.id = lv.lecture_id
		JOIN universities_data.subjects s
			ON s.id = l.subject_id
		WHERE lv.user_id = $1
		ORDER BY s.name;
	`

	rows, err := r.db.Query(ctx, q, isu)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subjects := make([]domain.Subject, 0)
	for rows.Next() {
		var s domain.Subject
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		subjects = append(subjects, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subjects, nil
}

func (r *lectureVisitsRepository) ListStudentLecturesBySubject(ctx context.Context, isu string, subjectID int64, filter visits.GetLecturesFilter) ([]visits.LectureAttendance, int, error) {
	isu = strings.TrimSpace(isu)
	if isu == "" {
		return nil, 0, fmt.Errorf("isu is empty")
	}

	order := "DESC"
	if strings.ToLower(filter.Order) == "asc" {
		order = "ASC"
	}

	limit := filter.PageSize
	offset := (filter.Page - 1) * filter.PageSize

	// 1) total (для пагинации): сколько лекций по subject у этого teacher/студента (без учёта снапшотов)
	totalQuery := `
		SELECT COUNT(*)
		FROM universities_data.lectures l
		WHERE l.subject_id = $1
		  AND ($2::timestamptz IS NULL OR l.date >= $2)
		  AND ($3::timestamptz IS NULL OR l.date <= $3)
		  AND EXISTS (
			  SELECT 1
			  FROM visits.lectures_visiting lv
			  WHERE lv.lecture_id = l.id
			    AND lv.user_id = $4
		  );
	`

	var total int
	if err := r.db.QueryRow(ctx, totalQuery, subjectID, filter.DateFrom, filter.DateTo, isu).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 2) list with present_seconds per lecture
	// present_seconds считаем через LEAD(date) и суммирование разницы, если gap <= filter.GapSeconds
	listQuery := fmt.Sprintf(`
		WITH snaps AS (
			SELECT
				lv.lecture_id,
				lv.date AS snap_time,
				LEAD(lv.date) OVER (PARTITION BY lv.lecture_id ORDER BY lv.date) AS next_time
			FROM visits.lectures_visiting lv
			JOIN universities_data.lectures l ON l.id = lv.lecture_id
			WHERE lv.user_id = $1
			  AND l.subject_id = $2
			  AND ($3::timestamptz IS NULL OR l.date >= $3)
			  AND ($4::timestamptz IS NULL OR l.date <= $4)
		)
		SELECT
			l.id,
			l.date,
			l.teacher_id,
			COALESCE(SUM(
				CASE
					WHEN s.next_time IS NOT NULL
					 AND EXTRACT(EPOCH FROM (s.next_time - s.snap_time)) <= $5
					THEN EXTRACT(EPOCH FROM (s.next_time - s.snap_time))
					ELSE 0
				END
			), 0)::bigint AS present_seconds
		FROM universities_data.lectures l
		JOIN snaps s ON s.lecture_id = l.id
		WHERE l.subject_id = $2
		  AND ($3::timestamptz IS NULL OR l.date >= $3)
		  AND ($4::timestamptz IS NULL OR l.date <= $4)
		GROUP BY l.id, l.date, l.teacher_id
		ORDER BY l.date %s
		LIMIT $6 OFFSET $7;
	`, order)

	rows, err := r.db.Query(ctx, listQuery, isu, subjectID, filter.DateFrom, filter.DateTo, filter.GapSeconds, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]visits.LectureAttendance, 0)
	for rows.Next() {
		var it visits.LectureAttendance
		if err := rows.Scan(&it.LectureID, &it.Date, &it.TeacherISU, &it.PresentSeconds); err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *lectureVisitsRepository) ListTeacherLecturesBySubject(
	ctx context.Context,
	teacherISU string,
	subjectID int64,
	filter visits.TeacherLecturesFilter,
) ([]visits.TeacherLecture, int, error) {
	order := "DESC"
	if strings.ToLower(filter.Order) == "asc" {
		order = "ASC"
	}
	limit := filter.PageSize
	offset := (filter.Page - 1) * filter.PageSize

	totalQuery := `
		SELECT COUNT(*)
		FROM universities_data.lectures l
		WHERE l.teacher_id = $1
		  AND l.subject_id = $2
		  AND ($3::timestamptz IS NULL OR l.date >= $3)
		  AND ($4::timestamptz IS NULL OR l.date <= $4);
	`

	var total int
	if err := r.db.QueryRow(ctx, totalQuery, teacherISU, subjectID, filter.DateFrom, filter.DateTo).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(`
		SELECT l.id, l.date
		FROM universities_data.lectures l
		WHERE l.teacher_id = $1
		  AND l.subject_id = $2
		  AND ($3::timestamptz IS NULL OR l.date >= $3)
		  AND ($4::timestamptz IS NULL OR l.date <= $4)
		ORDER BY l.date %s
		LIMIT $5 OFFSET $6;
	`, order)

	rows, err := r.db.Query(ctx, listQuery, teacherISU, subjectID, filter.DateFrom, filter.DateTo, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]visits.TeacherLecture, 0)
	for rows.Next() {
		var it visits.TeacherLecture
		if err := rows.Scan(&it.LectureID, &it.Date); err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *lectureVisitsRepository) ListLectureGroups(ctx context.Context, teacherISU string, lectureID int64) ([]string, error) {
	// защита: преподаватель может смотреть только свои лекции
	check := `
		SELECT 1
		FROM universities_data.lectures l
		WHERE l.id = $1 AND l.teacher_id = $2;
	`
	var ok int
	if err := r.db.QueryRow(ctx, check, lectureID, teacherISU).Scan(&ok); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT lg.group_id
		FROM universities_data.lectures_groups lg
		WHERE lg.lecture_id = $1
		ORDER BY lg.group_id;
	`, lectureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *lectureVisitsRepository) ListLectureGroupStudents(
	ctx context.Context,
	teacherISU string,
	lectureID int64,
	groupCode string,
	page int,
	pageSize int,
	gapSeconds int,
) ([]visits.StudentOnLecture, int, error) {
	groupCode = strings.TrimSpace(groupCode)
	limit := pageSize
	offset := (page - 1) * pageSize

	// защита: lecture принадлежит teacher и group реально привязана к lecture
	check := `
		SELECT 1
		FROM universities_data.lectures l
		JOIN universities_data.lectures_groups lg ON lg.lecture_id = l.id
		WHERE l.id = $1 AND l.teacher_id = $2 AND lg.group_id = $3;
	`
	var ok int
	if err := r.db.QueryRow(ctx, check, lectureID, teacherISU, groupCode).Scan(&ok); err != nil {
		return nil, 0, err
	}

	totalQuery := `
		SELECT COUNT(*)
		FROM universities_data.students_groups sg
		WHERE sg.group_code = $1;
	`
	var total int
	if err := r.db.QueryRow(ctx, totalQuery, groupCode).Scan(&total); err != nil {
		return nil, 0, err
	}

	// presence per student for this lecture (gapSeconds controls stitching)
	q := `
		WITH group_students AS (
			SELECT sg.user_id
			FROM universities_data.students_groups sg
			WHERE sg.group_code = $1
		),
		snaps AS (
			SELECT
				lv.user_id,
				lv.date AS snap_time,
				LEAD(lv.date) OVER (PARTITION BY lv.user_id ORDER BY lv.date) AS next_time
			FROM visits.lectures_visiting lv
			JOIN group_students gs ON gs.user_id = lv.user_id
			WHERE lv.lecture_id = $2
		),
		presence AS (
			SELECT
				s.user_id,
				COALESCE(SUM(
					CASE
						WHEN s.next_time IS NOT NULL
						 AND EXTRACT(EPOCH FROM (s.next_time - s.snap_time)) <= $3
						THEN EXTRACT(EPOCH FROM (s.next_time - s.snap_time))
						ELSE 0
					END
				), 0)::bigint AS present_seconds
			FROM snaps s
			GROUP BY s.user_id
		)
		SELECT
			u.isu,
			u.first_name,
			u.last_name,
			u.patronymic,
			COALESCE(p.present_seconds, 0)::bigint AS present_seconds
		FROM universities_data.students_groups sg
		JOIN cores.users u ON u.isu = sg.user_id
		LEFT JOIN presence p ON p.user_id = sg.user_id
		WHERE sg.group_code = $1
		ORDER BY u.last_name, u.first_name, u.isu
		LIMIT $4 OFFSET $5;
	`

	rows, err := r.db.Query(ctx, q, groupCode, lectureID, gapSeconds, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]visits.StudentOnLecture, 0)
	for rows.Next() {
		var it visits.StudentOnLecture
		if err := rows.Scan(&it.ISU, &it.FirstName, &it.LastName, &it.Patronymic, &it.PresentSeconds); err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *lectureVisitsRepository) ListTeacherSubjects(ctx context.Context, teacherISU string) ([]visits.SubjectDTO, error) {
	const q = `
		SELECT DISTINCT
			s.id,
			s.name
		FROM universities_data.lectures l
		JOIN universities_data.subjects s
			ON s.id = l.subject_id
		WHERE l.teacher_id = $1
		ORDER BY s.name;
	`

	rows, err := r.db.Query(ctx, q, teacherISU)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]visits.SubjectDTO, 0)
	for rows.Next() {
		var item visits.SubjectDTO
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
