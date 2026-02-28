package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

type APIKeyConfig struct {
	Key string
}

// APIKeyAuth validates requests using a static API key.
// Accepts key via "Authorization: Bearer <key>" header or "X-API-Key: <key>" header.
func APIKeyAuth(cfg APIKeyConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			key := extractAPIKey(c)
			if key == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false,
					"error":   "missing api key: use Authorization: Bearer <key> or X-API-Key header",
				})
			}

			if !secureCompare(key, cfg.Key) {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false,
					"error":   "invalid api key",
				})
			}

			return next(c)
		}
	}
}

func extractAPIKey(c *echo.Context) string {
	if auth := c.Request().Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	if key := c.Request().Header.Get("X-API-Key"); key != "" {
		return key
	}

	return ""
}

// secureCompare prevents timing attacks
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
