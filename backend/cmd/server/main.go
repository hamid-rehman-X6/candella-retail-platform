package main

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/health"
	appmiddleware "candella-ecosystem/backend/internal/middleware"
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

	// Per-process HMAC key for stateless mfa/oauth challenges. Regenerated on
	// restart (only invalidates in-flight <5-min challenges — acceptable).
	challengeKey := make([]byte, 32)
	if _, err := rand.Read(challengeKey); err != nil {
		log.Error("challenge key generation failed", "error", err)
		os.Exit(1)
	}

	mail := mailer.New(cfg.Mailer, cfg.ResendAPIKey, cfg.EmailFrom, log)

	authRepo := auth.NewPostgresRepository(pool)
	authSvc := auth.NewService(authRepo, mail, cfg, log, enc, challengeKey)
	authHandler := auth.NewHandler(authSvc, cfg)

	wsRepo := workspace.NewPostgresRepository(pool)
	wsSvc := workspace.NewService(wsRepo)
	wsHandler := workspace.NewHandler(wsSvc)

	// 20 requests/min/IP on sensitive, unauthenticated auth endpoints.
	authLimiter := appmiddleware.NewRateLimiter(20, time.Minute)

	// --- Router -----------------------------------------------------------
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AllowedCORS},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true, // required so the session cookie is sent/accepted
	}))

	r.Route("/api/v1", func(api chi.Router) {
		health.Routes(api)
		auth.Routes(api, authHandler, authLimiter.Middleware)
		workspace.Routes(api, wsHandler, authHandler.RequireAuth)
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
