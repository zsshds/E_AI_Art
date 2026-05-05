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
	repo        *repo.TaskRepo
	manager     *task.Manager
	projectRepo *repo.ProjectRepo
	styleRepo   *repo.StyleProfileRepo
}

func NewTaskHandler(repo *repo.TaskRepo, manager *task.Manager, projectRepo *repo.ProjectRepo, styleRepo *repo.StyleProfileRepo) *TaskHandler {
	return &TaskHandler{repo: repo, manager: manager, projectRepo: projectRepo, styleRepo: styleRepo}
}

func (h *TaskHandler) Create(c echo.Context) error {
	type CreateTaskRequest struct {
		StyleProfileID string `json:"style_profile_id"`
		UserInput      string `json:"user_input"`
		Model          string `json:"model"`
		Size           string `json:"size"`
		APIQuality     string `json:"api_quality"`
	}

	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	if req.StyleProfileID == "" || req.UserInput == "" {
		return fail(c, http.StatusBadRequest, "style_profile_id and user_input are required")
	}

	if req.Size == "" {
		req.Size = "auto"
	}
	if req.APIQuality == "" {
		req.APIQuality = "medium"
	}

	// Inherit project_id from style profile
	projectID := ""
	styleProfile, err := h.styleRepo.GetByID(c.Request().Context(), req.StyleProfileID)
	if err == nil && styleProfile != nil {
		projectID = styleProfile.ProjectID
	}

	t := &model.Task{
		StyleProfileID: styleProfileIDFromHex(req.StyleProfileID),
		UserInput:      req.UserInput,
		Model:          req.Model,
		Size:           req.Size,
		APIQuality:     req.APIQuality,
		ProjectID:      projectID,
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
	ctx := c.Request().Context()

	// Admin sees all tasks
	if role == "admin" {
		tasks, err := h.repo.List(ctx, "", nil)
		if err != nil {
			return fail(c, http.StatusInternalServerError, err.Error())
		}
		if tasks == nil {
			tasks = []model.Task{}
		}
		return ok(c, tasks)
	}

	// User: get their project IDs
	userProjects, _ := h.projectRepo.GetProjectsForUser(ctx, username)
	projectIDs := make([]string, 0, len(userProjects))
	for _, p := range userProjects {
		projectIDs = append(projectIDs, p.ID.Hex())
	}

	// Fetch tasks: user's own OR from their projects
	allTasks, err := h.repo.List(ctx, "", nil)
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}

	filtered := make([]model.Task, 0)
	for _, t := range allTasks {
		if t.CreatedBy == username {
			filtered = append(filtered, t)
			continue
		}
		if t.ProjectID != "" {
			for _, pid := range projectIDs {
				if t.ProjectID == pid {
					filtered = append(filtered, t)
					break
				}
			}
		}
	}

	if filtered == nil {
		filtered = []model.Task{}
	}
	return ok(c, filtered)
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
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="E_AI_Art-%s.png"`, id[:8]))
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
