package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL         string
	StripeSecretKey     string
	StripeWebhookSecret string
	RedisURL            string
	APIKey              string
	Port                string
	AppEnv              string
}

// Load reads configuration from .env file (if exists) then environment variables.
// Environment variables always take precedence over .env values.
func Load() (*Config, error) {
	loadEnvFile(".env")

	cfg := &Config{
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/payments?sslmode=disable"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379/0"),
		APIKey:              getEnv("API_KEY", ""),
		Port:                getEnv("PORT", "8080"),
		AppEnv:              getEnv("APP_ENV", "development"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func (c *Config) validate() error {
	if c.StripeSecretKey == "" {
		return fmt.Errorf("STRIPE_SECRET_KEY is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// loadEnvFile parses a .env file and sets values into the process environment.
// Existing env vars are NOT overwritten.
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)

		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
