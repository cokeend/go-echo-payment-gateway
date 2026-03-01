package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

const testAPIKey = "test-secret-key-123"

func setupEcho(authMiddleware echo.MiddlewareFunc) *echo.Echo {
	e := echo.New()
	e.GET("/protected", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}, authMiddleware)
	return e
}

func TestAPIKeyAuth_ValidBearerToken(t *testing.T) {
	auth := APIKeyAuth(APIKeyConfig{Key: testAPIKey})
	e := setupEcho(auth)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyAuth_ValidXAPIKey(t *testing.T) {
	auth := APIKeyAuth(APIKeyConfig{Key: testAPIKey})
	e := setupEcho(auth)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-API-Key", testAPIKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	auth := APIKeyAuth(APIKeyConfig{Key: testAPIKey})
	e := setupEcho(auth)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing api key")
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	auth := APIKeyAuth(APIKeyConfig{Key: testAPIKey})
	e := setupEcho(auth)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid api key")
}

func TestAPIKeyAuth_BearerPrefixOnly(t *testing.T) {
	auth := APIKeyAuth(APIKeyConfig{Key: testAPIKey})
	e := setupEcho(auth)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIKeyAuth_NonBearerAuth(t *testing.T) {
	auth := APIKeyAuth(APIKeyConfig{Key: testAPIKey})
	e := setupEcho(auth)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSecureCompare(t *testing.T) {
	assert.True(t, secureCompare("abc", "abc"))
	assert.False(t, secureCompare("abc", "def"))
	assert.False(t, secureCompare("abc", "abcd"))
	assert.False(t, secureCompare("", "abc"))
}
