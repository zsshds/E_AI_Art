package handler

import (
	"net/http"

	"github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
	"github.com/imagegen/backend/internal/service/image"
	"github.com/labstack/echo/v4"
)

type SettingHandler struct {
	repo        *repo.SettingRepo
	imageClient *image.Client
}

func NewSettingHandler(repo *repo.SettingRepo, imageClient *image.Client) *SettingHandler {
	return &SettingHandler{repo: repo, imageClient: imageClient}
}

func (h *SettingHandler) Get(c echo.Context) error {
	settings, err := h.repo.GetAll(c.Request().Context())
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	if settings == nil {
		settings = map[string]string{}
	}
	return ok(c, settings)
}

func (h *SettingHandler) Update(c echo.Context) error {
	var req map[string]string
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	allowedKeys := map[string]bool{
		"api_base_url":        true,
		"api_key":             true,
		"api_generation_path": true,
		"api_poll_path":       true,
		"available_models":    true,
		"model_fetch_url":     true,
		"model_filter_type":   true,
	}

	for key, value := range req {
		if !allowedKeys[key] {
			return fail(c, http.StatusBadRequest, "unknown setting: "+key)
		}
		if err := h.repo.Set(c.Request().Context(), key, value); err != nil {
			return fail(c, http.StatusInternalServerError, err.Error())
		}
	}

	// Return updated settings
	return h.Get(c)
}

func (h *SettingHandler) ListModels(c echo.Context) error {
	if h.imageClient == nil {
		return fail(c, http.StatusServiceUnavailable, "image client not configured")
	}

	ctx := c.Request().Context()

	// Read the configured fetch URL from settings (default: /v1/models)
	fetchURL := "/v1/models"
	if v, err := h.repo.Get(ctx, "model_fetch_url"); err == nil && v != "" {
		fetchURL = v
	}

	// Read filter type: query param takes priority, fall back to DB setting, default "image"
	filterType := c.QueryParam("filter")
	if filterType == "" {
		filterType = "image"
		if v, err := h.repo.Get(ctx, "model_filter_type"); err == nil && v != "" {
			filterType = v
		}
	}

	result, err := h.imageClient.ListModels(ctx, fetchURL)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "failed to fetch models: "+err.Error())
	}

	// Apply filter based on configured filter type
	if filterType == "image" && result != nil {
		filtered := make([]image.ModelInfo, 0, len(result.Data))
		for _, m := range result.Data {
			if model.IsImageModel(m.ID) {
				filtered = append(filtered, m)
			}
		}
		result.Data = filtered
	}

	return ok(c, result)
}

func (h *SettingHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Get)
	g.PUT("", h.Update, middleware.RequireAdmin())
	g.GET("/models", h.ListModels, middleware.RequireAdmin())
}
