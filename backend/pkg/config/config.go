// Package config loads runtime configuration from environment variables,
// following 12-factor conventions. It intentionally has no external
// dependencies (no YAML/dotenv parsing) — .env files are loaded by the
// process manager or shell in local dev; in staging/production real
// environment variables are injected by the platform.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config is the fully-resolved application configuration. One value is loaded at
// startup and passed (read-only) wherever settings are needed.
type Config struct {
	// Core
	Env         string
	Port        string
	DatabaseURL string
	RedisURL    string
	AllowedCORS string
	AppBaseURL  string // public origin of the frontend (reset links, OAuth redirects)

	// Sessions / cookies
	SessionCookieName     string
	SessionTTL            time.Duration
	SessionCookieSecure   bool
	SessionCookieSameSite string // "lax" | "strict" | "none"

	// Password hashing (Argon2id)
	Argon2MemoryKB uint32
	Argon2Time     uint32
	Argon2Threads  uint8

	// Two-factor auth — base64 32-byte AES-256-GCM key encrypting TOTP secrets.
	TOTPEncryptionKey string

	// One-time codes / tokens
	EmailCodeTTL  time.Duration
	ResetTokenTTL time.Duration

	// Email delivery
	Mailer       string // "console" | "resend"
	ResendAPIKey string
	EmailFrom    string

	// Google OAuth (blank = feature disabled)
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

// IsProduction reports whether the app is running in a production environment.
func (c Config) IsProduction() bool { return c.Env == "production" }

// GoogleEnabled reports whether Google OAuth is configured.
func (c Config) GoogleEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

// Load reads configuration from the environment, applying safe local-dev defaults.
func Load() Config {
	return Config{
		Env:         getenv("APP_ENV", "development"),
		Port:        getenv("PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://candella:candella@localhost:5433/candella?sslmode=disable"),
		RedisURL:    getenv("REDIS_URL", "redis://localhost:6381"),
		AllowedCORS: getenv("ALLOWED_CORS_ORIGIN", "http://localhost:3000"),
		AppBaseURL:  getenv("APP_BASE_URL", "http://localhost:3000"),

		SessionCookieName:     getenv("SESSION_COOKIE_NAME", "candella_session"),
		SessionTTL:            time.Duration(getenvInt("SESSION_TTL_HOURS", 720)) * time.Hour,
		SessionCookieSecure:   getenvBool("SESSION_COOKIE_SECURE", false),
		SessionCookieSameSite: getenv("SESSION_COOKIE_SAMESITE", "lax"),

		Argon2MemoryKB: uint32(getenvInt("ARGON2_MEMORY_KB", 65536)),
		Argon2Time:     uint32(getenvInt("ARGON2_TIME", 3)),
		Argon2Threads:  uint8(getenvInt("ARGON2_THREADS", 2)),

		TOTPEncryptionKey: getenv("TOTP_ENCRYPTION_KEY", ""),

		EmailCodeTTL:  time.Duration(getenvInt("EMAIL_CODE_TTL_MINUTES", 10)) * time.Minute,
		ResetTokenTTL: time.Duration(getenvInt("RESET_TOKEN_TTL_MINUTES", 30)) * time.Minute,

		Mailer:       getenv("MAILER", "console"),
		ResendAPIKey: getenv("RESEND_API_KEY", ""),
		EmailFrom:    getenv("EMAIL_FROM", "Candella <noreply@candella.app>"),

		GoogleClientID:     getenv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getenv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
