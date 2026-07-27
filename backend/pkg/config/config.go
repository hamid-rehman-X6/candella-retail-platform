// Package config loads runtime configuration from environment variables,
// following 12-factor conventions. It intentionally has no external
// dependencies (no YAML/dotenv parsing) — .env files are loaded by the
// process manager or shell in local dev; in staging/production real
// environment variables are injected by the platform.
package config

import "os"

type Config struct {
	Env         string
	Port        string
	DatabaseURL string
	RedisURL    string
	AllowedCORS string
}

func Load() Config {
	return Config{
		Env:         getenv("APP_ENV", "development"),
		Port:        getenv("PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://candella:candella@localhost:5433/candella?sslmode=disable"),
		RedisURL:    getenv("REDIS_URL", "redis://localhost:6381"),
		AllowedCORS: getenv("ALLOWED_CORS_ORIGIN", "http://localhost:3000"),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
