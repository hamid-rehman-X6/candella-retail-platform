// Package workspace is the vertical slice for a customer's business/workspace.
// A "workspace" is the user-facing name for a tenant; it is stored in the
// `tenants` + `tenant_memberships` tables. Users belong to one or more workspaces
// through a membership that carries their role.
package workspace

import "time"

// Workspace is a tenant plus the current user's role in it.
type Workspace struct {
	ID           string
	Name         string
	Slug         string
	IndustryType string
	Status       string
	CurrencyCode string
	Timezone     string
	Role         string // the requesting user's role: owner | admin | member
	CreatedAt    time.Time
}
