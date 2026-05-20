package handlers

import (
	"errors"
	"fmt"
	"monitoring_backend/internal/domain"
	"monitoring_backend/internal/http/response"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ClientIP извлекает IP-адрес клиента из запроса. Учитывает заголовок
// X-Forwarded-For (актуально за reverse-proxy / nginx).
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func PathInt64(r *http.Request, key string, vars map[string]string) (int64, error) {
	raw, ok := vars[key]
	if !ok || raw == "" {
		return 0, fmt.Errorf("missing path param: %s", key)
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid path param %s", key)
	}
	return v, nil
}

func PathString(key string, vars map[string]string) (string, error) {
	raw, ok := vars[key]
	if !ok || raw == "" {
		return "", fmt.Errorf("missing path param: %s", key)
	}
	return raw, nil
}

func QueryInt(r *http.Request, key string, def int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid query param %s", key)
	}
	return v, nil
}

func QueryTimeRFC3339(r *http.Request, key string) (time.Time, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return time.Time{}, fmt.Errorf("missing query param: %s", key)
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time query param %s (expected RFC3339)", key)
	}
	return t, nil
}

func WriteServiceError(w http.ResponseWriter, err error) {
	// 404
	if errors.Is(err, pgx.ErrNoRows) ||
		errors.Is(err, domain.ErrorDepartmentNotFound) ||
		errors.Is(err, domain.ErrorDepartmentsNotFound) ||
		errors.Is(err, domain.ErrGroupNotFound) ||
		errors.Is(err, domain.ErrGroupsNotFound) {
		response.WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	// 409 (unique_violation)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		response.WriteError(w, http.StatusConflict, "conflict")
		return
	}

	// 404 (foreign_key_violation)
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		msg := "related entity not found"
		switch pgErr.ConstraintName {
		case "students_groups_user_id_fkey":
			msg = "student not found"
		case "students_groups_group_code_fkey":
			msg = "group not found"
		}
		response.WriteError(w, http.StatusNotFound, msg)
		return
	}

	response.WriteError(w, http.StatusInternalServerError, err.Error())
}
