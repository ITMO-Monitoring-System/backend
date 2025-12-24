// @title Monitoring Backend API
// @version 1.0
// @description Backend for lecture monitoring via RabbitMQ and WebSocket
// @BasePath /
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	db, err := pgxpool.New(ctx, cfg.Postgres.DSN())
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}
	defer db.Close()

	if db.Ping(ctx) != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.TTL)

	a := app.New(cfg, db, jwtManager)

	if err := a.Run(ctx); err != nil {
		log.Fatalf("app stopped with error: %v", err)
	}
}
