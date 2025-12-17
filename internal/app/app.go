package app

import (
	"context"
	"fmt"
	"monitoring_backend/internal/config"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	httpHandler "monitoring_backend/internal/http/handlers"
	httpRouter "monitoring_backend/internal/http/router"
)

type App struct {
	cfg    *config.Config
	db     *pgxpool.Pool
	server *http.Server
}

func New(cfg *config.Config, db *pgxpool.Pool) *App {
	h := httpHandler.New(db)
	r := httpRouter.New(cfg, h)

	return &App{
		cfg: cfg,
		db:  db,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
			Handler:      r,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
	}
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.server.Shutdown(shCtx)

	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
