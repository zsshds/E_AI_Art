package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/imagegen/backend/config"
	"github.com/imagegen/backend/internal/handler"
	authmw "github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
	"github.com/imagegen/backend/internal/service/image"
	"github.com/imagegen/backend/internal/service/task"
	"github.com/imagegen/backend/internal/ws"
)

func main() {
	// Resolve config path: try executable dir > cwd > cwd/backend
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "config", "config.yaml"))
	}
	candidates = append(candidates,
		"config/config.yaml",         // from backend/ dir
		"backend/config/config.yaml", // from project root
	)

	var cfg *config.Config
	for _, p := range candidates {
		if c, err := config.Load(p); err == nil {
			cfg = c
			break
		}
	}
	if cfg == nil {
		log.Fatalf("load config: config.yaml not found in any of %v", candidates)
	}

	// Override with env vars
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("WORKER_CONCURRENCY"); v != "" {
		if c, err := strconv.Atoi(v); err == nil {
			cfg.Worker.Concurrency = c
		}
	}
	if v := os.Getenv("IMAGE_TASK_TIMEOUT_SECONDS"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.Worker.TaskTimeoutSec = t
			cfg.Worker.TimeoutSec = t
		}
	}
	if v := os.Getenv("IMAGE_REQUEST_TIMEOUT_SECONDS"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.Worker.ImageRequestTimeoutSec = t
		}
	}
	if v := os.Getenv("TASK_DOWNLOAD_TIMEOUT_SECONDS"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.Worker.DownloadTimeoutSec = t
		}
	}
	if v := os.Getenv("TASK_MAX_RETRY"); v != "" {
		if r, err := strconv.Atoi(v); err == nil {
			cfg.Worker.MaxRetry = r
		}
	}

	if cfg.Worker.TaskTimeoutSec <= 0 {
		cfg.Worker.TaskTimeoutSec = 300
	}
	if cfg.Worker.ImageRequestTimeoutSec <= 0 {
		cfg.Worker.ImageRequestTimeoutSec = cfg.Worker.TaskTimeoutSec
	}
	if cfg.Worker.DownloadTimeoutSec <= 0 {
		cfg.Worker.DownloadTimeoutSec = 120
	}
	if cfg.Worker.ImageRequestTimeoutSec > cfg.Worker.TaskTimeoutSec {
		cfg.Worker.TaskTimeoutSec = cfg.Worker.ImageRequestTimeoutSec
	}
	cfg.Worker.TimeoutSec = cfg.Worker.TaskTimeoutSec

	// MongoDB
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		log.Fatalf("connect to mongodb: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	if err := mongoClient.Ping(mongoCtx, nil); err != nil {
		log.Fatalf("ping mongodb: %v", err)
	}
	log.Println("connected to MongoDB")

	db := mongoClient.Database(cfg.MongoDB.Database)

	// Repos
	userRepo := repo.NewUserRepo(db)
	styleProfileRepo := repo.NewStyleProfileRepo(db)
	taskRepo := repo.NewTaskRepo(db)
	settingRepo := repo.NewSettingRepo(db)
	projectRepo := repo.NewProjectRepo(db)

	// Seed default admin
	seedAdmin(userRepo, cfg.Auth.AdminUser, cfg.Auth.AdminPass)

	// Seed default settings if DB is empty
	if _, err := settingRepo.Get(context.Background(), "api_base_url"); err != nil {
		settingRepo.Set(context.Background(), "api_base_url", cfg.OpenAI.BaseURL)
	}
	if _, err := settingRepo.Get(context.Background(), "api_key"); err != nil {
		settingRepo.Set(context.Background(), "api_key", cfg.OpenAI.APIKey)
	}
	if _, err := settingRepo.Get(context.Background(), "api_generation_path"); err != nil {
		settingRepo.Set(context.Background(), "api_generation_path", "/v1/images/generations/tasks")
	}
	if _, err := settingRepo.Get(context.Background(), "api_poll_path"); err != nil {
		settingRepo.Set(context.Background(), "api_poll_path", "/v1/images/tasks/")
	}
	if _, err := settingRepo.Get(context.Background(), "api_chat_path"); err != nil {
		settingRepo.Set(context.Background(), "api_chat_path", "/v1/chat/completions")
	}
	if _, err := settingRepo.Get(context.Background(), "api_generation_url"); err != nil {
		settingRepo.Set(context.Background(), "api_generation_url", "")
	}
	if _, err := settingRepo.Get(context.Background(), "api_poll_url"); err != nil {
		settingRepo.Set(context.Background(), "api_poll_url", "")
	}
	if _, err := settingRepo.Get(context.Background(), "api_chat_url"); err != nil {
		settingRepo.Set(context.Background(), "api_chat_url", "")
	}
	if _, err := settingRepo.Get(context.Background(), "api_banana_generation_url"); err != nil {
		settingRepo.Set(context.Background(), "api_banana_generation_url", "")
	}
	if _, err := settingRepo.Get(context.Background(), "model_fetch_url"); err != nil {
		settingRepo.Set(context.Background(), "model_fetch_url", "/v1/models")
	}
	if _, err := settingRepo.Get(context.Background(), "model_filter_type"); err != nil {
		settingRepo.Set(context.Background(), "model_filter_type", "image")
	}

	// Services
	imageClient := image.NewClient(cfg.OpenAI.APIKey, cfg.OpenAI.BaseURL, settingRepo, cfg.Worker.ImageRequestTimeoutSec)

	// WebSocket Hub
	hub := ws.NewHub()

	taskManager, err := task.NewManager(
		cfg.RabbitMQ.URI,
		imageClient,
		taskRepo,
		styleProfileRepo,
		hub,
		cfg.Worker.Concurrency,
		cfg.Worker.MaxRetry,
		cfg.Worker.TaskTimeoutSec,
	)
	if err != nil {
		log.Fatalf("connect to rabbitmq: %v", err)
	}
	defer taskManager.Close()

	// Start worker pool (context for graceful shutdown)
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	taskManager.StartWorkerPool(workerCtx)

	// Echo app
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
	}))

	// Auth routes (public: login, protected: register)
	authGroup := e.Group("/api/v1/auth")
	authHandler := handler.NewAuthHandler(userRepo, cfg.Auth.JWTSecret)
	authHandler.RegisterRoutes(authGroup)

	// Protected API routes
	api := e.Group("/api/v1")
	api.Use(authmw.JWTAuth(cfg.Auth.JWTSecret))

	styleHandler := handler.NewStyleProfileHandler(styleProfileRepo, projectRepo)
	styleHandler.RegisterRoutes(api.Group("/style-profiles"))

	taskHandler := handler.NewTaskHandler(taskRepo, taskManager, projectRepo, styleProfileRepo, cfg.Worker.DownloadTimeoutSec)
	taskHandler.RegisterRoutes(api.Group("/tasks"))

	settingHandler := handler.NewSettingHandler(settingRepo, imageClient)
	settingHandler.RegisterRoutes(api.Group("/settings"))

	projectHandler := handler.NewProjectHandler(projectRepo)
	projectHandler.RegisterRoutes(api.Group("/projects"))

	userHandler := handler.NewUserHandler(userRepo)
	userHandler.RegisterRoutes(api.Group("/users", authmw.RequireAdmin()))

	// WebSocket endpoint
	e.GET("/ws/tasks/:id", func(c echo.Context) error {
		taskID := c.Param("id")
		hub.HandleWebSocket(c.Response(), c.Request(), taskID)
		return nil
	})

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("shutting down server...")
		workerCancel()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		e.Shutdown(ctx)
	}()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("server starting on %s", addr)
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func seedAdmin(userRepo *repo.UserRepo, username, password string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := userRepo.GetByUsername(ctx, username)
	if err == nil {
		log.Printf("admin user '%s' already exists, skipping seed", username)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("seed admin: hash password error: %v", err)
		return
	}

	admin := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         "admin",
	}

	if err := userRepo.Create(ctx, admin); err != nil {
		log.Printf("seed admin: create error: %v", err)
		return
	}

	log.Printf("default admin created: %s / %s (change in production!)", username, password)
}
