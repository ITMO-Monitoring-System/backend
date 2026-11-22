package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/domain"
	lectdto "monitoring_backend/internal/http/handlers/lecture"
)

type LectureService struct {
	db *pgxpool.Pool
}

func NewLectureService(db *pgxpool.Pool) *LectureService {
	return &LectureService{db: db}
}

const lectureCatalogQuery = `
 SELECT id, date, subject_id, teacher_id, subject_name, group_ids,
        department_ids, department_codes, status
 FROM universities_data.lecture_catalog
`

func scanLecture(row pgx.CollectableRow) (lectdto.LectureResponse, error) {
	var lecture lectdto.LectureResponse
	err := row.Scan(
		&lecture.ID, &lecture.Date, &lecture.SubjectID, &lecture.TeacherID,
		&lecture.SubjectName, &lecture.GroupIDs, &lecture.DepartmentIDs,
		&lecture.DepartmentCodes, &lecture.Status,
	)
	return lecture, err
}

func (s *LectureService) Create(ctx context.Context, req lectdto.CreateLectureRequest) (lectdto.LectureResponse, error) {
	return s.save(ctx, 0, req)
}

func (s *LectureService) Update(ctx context.Context, id int64, req lectdto.CreateLectureRequest) (lectdto.LectureResponse, error) {
	return s.save(ctx, id, req)
}

func (s *LectureService) save(ctx context.Context, id int64, req lectdto.CreateLectureRequest) (lectdto.LectureResponse, error) {
	var result lectdto.LectureResponse
	groups := uniqueStrings(req.GroupIDs)
	if len(groups) == 0 {
		return result, fmt.Errorf("%w: select a group", domain.ErrInvalidInput)
	}
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if id == 0 {
			err := tx.QueryRow(ctx, `INSERT INTO universities_data.lectures (date, subject_id, teacher_id)
                VALUES ($1, $2, $3)
                RETURNING id`, req.Date, req.SubjectID, req.TeacherID).Scan(&id)
			if err != nil {
				return err
			}
		} else {
			tag, err := tx.Exec(ctx, `UPDATE universities_data.lectures
                SET date = $1, subject_id = $2
                WHERE id = $3 AND teacher_id = $4`, req.Date, req.SubjectID, id, req.TeacherID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return pgx.ErrNoRows
			}
			if _, err = tx.Exec(ctx, `DELETE FROM universities_data.lectures_groups WHERE lecture_id = $1`, id); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO universities_data.lectures_groups (lecture_id, group_id)
            SELECT $1, unnest($2::text[])`, id, groups); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, lectureCatalogQuery+"WHERE id = $1", id)
		if err != nil {
			return err
		}
		result, err = pgx.CollectExactlyOneRow(rows, scanLecture)
		return err
	})
	return result, err
}

func (s *LectureService) GetByID(ctx context.Context, req lectdto.GetLectureByIDRequest) (lectdto.LectureResponse, error) {
	rows, err := s.db.Query(ctx, lectureCatalogQuery+"WHERE id = $1", req.ID)
	if err != nil {
		return lectdto.LectureResponse{}, err
	}
	return pgx.CollectExactlyOneRow(rows, scanLecture)
}

func (s *LectureService) list(ctx context.Context, predicate string, args ...any) ([]lectdto.LectureResponse, error) {
	rows, err := s.db.Query(ctx, lectureCatalogQuery+"WHERE "+predicate+" ORDER BY date DESC, id DESC", args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanLecture)
}

func (s *LectureService) ListByTeacher(ctx context.Context, req lectdto.ListLecturesByTeacherRequest) ([]lectdto.LectureResponse, error) {
	var from, to any
	if !req.From.IsZero() {
		from = req.From
	}
	if !req.To.IsZero() {
		to = req.To
	}
	return s.list(ctx, `teacher_id = $1
        AND ($2::timestamptz IS NULL OR date >= $2)
        AND ($3::timestamptz IS NULL OR date <= $3)`, req.TeacherID, from, to)
}

func (s *LectureService) ListBySubject(ctx context.Context, req lectdto.ListLecturesBySubjectRequest) ([]lectdto.LectureResponse, error) {
	return s.list(ctx, "subject_id = $1 AND date BETWEEN $2 AND $3", req.SubjectID, req.From, req.To)
}

func (s *LectureService) ListByGroup(ctx context.Context, req lectdto.ListLecturesByGroupRequest) ([]lectdto.LectureResponse, error) {
	return s.list(ctx, `id IN (SELECT lecture_id FROM universities_data.lectures_groups WHERE group_id = $1) AND date BETWEEN $2 AND $3`, req.GroupCode, req.From, req.To)
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
