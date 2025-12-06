package main

import (
	"log"
	"monitoring_backend/internal/config"
)

func main() {
	cfg, err := config.Load("config.toml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dsn := cfg.Postgres.DSN()
	if dsn == "" {
		log.Fatal("DB_DSN environment variable not set")
	}
}
