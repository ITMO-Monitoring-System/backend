package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/handlers/visits"
)

const maxTeacherReportRows = 50000

func (r *lectureVisitsRepository) GetTeacherReport(ctx context.Context, teacherISU string, subjectID int64, f visits.TeacherReportFilter) (visits.TeacherReport, error) {
	out := visits.TeacherReport{Lectures: []visits.ReportLecture{}, Items: []visits.AttendanceRow{}}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT l.id, l.date,
  COALESCE(array_agg(lg.group_id ORDER BY lg.group_id) FILTER (WHERE lg.group_id IS NOT NULL), ARRAY[]::varchar[])
  FROM universities_data.lectures l
  LEFT JOIN universities_data.lectures_groups lg ON lg.lecture_id = l.id
  WHERE l.teacher_id = $1 AND l.subject_id = $2 AND l.date >= $3 AND l.date < $4 AND l.date <= now()
  GROUP BY l.id, l.date ORDER BY l.date, l.id`, teacherISU, subjectID, f.DateFrom, f.DateTo)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var l visits.ReportLecture
		if err := rows.Scan(&l.LectureID, &l.Date, &l.Groups); err != nil {
			rows.Close()
			return out, err
		}
		out.Lectures = append(out.Lectures, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	// The roster supplies zero rows for missing detections, so absences stay in denominators.
	rows, err = tx.Query(ctx, `WITH selected_lectures AS (
  SELECT id FROM universities_data.lectures
  WHERE teacher_id = $1 AND subject_id = $2 AND date >= $3 AND date < $4 AND date <= now()
 ), roster AS (
  SELECT l.id AS lecture_id, lg.group_id AS group_code, sg.user_id
  FROM selected_lectures l
  JOIN universities_data.lectures_groups lg ON lg.lecture_id = l.id
  JOIN universities_data.students_groups sg ON sg.group_code = lg.group_id
 ), snaps AS (
  SELECT lv.lecture_id, lv.user_id, lv.date,
   LEAD(lv.date) OVER (PARTITION BY lv.lecture_id, lv.user_id ORDER BY lv.date) AS next_time
  FROM visits.lectures_visiting lv
  JOIN roster r ON r.lecture_id = lv.lecture_id AND r.user_id = lv.user_id
 ), presence AS (
  SELECT lecture_id, user_id, COUNT(*) AS detection_count,
   COALESCE(SUM(CASE WHEN next_time IS NOT NULL AND next_time - date <= make_interval(secs => $5)
    THEN EXTRACT(EPOCH FROM (next_time - date)) ELSE 0 END), 0)::bigint AS present_seconds
  FROM snaps GROUP BY lecture_id, user_id
 )
 SELECT r.lecture_id, r.group_code, u.isu, u.first_name, u.last_name, u.patronymic,
  COALESCE(p.present_seconds,0), COALESCE(p.detection_count,0)
 FROM roster r JOIN cores.users u ON u.isu = r.user_id
 LEFT JOIN presence p ON p.lecture_id = r.lecture_id AND p.user_id = r.user_id
 ORDER BY r.lecture_id, r.group_code, u.last_name, u.first_name, u.isu LIMIT $6`,
		teacherISU, subjectID, f.DateFrom, f.DateTo, f.GapSeconds, maxTeacherReportRows+1)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item visits.AttendanceRow
		if err := rows.Scan(&item.LectureID, &item.GroupCode, &item.ISU, &item.FirstName, &item.LastName, &item.Patronymic, &item.PresentSeconds, &item.DetectionCount); err != nil {
			rows.Close()
			return out, err
		}
		out.Items = append(out.Items, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) > maxTeacherReportRows {
		return visits.TeacherReport{}, domain.ErrReportTooLarge
	}
	return out, tx.Commit(ctx)
}
