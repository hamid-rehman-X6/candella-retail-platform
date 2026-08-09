package platform

import "context"

// Repository is the data-access port for the platform console. Every method here
// reads across tenants by design — which is why nothing in this interface is
// reachable from a tenant-scoped route.
type Repository interface {
	// GetAdminByUserID returns the active admin record for a user, or ErrNotFound.
	GetAdminByUserID(ctx context.Context, userID string) (*Admin, error)
	// TouchAdminAccess stamps last_access_at (best-effort telemetry).
	TouchAdminAccess(ctx context.Context, adminID string) error
	// GrantAdmin creates or reactivates an admin for the user with the given email.
	GrantAdmin(ctx context.Context, email, role string) (*Admin, error)
	// RevokeAdmin removes console access from the user with the given email.
	RevokeAdmin(ctx context.Context, email string) error
	// ListAdmins returns every admin, newest first.
	ListAdmins(ctx context.Context) ([]Admin, error)

	// GetOverview aggregates the headline platform numbers.
	GetOverview(ctx context.Context) (*Overview, error)
	// ListTenants returns a filtered page of tenants plus the unpaged total.
	ListTenants(ctx context.Context, q TenantQuery) ([]TenantSummary, int, error)
	// GetTenant returns one tenant with its members and recent activity.
	GetTenant(ctx context.Context, tenantID string) (*TenantDetail, error)
	// RecentActivity returns the newest audit entries across all tenants.
	RecentActivity(ctx context.Context, limit int) ([]ActivityEntry, error)
}
