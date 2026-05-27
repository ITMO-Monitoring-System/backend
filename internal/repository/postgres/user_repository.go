package postgres

import (
	"context"
	"errors"
	"fmt"
	"monitoring_backend/internal/domain"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *userRepository {
	return &userRepository{
		db: db,
	}
}

func (u *userRepository) Create(ctx context.Context, user *domain.User) error {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return err
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
    INSERT INTO cores.users (isu, first_name, last_name, patronymic) VALUES ($1, $2, $3, $4)
  `

	_, err = tx.Exec(ctx, insertQuery, user.ISU, user.FirstName, user.LastName, user.Patronymic)
	if err != nil {
		return err
	}

	return nil
}

func (u *userRepository) AddFaceEmbeddings(ctx context.Context, user *domain.UserFaces) (err error) {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return err
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
		INSERT INTO cores.face_images (student_id,
			left_face, left_face_embedding,
			right_face, right_face_embedding,
			full_face, full_face_embedding,
			updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (student_id) DO UPDATE SET
			left_face            = EXCLUDED.left_face,
			left_face_embedding  = EXCLUDED.left_face_embedding,
			right_face           = EXCLUDED.right_face,
			right_face_embedding = EXCLUDED.right_face_embedding,
			full_face            = EXCLUDED.full_face,
			full_face_embedding  = EXCLUDED.full_face_embedding,
			updated_at           = now();
 `

	_, err = tx.Exec(ctx, insertQuery,
		&user.User.ISU,
		&user.LeftFace,
		&user.LeftFaceEmbedding,
		&user.RightFace,
		&user.RightFaceEmbedding,
		&user.CenterFace,
		&user.CenterFaceEmbedding)
	if err != nil {
		return err
	}

	return nil
}

func (u *userRepository) GetByISU(ctx context.Context, isu string) (*domain.User, error) {
	var user domain.User

	const selectQuery = `
		SELECT 
			u.isu,
			u.first_name,
			u.last_name,
			u.patronymic
		FROM cores.users u
		WHERE isu = $1
		LIMIT 1;
	`
	err := u.db.QueryRow(ctx, selectQuery, isu).Scan(&user.ISU, &user.FirstName, &user.LastName, &user.Patronymic)
	if err != nil {
		return nil, err
	}

	const selectRolesQuery = `
	SELECT 
		r.role
	FROM cores.users_roles r
	WHERE r.isu = $1;
	`

	var roles []string

	rows, err := u.db.Query(ctx, selectRolesQuery, isu)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		err := rows.Scan(&role)
		if err != nil {
			return nil, err
		}
		roles = append(roles, strings.ToLower(role))
	}

	user.Roles = roles

	return &user, nil
}

func (u *userRepository) GetUserPassword(ctx context.Context, isu string) (string, error) {
	const selectQuery = `
		SELECT
			u.password
		FROM cores.users_passwords u
		WHERE u.isu = $1;
	`
	var password string

	err := u.db.QueryRow(ctx, selectQuery, isu).Scan(&password)
	if err != nil {
		return "", err
	}

	return password, nil
}

func (u *userRepository) Update(ctx context.Context, user domain.User) error {
	//TODO implement me
	panic("implement me")
}

func (u *userRepository) Delete(ctx context.Context, isu string) error {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	childDeletes := []string{
		`DELETE FROM visits.lectures_visiting   WHERE user_id    = $1`,
		`DELETE FROM visits.practices_visiting  WHERE user_id    = $1`,
		`DELETE FROM universities_data.students_groups WHERE user_id = $1`,
		`DELETE FROM cores.face_images          WHERE student_id = $1`,
		`DELETE FROM cores.users_roles          WHERE isu        = $1`,
		`DELETE FROM cores.users_passwords      WHERE isu        = $1`,
	}
	for _, q := range childDeletes {
		if _, err := tx.Exec(ctx, q, isu); err != nil {
			return err
		}
	}

	tag, err := tx.Exec(ctx, `DELETE FROM cores.users WHERE isu = $1`, isu)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.ErrUserHasLectures
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return tx.Commit(ctx)
}

func (u *userRepository) List(ctx context.Context, limit, offset int, role string) ([]domain.User, error) {
	var query string
	var args []any

	if role != "" {
		query = `
			SELECT u.isu, u.first_name, u.last_name, u.patronymic,
			       COALESCE(array_agg(r2.role) FILTER (WHERE r2.role IS NOT NULL), ARRAY[]::text[]) AS roles
			FROM cores.users u
			JOIN cores.users_roles r ON r.isu = u.isu AND r.role = $1
			LEFT JOIN cores.users_roles r2 ON r2.isu = u.isu
			GROUP BY u.isu, u.first_name, u.last_name, u.patronymic
			ORDER BY u.last_name NULLS LAST, u.first_name NULLS LAST, u.patronymic NULLS LAST, u.isu
			LIMIT $2 OFFSET $3
		`
		args = []any{role, limit, offset}
	} else {
		query = `
			SELECT u.isu, u.first_name, u.last_name, u.patronymic,
			       COALESCE(array_agg(r.role) FILTER (WHERE r.role IS NOT NULL), ARRAY[]::text[]) AS roles
			FROM cores.users u
			LEFT JOIN cores.users_roles r ON r.isu = u.isu
			GROUP BY u.isu, u.first_name, u.last_name, u.patronymic
			ORDER BY u.last_name NULLS LAST, u.first_name NULLS LAST, u.patronymic NULLS LAST, u.isu
			LIMIT $1 OFFSET $2
		`
		args = []any{limit, offset}
	}

	rows, err := u.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var usr domain.User
		if err := rows.Scan(&usr.ISU, &usr.FirstName, &usr.LastName, &usr.Patronymic, &usr.Roles); err != nil {
			return nil, err
		}
		users = append(users, usr)
	}
	return users, rows.Err()
}

func (u *userRepository) SetPassword(ctx context.Context, isu, password string) error {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return err
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
		INSERT INTO cores.users_passwords (isu, password) VALUES ($1, $2) ON CONFLICT (isu) DO UPDATE SET password = $2;
	`

	_, err = tx.Exec(ctx, insertQuery, isu, password)
	if err != nil {
		return err
	}

	return nil
}

func (u *userRepository) AddRole(ctx context.Context, isu, role string) error {
	isu = strings.TrimSpace(isu)
	role = strings.ToLower(strings.TrimSpace(role))

	_, err := u.db.Exec(ctx, `
		INSERT INTO cores.users_roles (isu, role)
		VALUES ($1, $2);
	`, isu, role)

	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// unique violation (если есть UNIQUE на (isu, role))
		if pgErr.Code == "23505" {
			return fmt.Errorf("user already exists")
		}
		// foreign key violation (isu не существует в cores.users)
		if pgErr.Code == "23503" {
			return fmt.Errorf("user doesn't exist")
		}
	}

	return err
}

func (u *userRepository) HasFaceImages(ctx context.Context, isu string) (bool, error) {
	var exists bool
	err := u.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cores.face_images WHERE student_id = $1)`, isu).Scan(&exists)
	return exists, err
}

func (u *userRepository) DeleteFaceImages(ctx context.Context, isu string) error {
	_, err := u.db.Exec(ctx, `DELETE FROM cores.face_images WHERE student_id = $1`, isu)
	return err
}

func (u *userRepository) GetFaceImage(ctx context.Context, isu string, slot domain.FaceSlot) ([]byte, time.Time, error) {
	col := slot.Column()
	if col == "" {
		return nil, time.Time{}, fmt.Errorf("invalid face slot: %q", slot)
	}

	var (
		data      []byte
		updatedAt time.Time
	)
	// slot.Column() возвращает только whitelisted значения (см. domain.FaceSlot),
	// поэтому интерполяция в имя колонки безопасна.
	query := fmt.Sprintf(`SELECT %s, updated_at FROM cores.face_images WHERE student_id = $1`, col)
	err := u.db.QueryRow(ctx, query, isu).Scan(&data, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, time.Time{}, domain.ErrFaceImageNotFound
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	return data, updatedAt, nil
}

func (u *userRepository) GetFacesUpdatedAt(ctx context.Context, isu string) (time.Time, error) {
	var updatedAt time.Time
	err := u.db.QueryRow(ctx,
		`SELECT updated_at FROM cores.face_images WHERE student_id = $1`, isu,
	).Scan(&updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, domain.ErrFaceImageNotFound
	}
	return updatedAt, err
}

func (u *userRepository) GetRoles(ctx context.Context, isu string) ([]string, error) {
	isu = strings.TrimSpace(isu)
	if isu == "" {
		return nil, fmt.Errorf("isu is empty")
	}

	// проверка, что пользователь существует
	var exists int
	if err := u.db.QueryRow(ctx, `SELECT 1 FROM cores.users WHERE isu = $1;`, isu).Scan(&exists); err != nil {
		return nil, err
	}

	rows, err := u.db.Query(ctx, `
		SELECT role
		FROM cores.users_roles
		WHERE isu = $1;
	`, isu)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, strings.ToLower(role))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}
