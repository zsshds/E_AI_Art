package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
	"github.com/imagegen/backend/internal/service/task"
	"github.com/labstack/echo/v4"
)

type TaskHandler struct {
	repo    *repo.TaskRepo
	manager *task.Manager
}

func NewTaskHandler(repo *repo.TaskRepo, manager *task.Manager) *TaskHandler {
	return &TaskHandler{repo: repo, manager: manager}
}

func (h *TaskHandler) Create(c echo.Context) error {
	type CreateTaskRequest struct {
		StyleProfileID string `json:"style_profile_id"`
		UserInput      string `json:"user_input"`
		Model          string `json:"model"`
	}

	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	if req.StyleProfileID == "" || req.UserInput == "" {
		return fail(c, http.StatusBadRequest, "style_profile_id and user_input are required")
	}

	t := &model.Task{
		StyleProfileID: styleProfileIDFromHex(req.StyleProfileID),
		UserInput:      req.UserInput,
		Model:          req.Model,
		CreatedBy:      middleware.GetUsername(c),
	}

	if err := h.repo.Create(c.Request().Context(), t); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}

	if err := h.manager.PublishTask(c.Request().Context(), t.ID.Hex()); err != nil {
		return fail(c, http.StatusInternalServerError, "failed to enqueue task")
	}

	return ok(c, t)
}

func (h *TaskHandler) GetByID(c echo.Context) error {
	id := c.Param("id")
	t, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, "task not found")
	}
	return ok(c, t)
}

func (h *TaskHandler) List(c echo.Context) error {
	role := middleware.GetRole(c)
	username := middleware.GetUsername(c)

	// Admin sees all, user sees only their own
	var createdBy string
	if role == "user" {
		createdBy = username
	}

	tasks, err := h.repo.List(c.Request().Context(), createdBy)
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	if tasks == nil {
		tasks = []model.Task{}
	}
	return ok(c, tasks)
}

func (h *TaskHandler) Download(c echo.Context) error {
	id := c.Param("id")
	t, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, "task not found")
	}
	if t.ResultImageURL == "" {
		return fail(c, http.StatusNotFound, "no image available")
	}

	// Fetch the image from the platform
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(t.ResultImageURL)
	if err != nil {
		return fail(c, http.StatusInternalServerError, fmt.Sprintf("fetch image: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fail(c, http.StatusInternalServerError, fmt.Sprintf("upstream image error %d", resp.StatusCode))
	}

	c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="imagegen-%s.png"`, id[:8]))
	c.Response().WriteHeader(http.StatusOK)
	io.Copy(c.Response(), resp.Body)
	return nil
}

func (h *TaskHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.GetByID)
	g.GET("/:id/download", h.Download)
}

func styleProfileIDFromHex(id string) primitive.ObjectID {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID
	}
	return objID
}
