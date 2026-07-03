package handler

import (
	"net/http"

	"github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
	"github.com/imagegen/backend/internal/service/prompt"
	"github.com/labstack/echo/v4"
)

type StyleProfileHandler struct {
	repo        *repo.StyleProfileRepo
	projectRepo *repo.ProjectRepo
}

func NewStyleProfileHandler(repo *repo.StyleProfileRepo, projectRepo *repo.ProjectRepo) *StyleProfileHandler {
	return &StyleProfileHandler{repo: repo, projectRepo: projectRepo}
}

type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func ok(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, APIResponse{Code: 0, Message: "success", Data: data})
}

func fail(c echo.Context, code int, msg string) error {
	return c.JSON(code, APIResponse{Code: code, Message: msg, Data: nil})
}

func (h *StyleProfileHandler) Create(c echo.Context) error {
	var profile model.StyleProfile
	if err := c.Bind(&profile); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	if profile.Name == "" {
		return fail(c, http.StatusBadRequest, "name is required")
	}

	profile.CreatedBy = middleware.GetUsername(c)

	if err := h.repo.Create(c.Request().Context(), &profile); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}

	return ok(c, profile)
}

func (h *StyleProfileHandler) GetByID(c echo.Context) error {
	id := c.Param("id")
	profile, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, "style profile not found")
	}
	return ok(c, profile)
}

func (h *StyleProfileHandler) List(c echo.Context) error {
	role := middleware.GetRole(c)
	ctx := c.Request().Context()

	// Admin sees all profiles
	if role == "admin" {
		profiles, err := h.repo.List(ctx, "", nil)
		if err != nil {
			return fail(c, http.StatusInternalServerError, err.Error())
		}
		return ok(c, profiles)
	}

	// Return all profiles for all authenticated users (admin sees all above)
	profiles, err := h.repo.List(ctx, "", nil)
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return ok(c, profiles)
}

func (h *StyleProfileHandler) Update(c echo.Context) error {
	id := c.Param("id")
	existing, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, "style profile not found")
	}

	var updated model.StyleProfile
	if err := c.Bind(&updated); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	// Preserve immutable fields
	updated.ID = existing.ID
	updated.CreatedAt = existing.CreatedAt
	updated.Version = existing.Version + 1

	if err := h.repo.Update(c.Request().Context(), id, &updated); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}

	return ok(c, updated)
}

func (h *StyleProfileHandler) Preview(c echo.Context) error {
	id := c.Param("id")

	type PreviewRequest struct {
		UserInput string `json:"user_input"`
	}
	var req PreviewRequest
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	profile, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, "style profile not found")
	}

	builder := prompt.NewPromptBuilder(profile)
	apiReq := builder.BuildAPIRequest(req.UserInput)

	// Return the request that would be sent to the API (preview mode)
	return ok(c, apiReq)
}

func (h *StyleProfileHandler) RegisterRoutes(g *echo.Group) {
	// Admin-only operations
	g.POST("", h.Create, middleware.RequireAdmin())
	g.PUT("/:id", h.Update, middleware.RequireAdmin())

	// Authenticated operations (any role)
	g.GET("", h.List)
	g.GET("/:id", h.GetByID)
	g.POST("/:id/preview", h.Preview)
}