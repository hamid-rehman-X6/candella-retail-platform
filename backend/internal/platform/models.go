// Package platform is the vertical slice for the Candella staff console — the
// platform plane. It reads across every tenant, which is exactly why it is kept
// physically separate from the tenant-scoped slices: its own routes, its own
// middleware, and no shared handlers that could accidentally leak cross-tenant
// data into a customer-facing response.
package platform

import "time"

// Admin is a Candella staff member with console access.
type Admin struct {
	ID           string
	UserID       string
	Email        string
	FullName     string
	Role         string // owner | support | billing | engineering
	Status       string // active | suspended
	LastAccessAt *time.Time
	CreatedAt    time.Time
}

// Overview is the headline state of the platform.
type Overview struct {
	Tenants        StatusCounts
	TotalUsers     int
	NewUsers7d     int
	NewTenants7d   int
	ActiveSessions int
	RecentActivity []ActivityEntry
}

// StatusCounts breaks tenants down by lifecycle status.
type StatusCounts struct {
	Total     int
	Active    int
	Trialing  int
	Suspended int
	Cancelled int
}

// TenantSummary is one row in the tenants list.
type TenantSummary struct {
	ID           string
	Name         string
	Slug         string
	IndustryType string
	Status       string
	CurrencyCode string
	MemberCount  int
	OwnerEmail   string
	CreatedAt    time.Time
}

// TenantDetail is everything the console shows for a single tenant.
type TenantDetail struct {
	TenantSummary
	Timezone       string
	Members        []TenantMember
	RecentActivity []ActivityEntry
}

// TenantMember is a user's membership of a tenant, as staff see it.
type TenantMember struct {
	UserID      string
	Email       string
	FullName    string
	Role        string
	Status      string
	MFAEnabled  bool
	LastLoginAt *time.Time
	JoinedAt    time.Time
}

// ActivityEntry is one row of the audit trail.
type ActivityEntry struct {
	ID         string
	TenantID   *string
	TenantName *string
	ActorType  string
	ActorEmail *string
	Action     string
	EntityType string
	EntityID   string
	IPAddress  *string
	CreatedAt  time.Time
}

// TenantQuery filters the tenants list.
type TenantQuery struct {
	Search   string
	Status   string
	Industry string
	Limit    int
	Offset   int
}
