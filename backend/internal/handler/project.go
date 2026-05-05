package handler

import (
	"net/http"

	"github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
	"github.com/labstack/echo/v4"
)

type ProjectHandler struct {
	repo *repo.ProjectRepo
}

func NewProjectHandler(repo *repo.ProjectRepo) *ProjectHandler {
	return &ProjectHandler{repo: repo}
}

func (h *ProjectHandler) List(c echo.Context) error {
	projects, err := h.repo.List(c.Request().Context())
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return ok(c, projects)
}

func (h *ProjectHandler) Create(c echo.Context) error {
	var req struct {
		Name    string   `json:"name"`
		Members []string `json:"members"`
	}
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return fail(c, http.StatusBadRequest, "name is required")
	}

	project := &model.Project{
		Name:      req.Name,
		CreatedBy: middleware.GetUsername(c),
		Members:   req.Members,
	}
	if project.Members == nil {
		project.Members = []string{}
	}

	if err := h.repo.Create(c.Request().Context(), project); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return ok(c, project)
}

func (h *ProjectHandler) Update(c echo.Context) error {
	id := c.Param("id")
	existing, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, "project not found")
	}

	var req struct {
		Name    *string  `json:"name"`
		Members []string `json:"members"`
	}
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Members != nil {
		existing.Members = req.Members
	}

	if err := h.repo.Update(c.Request().Context(), id, existing); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return ok(c, existing)
}

func (h *ProjectHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.repo.Delete(c.Request().Context(), id); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return ok(c, nil)
}

func (h *ProjectHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create, middleware.RequireAdmin())
	g.PUT("/:id", h.Update, middleware.RequireAdmin())
	g.DELETE("/:id", h.Delete, middleware.RequireAdmin())
}
