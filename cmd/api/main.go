package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring_backend/internal/app"
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

	a := app.New(cfg, db)

	if err := a.Run(ctx); err != nil {
		log.Fatalf("app stopped with error: %v", err)
	}
}
