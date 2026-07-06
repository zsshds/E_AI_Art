package handler

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
	"github.com/imagegen/backend/internal/service/task"
	"github.com/labstack/echo/v4"
)

type TaskHandler struct {
	repo               *repo.TaskRepo
	manager            *task.Manager
	projectRepo        *repo.ProjectRepo
	styleRepo          *repo.StyleProfileRepo
	downloadTimeoutSec int
}

func NewTaskHandler(repo *repo.TaskRepo, manager *task.Manager, projectRepo *repo.ProjectRepo, styleRepo *repo.StyleProfileRepo, downloadTimeoutSec int) *TaskHandler {
	if downloadTimeoutSec <= 0 {
		downloadTimeoutSec = 120
	}
	return &TaskHandler{repo: repo, manager: manager, projectRepo: projectRepo, styleRepo: styleRepo, downloadTimeoutSec: downloadTimeoutSec}
}

func (h *TaskHandler) Create(c echo.Context) error {
	type CreateTaskRequest struct {
		StyleProfileID string   `json:"style_profile_id"`
		UserInput      string   `json:"user_input"`
		Model          string   `json:"model"`
		Size           string   `json:"size"`
		APIQuality     string   `json:"api_quality"`
		SourceImage    string   `json:"source_image"`  // backward compat (single)
		SourceImages   []string `json:"source_images"` // new: multiple images
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

type TaskListResponse struct {
	Items      []model.Task `json:"items"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
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
	createdByFilter := strings.TrimSpace(c.QueryParam("created_by"))
	statusFilter := strings.TrimSpace(c.QueryParam("status"))
	page := parsePositiveInt(c.QueryParam("page"), repo.DefaultTaskListPage)
	pageSize := parsePositiveInt(c.QueryParam("page_size"), repo.DefaultTaskListPageSize)
	if pageSize > repo.MaxTaskListPageSize {
		pageSize = repo.MaxTaskListPageSize
	}
	ctx := c.Request().Context()

	filter := bson.M{}
	if createdByFilter != "" {
		filter["created_by"] = createdByFilter
	}
	if statusFilter != "" {
		if !isValidTaskStatus(statusFilter) {
			return fail(c, http.StatusBadRequest, "invalid status")
		}
		filter["status"] = statusFilter
	}

	// Admin sees all tasks
	if role == "admin" {
		tasks, total, err := h.repo.List(ctx, filter, page, pageSize)
		if err != nil {
			return fail(c, http.StatusInternalServerError, err.Error())
		}
		return ok(c, newTaskListResponse(tasks, total, page, pageSize))
	}

	// User: get their project IDs
	userProjects, _ := h.projectRepo.GetProjectsForUser(ctx, username)
	projectIDs := make([]string, 0, len(userProjects))
	for _, p := range userProjects {
		projectIDs = append(projectIDs, p.ID.Hex())
	}

	visibilityRules := []bson.M{{"created_by": username}}
	if len(projectIDs) > 0 {
		visibilityRules = append(visibilityRules, bson.M{"project_id": bson.M{"$in": projectIDs}})
	}
	filter["$or"] = visibilityRules

	tasks, total, err := h.repo.List(ctx, filter, page, pageSize)
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}

	return ok(c, newTaskListResponse(tasks, total, page, pageSize))
}

func summarizeTasksForList(tasks []model.Task) []model.Task {
	if tasks == nil {
		return []model.Task{}
	}

	summaries := make([]model.Task, len(tasks))
	copy(summaries, tasks)
	for i := range summaries {
		summaries[i].SourceImageURLs = nil
	}
	return summaries
}

func newTaskListResponse(tasks []model.Task, total int64, page, pageSize int) TaskListResponse {
	totalPages := 0
	if total > 0 && pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return TaskListResponse{
		Items:      summarizeTasksForList(tasks),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func isValidTaskStatus(status string) bool {
	switch model.TaskStatus(status) {
	case model.TaskStatusPending, model.TaskStatusProcessing, model.TaskStatusDone, model.TaskStatusFailed:
		return true
	default:
		return false
	}
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

	if strings.HasPrefix(t.ResultImageURL, "data:") {
		return h.downloadDataURL(c, id, t.ResultImageURL)
	}

	if !strings.HasPrefix(t.ResultImageURL, "http://") && !strings.HasPrefix(t.ResultImageURL, "https://") {
		return fail(c, http.StatusBadRequest, "unsupported image url")
	}

	client := &http.Client{Timeout: time.Duration(h.downloadTimeoutSec) * time.Second}
	resp, err := client.Get(t.ResultImageURL)
	if err != nil {
		return fail(c, http.StatusInternalServerError, fmt.Sprintf("fetch image: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fail(c, http.StatusInternalServerError, fmt.Sprintf("upstream image error %d", resp.StatusCode))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="E_AI_Art-%s.png"`, shortTaskID(id)))
	c.Response().WriteHeader(http.StatusOK)
	_, _ = io.Copy(c.Response(), resp.Body)
	return nil
}

func (h *TaskHandler) downloadDataURL(c echo.Context, id, dataURL string) error {
	comma := strings.Index(dataURL, ",")
	if comma <= len("data:") {
		return fail(c, http.StatusBadRequest, "invalid data url")
	}

	meta := dataURL[len("data:"):comma]
	payload := dataURL[comma+1:]
	if !strings.Contains(meta, ";base64") {
		return fail(c, http.StatusBadRequest, "unsupported data url encoding")
	}

	mimeType := strings.Split(meta, ";")[0]
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return fail(c, http.StatusBadRequest, "invalid data url payload")
		}
	}

	c.Response().Header().Set("Content-Type", mimeType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="E_AI_Art-%s.png"`, shortTaskID(id)))
	return c.Blob(http.StatusOK, mimeType, decoded)
}

func shortTaskID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
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
