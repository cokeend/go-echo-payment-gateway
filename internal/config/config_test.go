package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRequiredEnvs(t *testing.T) {
	t.Helper()
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_xxx")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("API_KEY", "test-api-key")
}

func TestLoad_Defaults(t *testing.T) {
	setRequiredEnvs(t)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "sk_test_xxx", cfg.StripeSecretKey)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURL)
	assert.Equal(t, "test-api-key", cfg.APIKey)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "redis://localhost:6379/0", cfg.RedisURL)
}

func TestLoad_CustomValues(t *testing.T) {
	setRequiredEnvs(t)
	t.Setenv("PORT", "3000")
	t.Setenv("APP_ENV", "production")
	t.Setenv("REDIS_URL", "redis://redis:6379/1")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "3000", cfg.Port)
	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "redis://redis:6379/1", cfg.RedisURL)
}

func TestLoad_MissingStripeKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("API_KEY", "key")
	os.Unsetenv("STRIPE_SECRET_KEY")

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "STRIPE_SECRET_KEY is required")
}

func TestLoad_MissingAPIKey(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_xxx")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Unsetenv("API_KEY")

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API_KEY is required")
}

func TestIsDevelopment(t *testing.T) {
	cfg := &Config{AppEnv: "development"}
	assert.True(t, cfg.IsDevelopment())
	assert.False(t, cfg.IsProduction())
}

func TestIsProduction(t *testing.T) {
	cfg := &Config{AppEnv: "production"}
	assert.True(t, cfg.IsProduction())
	assert.False(t, cfg.IsDevelopment())
}

func TestGetEnv_Fallback(t *testing.T) {
	os.Unsetenv("TEST_NONEXISTENT_VAR")
	assert.Equal(t, "default_val", getEnv("TEST_NONEXISTENT_VAR", "default_val"))
}

func TestGetEnv_EnvSet(t *testing.T) {
	t.Setenv("TEST_VAR_SET", "real_value")
	assert.Equal(t, "real_value", getEnv("TEST_VAR_SET", "fallback"))
}
