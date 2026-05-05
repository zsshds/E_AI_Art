package handler

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/imagegen/backend/internal/middleware"
	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo  *repo.UserRepo
	jwtSecret string
}

func NewAuthHandler(userRepo *repo.UserRepo, jwtSecret string) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Login(c echo.Context) error {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return fail(c, http.StatusBadRequest, "username and password are required")
	}

	user, err := h.userRepo.GetByUsername(c.Request().Context(), req.Username)
	if err != nil {
		return fail(c, http.StatusUnauthorized, "invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return fail(c, http.StatusUnauthorized, "invalid username or password")
	}

	token, err := h.generateToken(user)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "failed to generate token")
	}

	return ok(c, map[string]any{
		"token":    token,
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *AuthHandler) Signup(c echo.Context) error {
	type SignupRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return fail(c, http.StatusBadRequest, "username and password are required")
	}
	if len(req.Username) < 3 {
		return fail(c, http.StatusBadRequest, "username must be at least 3 characters")
	}
	if len(req.Password) < 6 {
		return fail(c, http.StatusBadRequest, "password must be at least 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "failed to hash password")
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         "user", // public signup always creates regular users
	}

	if err := h.userRepo.Create(c.Request().Context(), user); err != nil {
		return fail(c, http.StatusConflict, "username already exists")
	}

	token, err := h.generateToken(user)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "failed to generate token")
	}

	return ok(c, map[string]any{
		"token":    token,
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *AuthHandler) Register(c echo.Context) error {
	// Only admins can register new users
	if middleware.GetRole(c) != "admin" {
		return fail(c, http.StatusForbidden, "only admins can register new users")
	}

	type RegisterRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return fail(c, http.StatusBadRequest, "username and password are required")
	}
	if req.Role != "admin" && req.Role != "user" {
		return fail(c, http.StatusBadRequest, "role must be 'admin' or 'user'")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "failed to hash password")
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         req.Role,
	}

	if err := h.userRepo.Create(c.Request().Context(), user); err != nil {
		return fail(c, http.StatusConflict, "username already exists or failed to create user")
	}

	return ok(c, map[string]any{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *AuthHandler) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func (h *AuthHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/login", h.Login)
	g.POST("/signup", h.Signup)
	g.POST("/register", h.Register, middleware.JWTAuth(h.jwtSecret), middleware.RequireAdmin())
}
