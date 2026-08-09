package platform

import (
	"context"
	"strings"
)

// ValidRoles mirrors the CHECK constraint on platform_admins.role.
var ValidRoles = map[string]bool{
	"owner": true, "support": true, "billing": true, "engineering": true,
}

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

// Service holds the platform console's business logic.
type Service struct {
	repo Repository
}

// NewService constructs the platform service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ResolveAdmin returns the active admin record for a user, or ErrNotFound.
func (s *Service) ResolveAdmin(ctx context.Context, userID string) (*Admin, error) {
	return s.repo.GetAdminByUserID(ctx, userID)
}

// TouchAccess records that an admin used the console. Best-effort.
func (s *Service) TouchAccess(ctx context.Context, adminID string) {
	_ = s.repo.TouchAdminAccess(ctx, adminID)
}

// Overview returns the headline platform numbers.
func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	return s.repo.GetOverview(ctx)
}

// Tenants returns a filtered page of tenants plus the unpaged total.
func (s *Service) Tenants(ctx context.Context, q TenantQuery) ([]TenantSummary, int, error) {
	q.Search = strings.TrimSpace(q.Search)
	if q.Limit <= 0 {
		q.Limit = defaultPageSize
	}
	if q.Limit > maxPageSize {
		q.Limit = maxPageSize
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	return s.repo.ListTenants(ctx, q)
}

// Tenant returns one tenant with members and recent activity.
func (s *Service) Tenant(ctx context.Context, tenantID string) (*TenantDetail, error) {
	return s.repo.GetTenant(ctx, tenantID)
}

// Grant gives a user console access. Used by the CLI, deliberately not exposed
// over HTTP — a "make me an admin" endpoint is a permanent liability however
// well it's guarded.
func (s *Service) Grant(ctx context.Context, email, role string) (*Admin, error) {
	if !ValidRoles[role] {
		return nil, ErrForbidden
	}
	return s.repo.GrantAdmin(ctx, strings.TrimSpace(email), role)
}

// Revoke removes console access.
func (s *Service) Revoke(ctx context.Context, email string) error {
	return s.repo.RevokeAdmin(ctx, strings.TrimSpace(email))
}

// Admins lists everyone with console access.
func (s *Service) Admins(ctx context.Context) ([]Admin, error) {
	return s.repo.ListAdmins(ctx)
}
