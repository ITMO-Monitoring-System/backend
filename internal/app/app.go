package app

import (
	"context"
	"errors"
	"fmt"
	"monitoring_backend/internal/config"
	"monitoring_backend/internal/http/middleware"

	"monitoring_backend/internal/http/handlers/department"
	"monitoring_backend/internal/http/handlers/group"
	lecture2 "monitoring_backend/internal/http/handlers/lecture"
	"monitoring_backend/internal/http/handlers/practice"
	"monitoring_backend/internal/http/handlers/student_group"
	"monitoring_backend/internal/http/handlers/subject"
	"monitoring_backend/internal/http/handlers/user"
	"monitoring_backend/internal/lecture"
	"monitoring_backend/internal/repository/postgres"

	"monitoring_backend/internal/service"
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
	health := httpHandler.New(db)
	wsHub := ws.NewHub()
	lectureManager := lecture.NewManager(wsHub, cfg.Rabbit.AMPQURL)

	// repositories
	userRepo := postgres.NewUserRepository(db)
	deptRepo := postgres.NewDepartmentRepository(db)
	groupRepo := postgres.NewGroupRepository(db)
	sgRepo := postgres.NewStudentGroupRepository(db)
	subjRepo := postgres.NewSubjectRepository(db)
	lecRepo := postgres.NewLectureRepository(db)
	lecGroupRepo := postgres.NewLectureGroupRepository(db)
	pracRepo := postgres.NewPracticeRepository(db)
	pracGroupRepo := postgres.NewPracticeGroupRepository(db)

	// services
	userServ := service.NewUserService(userRepo)
	deptServ := service.NewDepartmentService(deptRepo)
	groupServ := service.NewGroupService(groupRepo)
	sgServ := service.NewStudentGroupService(sgRepo)
	subjServ := service.NewSubjectService(db, subjRepo)
	lecServ := service.NewLectureService(db, lecRepo, lecGroupRepo)
	pracServ := service.NewPracticeService(db, pracRepo, pracGroupRepo)

	// handlers
	userHandler := user.NewUserHandler(userServ)
	deptHandler := department.NewDepartmentHandler(deptServ)
	groupHandler := group.NewGroupHandler(groupServ)
	sgHandler := student_group.NewStudentGroupHandler(sgServ)
	subjHandler := subject.NewSubjectHandler(subjServ)
	lecHandler := lecture2.NewLectureHandler(lecServ)
	pracHandler := practice.NewPracticeHandler(pracServ)

	r := httpRouter.New(httpRouter.Dependencies{
		Health:         health,
		Department:     deptHandler,
		Group:          groupHandler,
		StudentGroup:   sgHandler,
		Subject:        subjHandler,
		Lecture:        lecHandler,
		Practice:       pracHandler,
		User:           userHandler,
		WsHub:          wsHub,
		LectureManager: lectureManager,
	})

	// создаём конфиг CORS
	corsMiddleware := middleware.NewCORS(middleware.CORSConfig{
		AllowedOrigins: []string{"*"}, // в продакшене укажи домены
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	handler := corsMiddleware(r)

	return &App{
		cfg: cfg,
		db:  db,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
			Handler:      handler,
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
