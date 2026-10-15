// Command admin manages platform console access from the server itself.
//
// Granting staff access is deliberately *not* an HTTP endpoint. A "make me an
// admin" route is a permanent liability however well it's guarded — one auth
// bug and it hands over the whole platform. Requiring shell access to the
// server means the bootstrap is protected by your infrastructure, not by
// application code.
//
// Usage:
//
//	go run ./cmd/admin list
//	go run ./cmd/admin grant  you@example.com owner
//	go run ./cmd/admin revoke you@example.com
//
// Roles: owner | support | billing | engineering
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"candella-ecosystem/backend/internal/platform"
	"candella-ecosystem/backend/pkg/config"
	"candella-ecosystem/backend/pkg/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "admin:", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		return errors.New("no command given")
	}

	config.LoadDotEnv(".env")
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Granting platform access is an operator action, so it runs with the
	// login's own privileges rather than either API role.
	pool, err := database.Connect(ctx, cfg.DatabaseURL, "")
	if err != nil {
		return fmt.Errorf("connecting to the database: %w", err)
	}
	defer pool.Close()

	svc := platform.NewService(platform.NewPostgresRepository(pool))

	switch args[0] {
	case "grant":
		if len(args) != 3 {
			usage()
			return errors.New("grant needs an email and a role")
		}
		email, role := args[1], args[2]
		if !platform.ValidRoles[role] {
			return fmt.Errorf("unknown role %q — use owner, support, billing or engineering", role)
		}
		admin, err := svc.Grant(ctx, email, role)
		if err != nil {
			if errors.Is(err, platform.ErrNotFound) {
				return fmt.Errorf("no account found for %s — they must sign up first", email)
			}
			return err
		}
		fmt.Printf("granted %s to %s (%s)\n", admin.Role, admin.Email, admin.ID)
		fmt.Println("note: platform access also requires 2FA on that account.")
		return nil

	case "revoke":
		if len(args) != 2 {
			usage()
			return errors.New("revoke needs an email")
		}
		if err := svc.Revoke(ctx, args[1]); err != nil {
			if errors.Is(err, platform.ErrNotFound) {
				return fmt.Errorf("%s is not a platform admin", args[1])
			}
			return err
		}
		fmt.Printf("revoked platform access for %s\n", args[1])
		return nil

	case "list":
		admins, err := svc.Admins(ctx)
		if err != nil {
			return err
		}
		if len(admins) == 0 {
			fmt.Println("no platform admins yet — grant one with:")
			fmt.Println("  go run ./cmd/admin grant you@example.com owner")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "EMAIL\tROLE\tSTATUS\tLAST ACCESS")
		for _, a := range admins {
			last := "never"
			if a.LastAccessAt != nil {
				last = a.LastAccessAt.Format(time.RFC3339)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Email, a.Role, a.Status, last)
		}
		return w.Flush()

	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  go run ./cmd/admin list
  go run ./cmd/admin grant  <email> <owner|support|billing|engineering>
  go run ./cmd/admin revoke <email>`)
}
