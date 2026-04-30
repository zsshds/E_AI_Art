package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/imagegen/backend/config"
	"github.com/imagegen/backend/internal/handler"
	"github.com/imagegen/backend/internal/repo"
	"github.com/imagegen/backend/internal/service/image"
	"github.com/imagegen/backend/internal/service/task"
	"github.com/imagegen/backend/internal/ws"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
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
			cfg.Worker.TimeoutSec = t
		}
	}
	if v := os.Getenv("TASK_MAX_RETRY"); v != "" {
		if r, err := strconv.Atoi(v); err == nil {
			cfg.Worker.MaxRetry = r
		}
	}

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
	styleProfileRepo := repo.NewStyleProfileRepo(db)
	taskRepo := repo.NewTaskRepo(db)

	// Services
	imageClient := image.NewClient(cfg.OpenAI.APIKey, cfg.Worker.TimeoutSec)
	taskManager, err := task.NewManager(cfg.RabbitMQ.URI, imageClient, cfg.Worker.Concurrency, cfg.Worker.MaxRetry)
	if err != nil {
		log.Fatalf("connect to rabbitmq: %v", err)
	}
	defer taskManager.Close()

	// WebSocket Hub
	hub := ws.NewHub()

	// Echo app
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	api := e.Group("/api/v1")

	styleHandler := handler.NewStyleProfileHandler(styleProfileRepo)
	styleHandler.RegisterRoutes(api.Group("/style-profiles"))

	taskHandler := handler.NewTaskHandler(taskRepo, taskManager)
	taskHandler.RegisterRoutes(api.Group("/tasks"))

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
