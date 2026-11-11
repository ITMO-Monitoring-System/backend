// @title Monitoring Backend API
// @version 1.0
// @description Backend for lecture monitoring via RabbitMQ and WebSocket
// @BasePath /
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "monitoring_backend/docs"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/app"
	"monitoring_backend/internal/auth"
	"monitoring_backend/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("config.toml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Logger.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.String("timestamp", attr.Value.Time().UTC().Format(time.RFC3339Nano))
			}
			if attr.Key == slog.LevelKey {
				return slog.String("level", strings.ToLower(attr.Value.String()))
			}
			return attr
		},
	})))

	poolCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN())
	if err != nil {
		log.Fatalf("failed to parse postgres dsn: %T", err)
	}
	if cfg.Postgres.MaxConns > 0 {
		poolCfg.MaxConns = cfg.Postgres.MaxConns
	}
	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("failed to connect postgres: %T", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping postgres: %T", err)
	}

	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.TTL)

	a := app.New(cfg, db, jwtManager)

	if err := a.Run(ctx); err != nil {
		log.Fatalf("app stopped with error: %v", err)
	}
}
