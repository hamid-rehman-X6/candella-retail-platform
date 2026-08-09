package workspace

import "context"

// Repository is the data-access port for the workspace slice.
type Repository interface {
	// SlugExists reports whether a live tenant already uses the given slug.
	SlugExists(ctx context.Context, slug string) (bool, error)
	// CreateWithOwner atomically creates the tenant and an owner membership for the
	// given user. On success w.ID is populated.
	CreateWithOwner(ctx context.Context, w *Workspace, ownerUserID string) error
	// ListForUser returns every workspace the user is an active member of, newest
	// first, each annotated with that user's role.
	ListForUser(ctx context.Context, userID string) ([]Workspace, error)
	// GetMembership returns the user's active membership of one live tenant, or
	// ErrNoMembership. This is the authority that turns a client-supplied tenant
	// ID into a trusted one.
	GetMembership(ctx context.Context, userID, tenantID string) (*Membership, error)
}
