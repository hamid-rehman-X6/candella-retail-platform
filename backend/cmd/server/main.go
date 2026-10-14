package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/catalog"
	"candella-ecosystem/backend/internal/health"
	appmiddleware "candella-ecosystem/backend/internal/middleware"
	"candella-ecosystem/backend/internal/platform"
	"candella-ecosystem/backend/internal/sales"
	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/internal/workspace"
	"candella-ecosystem/backend/pkg/config"
	"candella-ecosystem/backend/pkg/crypto"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/logger"
	"candella-ecosystem/backend/pkg/mailer"
)

func main() {
	// Load backend/.env (if present) before reading config, so local dev "just
	// works" without exporting env vars by hand. Real env vars still take priority.
	config.LoadDotEnv(".env")

	cfg := config.Load()
	log := logger.New(cfg.Env)

	rootCtx := context.Background()

	// --- Database ---------------------------------------------------------
	pool, err := database.Connect(rootCtx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("database connected")

	// --- Auth dependencies ------------------------------------------------
	// TOTP secret encryptor — optional; 2FA setup is rejected when unset.
	var enc *crypto.Encryptor
	if cfg.TOTPEncryptionKey != "" {
		enc, err = crypto.NewEncryptor(cfg.TOTPEncryptionKey)
		if err != nil {
			log.Warn("invalid TOTP_ENCRYPTION_KEY — 2FA disabled", "error", err)
			enc = nil
		}
	}

	challengeKey, err := resolveChallengeKey(cfg, log)
	if err != nil {
		log.Error("challenge key unavailable", "error", err)
		os.Exit(1)
	}

	mail := mailer.New(cfg.Mailer, cfg.ResendAPIKey, cfg.EmailFrom, log)

	// The single writer to audit_logs, shared by every slice.
	auditor := audit.NewRecorder(pool, log)

	authRepo := auth.NewPostgresRepository(pool)
	authSvc := auth.NewService(authRepo, mail, auditor, cfg, log, enc, challengeKey)
	authHandler := auth.NewHandler(authSvc, cfg)

	wsRepo := workspace.NewPostgresRepository(pool)
	wsSvc := workspace.NewService(wsRepo)
	wsHandler := workspace.NewHandler(wsSvc)
	staffHandler := workspace.NewStaffHandler(workspace.NewStaffService(
		workspace.NewStaffPostgresRepository(pool), mail, cfg.AppBaseURL), auditor)
	catalogHandler := catalog.NewHandler(catalog.NewService(catalog.NewPostgresRepository(pool)), auditor, log)
	importHandler := catalog.NewImportHandler(catalog.NewImportService(catalog.NewImportPostgresRepository(pool)), auditor, log)
	storeHandler := store.NewHandler(pool, auditor, log)
	salesHandler := sales.NewHandler(sales.NewService(sales.NewPostgresRepository(pool)), auditor, log)

	// The platform console reads across every tenant. It will move to its own
	// pool running as a dedicated DB role once RLS lands (see the architecture
	// plan §3.2); the seam is here so that's a wiring change, not a refactor.
	platformSvc := platform.NewService(platform.NewPostgresRepository(pool))
	platformHandler := platform.NewHandler(platformSvc, auditor, log)

	// 20 requests/min/IP on sensitive, unauthenticated auth endpoints.
	authLimiter := appmiddleware.NewRateLimiter(20, time.Minute)

	// --- Router -----------------------------------------------------------
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	// After RealIP, so audited IPs reflect any trusted proxy headers.
	r.Use(audit.Middleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AllowedCORS},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", workspace.TenantHeader},
		AllowCredentials: true, // required so the session cookie is sent/accepted
	}))

	r.Route("/api/v1", func(api chi.Router) {
		health.Routes(api)
		auth.Routes(api, authHandler, authLimiter.Middleware)
		workspace.Routes(api, wsHandler, authHandler.RequireAuth)
		workspace.StaffRoutes(api, staffHandler, authHandler.RequireAuth, wsHandler.RequireTenant)
		catalog.Routes(api, catalogHandler, authHandler.RequireAuth, wsHandler.RequireTenant)
		catalog.ImportRoutes(api, importHandler, authHandler.RequireAuth, wsHandler.RequireTenant)
		store.Routes(api, storeHandler, authHandler.RequireAuth, wsHandler.RequireTenant)
		sales.Routes(api, salesHandler, authHandler.RequireAuth, wsHandler.RequireTenant)
		platform.Routes(api, platformHandler, authHandler.RequireAuth)
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("shutdown complete")
}

// resolveChallengeKey returns the HMAC key used to sign the stateless challenges
// issued during sign-in: the 2FA hand-off token and the OAuth state parameter.
//
// The key must be stable across restarts and identical on every instance. A
// random per-process key silently breaks two things: a sign-in already in
// flight fails after a redeploy (the callback can't verify its own state), and
// running more than one instance is impossible, because instance B can't verify
// a challenge signed by instance A. Production therefore requires it to be set.
func resolveChallengeKey(cfg config.Config, log *slog.Logger) ([]byte, error) {
	if cfg.ChallengeSigningKey != "" {
		key, err := base64.StdEncoding.DecodeString(cfg.ChallengeSigningKey)
		if err != nil {
			return nil, fmt.Errorf("CHALLENGE_SIGNING_KEY is not valid base64: %w", err)
		}
		if len(key) < 32 {
			return nil, fmt.Errorf("CHALLENGE_SIGNING_KEY must decode to at least 32 bytes, got %d", len(key))
		}
		return key, nil
	}

	if cfg.IsProduction() {
		return nil, errors.New("CHALLENGE_SIGNING_KEY is required in production")
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating a fallback challenge key: %w", err)
	}
	log.Warn("CHALLENGE_SIGNING_KEY is not set — using a random key for this process; " +
		"sign-ins started before a restart will fail. Set one in backend/.env.")
	return key, nil
}
