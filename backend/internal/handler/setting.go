package handler

import (
	"net/http"

	"github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/repo"
	"github.com/labstack/echo/v4"
)

type SettingHandler struct {
	repo *repo.SettingRepo
}

func NewSettingHandler(repo *repo.SettingRepo) *SettingHandler {
	return &SettingHandler{repo: repo}
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
		"api_base_url": true,
		"api_key":      true,
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

func (h *SettingHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Get)
	g.PUT("", h.Update, middleware.RequireAdmin())
}
