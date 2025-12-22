package postgres

import (
	"context"
	"monitoring_backend/internal/domain"
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
