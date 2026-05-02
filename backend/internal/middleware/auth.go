package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type contextKey string

const (
	ContextUserID   contextKey = "userID"
	ContextUsername contextKey = "username"
	ContextRole     contextKey = "role"
)

func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"code":    401,
					"message": "missing authorization header",
					"data":    nil,
				})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"code":    401,
					"message": "invalid authorization format",
					"data":    nil,
				})
			}

			token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"code":    401,
					"message": "invalid or expired token",
					"data":    nil,
				})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"code":    401,
					"message": "invalid token claims",
					"data":    nil,
				})
			}

			c.Set(string(ContextUserID), claims["user_id"])
			c.Set(string(ContextUsername), claims["username"])
			c.Set(string(ContextRole), claims["role"])

			return next(c)
		}
	}
}

func RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := c.Get(string(ContextRole))
			if role != "admin" {
				return c.JSON(http.StatusForbidden, map[string]any{
					"code":    403,
					"message": "admin access required",
					"data":    nil,
				})
			}
			return next(c)
		}
	}
}

func GetUserID(c echo.Context) string {
	id, _ := c.Get(string(ContextUserID)).(string)
	return id
}

func GetUsername(c echo.Context) string {
	name, _ := c.Get(string(ContextUsername)).(string)
	return name
}

func GetRole(c echo.Context) string {
	role, _ := c.Get(string(ContextRole)).(string)
	return role
}
