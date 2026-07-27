// Package migrations embeds the SQL migration files so they can be applied by the
// migrate binary without shipping loose .sql files alongside the executable.
package migrations

import "embed"

// FS holds all goose migration files in this directory.
//
//go:embed *.sql
var FS embed.FS
