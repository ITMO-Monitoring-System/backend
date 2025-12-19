package app

import (
	"context"
	"errors"
	"fmt"
	"monitoring_backend/internal/config"
	"monitoring_backend/internal/lecture"
	"monitoring_backend/internal/ws"
	"net/http"
	"time"

	httpHandler "monitoring_backend/internal/http/handlers"
	httpRouter "monitoring_backend/internal/http/router"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	cfg    *config.Config
	db     *pgxpool.Pool
	server *http.Server
}

func New(cfg *config.Config, db *pgxpool.Pool) *App {
	h := httpHandler.New(db)
	wsHub := ws.NewHub()
	lectureManager := lecture.NewManager(wsHub, cfg.Rabbit.AMPQURL)

	r := httpRouter.New(h, wsHub, lectureManager)

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
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
