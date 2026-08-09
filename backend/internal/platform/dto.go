package platform

import "time"

type adminResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
	Role     string `json:"role"`
}

func toAdminResponse(a *Admin) adminResponse {
	return adminResponse{ID: a.ID, Email: a.Email, FullName: a.FullName, Role: a.Role}
}

type statusCountsResponse struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Trialing  int `json:"trialing"`
	Suspended int `json:"suspended"`
	Cancelled int `json:"cancelled"`
}

type overviewResponse struct {
	Tenants        statusCountsResponse `json:"tenants"`
	TotalUsers     int                  `json:"totalUsers"`
	NewUsers7d     int                  `json:"newUsers7d"`
	NewTenants7d   int                  `json:"newTenants7d"`
	ActiveSessions int                  `json:"activeSessions"`
	RecentActivity []activityResponse   `json:"recentActivity"`
}

type tenantResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	IndustryType string    `json:"industryType"`
	Status       string    `json:"status"`
	CurrencyCode string    `json:"currencyCode"`
	MemberCount  int       `json:"memberCount"`
	OwnerEmail   string    `json:"ownerEmail"`
	CreatedAt    time.Time `json:"createdAt"`
}

type tenantDetailResponse struct {
	tenantResponse
	Timezone       string             `json:"timezone"`
	Members        []memberResponse   `json:"members"`
	RecentActivity []activityResponse `json:"recentActivity"`
}

type memberResponse struct {
	UserID      string     `json:"userId"`
	Email       string     `json:"email"`
	FullName    string     `json:"fullName"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	MFAEnabled  bool       `json:"mfaEnabled"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
	JoinedAt    time.Time  `json:"joinedAt"`
}

type activityResponse struct {
	ID         string    `json:"id"`
	TenantID   *string   `json:"tenantId"`
	TenantName *string   `json:"tenantName"`
	ActorType  string    `json:"actorType"`
	ActorEmail *string   `json:"actorEmail"`
	Action     string    `json:"action"`
	EntityType string    `json:"entityType"`
	EntityID   string    `json:"entityId"`
	IPAddress  *string   `json:"ipAddress"`
	CreatedAt  time.Time `json:"createdAt"`
}

func toTenantResponse(t TenantSummary) tenantResponse {
	return tenantResponse{
		ID: t.ID, Name: t.Name, Slug: t.Slug, IndustryType: t.IndustryType,
		Status: t.Status, CurrencyCode: t.CurrencyCode, MemberCount: t.MemberCount,
		OwnerEmail: t.OwnerEmail, CreatedAt: t.CreatedAt,
	}
}

func toActivityResponse(entries []ActivityEntry) []activityResponse {
	out := make([]activityResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, activityResponse{
			ID: e.ID, TenantID: e.TenantID, TenantName: e.TenantName,
			ActorType: e.ActorType, ActorEmail: e.ActorEmail, Action: e.Action,
			EntityType: e.EntityType, EntityID: e.EntityID,
			IPAddress: e.IPAddress, CreatedAt: e.CreatedAt,
		})
	}
	return out
}

func toDetailResponse(d *TenantDetail) tenantDetailResponse {
	members := make([]memberResponse, 0, len(d.Members))
	for _, m := range d.Members {
		members = append(members, memberResponse{
			UserID: m.UserID, Email: m.Email, FullName: m.FullName, Role: m.Role,
			Status: m.Status, MFAEnabled: m.MFAEnabled,
			LastLoginAt: m.LastLoginAt, JoinedAt: m.JoinedAt,
		})
	}
	return tenantDetailResponse{
		tenantResponse: toTenantResponse(d.TenantSummary),
		Timezone:       d.Timezone,
		Members:        members,
		RecentActivity: toActivityResponse(d.RecentActivity),
	}
}
