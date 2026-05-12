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
		StyleProfileID string   `json:"style_profile_id"`
		UserInput      string   `json:"user_input"`
		Model          string   `json:"model"`
		Size           string   `json:"size"`
		APIQuality     string   `json:"api_quality"`
		SourceImage    string   `json:"source_image"`    // backward compat (single)
		SourceImages   []string `json:"source_images"`   // new: multiple images
		ImageCount     int      `json:"image_count"`
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
	if req.ImageCount < 1 || req.ImageCount > 10 {
		req.ImageCount = 1
	}

	// Inherit project_id from style profile
	projectID := ""
	styleProfile, err := h.styleRepo.GetByID(c.Request().Context(), req.StyleProfileID)
	if err == nil && styleProfile != nil {
		projectID = styleProfile.ProjectID
	}

	// Merge backward-compat single source_image into the array
	sourceURLs := req.SourceImages
	if req.SourceImage != "" {
		sourceURLs = append(sourceURLs, req.SourceImage)
	}
	if sourceURLs == nil {
		sourceURLs = []string{}
	}

	t := &model.Task{
		StyleProfileID:  styleProfileIDFromHex(req.StyleProfileID),
		UserInput:       req.UserInput,
		Model:           req.Model,
		Size:            req.Size,
		APIQuality:      req.APIQuality,
		ProjectID:       projectID,
		CreatedBy:       middleware.GetUsername(c),
		SourceImageURLs: sourceURLs,
		ImageCount:      req.ImageCount,
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

// Chat creates a child task for iterative editing. The parent task's result image
// becomes the source image for the new task, forming a conversation chain.
func (h *TaskHandler) Chat(c echo.Context) error {
	parentID := c.Param("id")

	type ChatRequest struct {
		Message string `json:"message"`
	}
	var req ChatRequest
	if err := c.Bind(&req); err != nil || req.Message == "" {
		return fail(c, http.StatusBadRequest, "message is required")
	}

	ctx := c.Request().Context()

	parent, err := h.repo.GetByID(ctx, parentID)
	if err != nil {
		return fail(c, http.StatusNotFound, "parent task not found")
	}
	if parent.Status != model.TaskStatusDone {
		return fail(c, http.StatusConflict, "parent task is not completed yet")
	}
	if parent.ResultImageURL == "" {
		return fail(c, http.StatusConflict, "parent task has no result image")
	}

	hasActive, err := h.repo.HasActiveChild(ctx, parentID)
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	if hasActive {
		return fail(c, http.StatusConflict, "this task already has a pending edit in progress")
	}

	// Verify style profile still exists
	if _, err := h.styleRepo.GetByID(ctx, parent.StyleProfileID.Hex()); err != nil {
		return fail(c, http.StatusNotFound, "style profile not found")
	}

	child := &model.Task{
		StyleProfileID:  parent.StyleProfileID,
		UserInput:       req.Message,
		Model:           parent.Model,
		Size:            parent.Size,
		APIQuality:      parent.APIQuality,
		SourceImageURLs: []string{parent.ResultImageURL},
		ImageCount:      1,
		ParentTaskID:    parent.ID,
		ProjectID:       parent.ProjectID,
		CreatedBy:       middleware.GetUsername(c),
	}

	if err := h.repo.Create(ctx, child); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}

	if err := h.manager.PublishTask(ctx, child.ID.Hex()); err != nil {
		return fail(c, http.StatusInternalServerError, "failed to enqueue task")
	}

	return ok(c, child)
}

// GetConversation returns the full ancestor chain of a task ordered from root to leaf.
func (h *TaskHandler) GetConversation(c echo.Context) error {
	id := c.Param("id")
	chain, err := h.repo.GetConversation(c.Request().Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, "task not found")
	}
	return ok(c, chain)
}

func (h *TaskHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.GetByID)
	g.GET("/:id/download", h.Download)
	g.POST("/:id/chat", h.Chat)
	g.GET("/:id/conversation", h.GetConversation)
}

func styleProfileIDFromHex(id string) primitive.ObjectID {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID
	}
	return objID
}