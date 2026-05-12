package handler

import (
	"net/http"

	"github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/repo"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	userRepo *repo.UserRepo
}

func NewUserHandler(userRepo *repo.UserRepo) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

func (h *UserHandler) List(c echo.Context) error {
	users, err := h.userRepo.List(c.Request().Context())
	if err != nil {
		return fail(c, http.StatusInternalServerError, "failed to list users")
	}
	return ok(c, users)
}

func (h *UserHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	callerID := middleware.GetUserID(c)
	if id == callerID {
		return fail(c, http.StatusBadRequest, "cannot delete yourself")
	}
	if err := h.userRepo.DeleteByID(c.Request().Context(), id); err != nil {
		return fail(c, http.StatusInternalServerError, "failed to delete user")
	}
	return ok(c, nil)
}

func (h *UserHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.DELETE("/:id", h.Delete)
}
