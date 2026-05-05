package handler

import (
	"encoding/json"
	"net/http"
	"strings"

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

// AvailableModel represents a model that the admin has configured as available.
type AvailableModel struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// GetAvailableModels returns the list of models configured by the admin.
func (h *SettingHandler) GetAvailableModels(c echo.Context) error {
	ctx := c.Request().Context()
	raw, err := h.repo.Get(ctx, "available_models")
	if err != nil || raw == "" {
		return ok(c, []AvailableModel{})
	}

	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return ok(c, []AvailableModel{})
	}

	result := make([]AvailableModel, 0, len(ids))
	for _, id := range ids {
		mt := model.GetModelType(id)
		// Banana models share the same UI as Gemini (aspect ratio dropdown)
		displayType := string(mt)
		if mt == model.ModelTypeBanana {
			displayType = "gemini"
		}
		result = append(result, AvailableModel{
			ID:    id,
			Type:  displayType,
			Label: formatModelLabel(id),
		})
	}
	return ok(c, result)
}

// formatModelLabel converts a model ID into a human-readable label.
func formatModelLabel(id string) string {
	// Known labels
	known := map[string]string{
		"gpt-4o-image":                "GPT-4o Image",
		"gpt-4o-image-vip":            "GPT-4o Image VIP",
		"gpt-image-2":                 "GPT Image 2",
		"gpt-image-1.5":               "GPT Image 1.5",
		"gemini-3-pro-image-preview":  "Gemini 3 Pro",
		"gemini-2.5-flash-image":      "Gemini 2.5 Flash",
		"nano-banana-pro":             "Nano Banana Pro",
		"nano-banana-2":               "Nano Banana 2",
		"nano-banana-2-2k":            "Nano Banana 2 (2K)",
		"nano-banana-2-4k":            "Nano Banana 2 (4K)",
		"mj_imagine":                  "Midjourney Imagine",
	}
	if label, ok := known[id]; ok {
		return label
	}
	// Auto-generate: replace hyphens/underscores with spaces, title case
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(id))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func (h *SettingHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Get)
	g.PUT("", h.Update, middleware.RequireAdmin())
	g.GET("/models", h.ListModels, middleware.RequireAdmin())
	g.GET("/models/available", h.GetAvailableModels)
}
