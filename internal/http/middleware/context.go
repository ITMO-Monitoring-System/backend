package middleware

import "context"

func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxUserID).(string)
	return id, ok
}

func Role(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(ctxRole).(string)
	return role, ok
}
