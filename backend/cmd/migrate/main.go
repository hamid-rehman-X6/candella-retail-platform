// Command migrate applies (or rolls back) database migrations using goose.
//
// Usage:
//
//	go run ./cmd/migrate            # apply all pending migrations (up)
//	go run ./cmd/migrate up         # same
//	go run ./cmd/migrate down       # roll back the most recent migration
//	go run ./cmd/migrate status     # show applied / pending migrations
//	go run ./cmd/migrate reset      # roll everything back
//
// The database DSN comes from DATABASE_URL (see pkg/config).
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/config"
)

func main() {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	config.LoadDotEnv(".env")
	cfg := config.Load()

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("migrate: open db: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("migrate: set dialect: %v", err)
	}

	if err := goose.RunContext(context.Background(), command, db, "."); err != nil {
		log.Fatalf("migrate: %q failed: %v", command, err)
	}
}
