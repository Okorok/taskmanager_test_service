package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskmanager/internal/config"
	"taskmanager/internal/http/analytics_handler"
	"taskmanager/internal/http/auth_handler"
	"taskmanager/internal/http/middleware"
	"taskmanager/internal/http/task_handler"
	"taskmanager/internal/http/team_handler"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/infrastructure/cache"
	"taskmanager/internal/repository"
	"taskmanager/internal/service"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	success = 0
	fail    = 1
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)

		return fail
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "mysql", cfg.DB.DSN)
	if err != nil {
		logger.Error("failed to connect db", "error", err)

		return fail
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close db", "error", err)
		}
	}()

	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("failed to close redis", "error", err)
		}
	}()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to ping redis", "error", err)

		return fail
	}

	jwtManager := infrastructure.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	hasher := infrastructure.NewBcryptHasher()
	unitOfWork := infrastructure.NewUnitOfWork(db)
	taskListCache := cache.NewTaskListCache(redisClient, cfg.Redis.TaskListTTL)

	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	memberRepo := repository.NewTeamMemberRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)
	commentRepo := repository.NewTaskCommentRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)

	authService := service.NewAuthService(userRepo, hasher, jwtManager)
	teamService := service.NewTeamService(unitOfWork, teamRepo, memberRepo, userRepo)
	taskService := service.NewTaskService(unitOfWork, taskRepo, historyRepo, memberRepo, taskListCache)
	commentService := service.NewCommentService(taskRepo, commentRepo, memberRepo)

	registerHandler := auth_handler.NewRegisterHandler(authService, logger)
	loginHandler := auth_handler.NewLoginHandler(authService, logger)
	createTeamHandler := team_handler.NewCreateTeamHandler(teamService, logger)
	listTeamsHandler := team_handler.NewListTeamsHandler(teamService, logger)
	inviteHandler := team_handler.NewInviteHandler(teamService, logger)
	createTaskHandler := task_handler.NewCreateTaskHandler(taskService, logger)
	listTasksHandler := task_handler.NewListTasksHandler(taskService, logger)
	updateTaskHandler := task_handler.NewUpdateTaskHandler(taskService, logger)
	historyHandler := task_handler.NewHistoryHandler(taskService, logger)
	commentHandler := task_handler.NewCommentHandler(commentService, commentService, logger)
	analyticsHandler := analytics_handler.NewAnalyticsHandler(analyticsRepo, logger)

	authMiddleware := middleware.NewAuthMiddleware(jwtManager)
	protected := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.Handle(h).ServeHTTP
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/register", registerHandler.Handle)
	mux.HandleFunc("POST /api/v1/login", loginHandler.Handle)

	mux.HandleFunc("POST /api/v1/teams", protected(createTeamHandler.Handle))
	mux.HandleFunc("GET /api/v1/teams", protected(listTeamsHandler.Handle))
	mux.HandleFunc("POST /api/v1/teams/{id}/invite", protected(func(w http.ResponseWriter, r *http.Request) {
		inviteHandler.Handle(w, r, r.PathValue("id"))
	}))

	mux.HandleFunc("POST /api/v1/tasks", protected(createTaskHandler.Handle))
	mux.HandleFunc("GET /api/v1/tasks", protected(listTasksHandler.Handle))
	mux.HandleFunc("PUT /api/v1/tasks/{id}", protected(func(w http.ResponseWriter, r *http.Request) {
		updateTaskHandler.Handle(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("GET /api/v1/tasks/{id}/history", protected(func(w http.ResponseWriter, r *http.Request) {
		historyHandler.Handle(w, r, r.PathValue("id"))
	}))

	mux.HandleFunc("POST /api/v1/tasks/{id}/comments", protected(func(w http.ResponseWriter, r *http.Request) {
		commentHandler.Add(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("GET /api/v1/tasks/{id}/comments", protected(func(w http.ResponseWriter, r *http.Request) {
		commentHandler.List(w, r, r.PathValue("id"))
	}))

	mux.HandleFunc("GET /api/v1/analytics/team-stats", protected(analyticsHandler.TeamStats))
	mux.HandleFunc("GET /api/v1/analytics/top-creators", protected(analyticsHandler.TopCreators))
	mux.HandleFunc("GET /api/v1/analytics/inconsistent-tasks", protected(analyticsHandler.InconsistentTasks))

	recoveryMiddleware := middleware.NewRecoveryMiddleware(logger)
	handler := middleware.Chain(
		mux,
		recoveryMiddleware.Handle,
	)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("running server", "addr", cfg.HTTP.Addr)

		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("got signal, shutting down", "signal", sig)
	case err := <-serverErrCh:
		logger.Error("server error", "error", err)

		return fail
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)

		return fail
	}

	logger.Info("server stopped")

	return success
}
