package handler

import (
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"

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
		CreatedBy      string `json:"created_by"`
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
		CreatedBy:      req.CreatedBy,
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
	createdBy := c.QueryParam("created_by")
	tasks, err := h.repo.List(c.Request().Context(), createdBy)
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	if tasks == nil {
		tasks = []model.Task{}
	}
	return ok(c, tasks)
}

func (h *TaskHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.GetByID)
}

func styleProfileIDFromHex(id string) primitive.ObjectID {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID
	}
	return objID
}
