package workspace

import (
	"context"
	"crypto/rand"
	"strings"
)

// validIndustries mirrors the CHECK constraint on tenants.industry_type.
var validIndustries = map[string]bool{
	"pos_general": true, "pharmacy": true, "garments": true, "cosmetics": true,
	"grocery": true, "electronics": true, "wholesale": true, "other": true,
}

// Service holds the workspace business logic.
type Service struct {
	repo Repository
}

// NewService constructs the workspace service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create makes a new workspace owned by the given user.
func (s *Service) Create(ctx context.Context, ownerUserID string, req createRequest) (*Workspace, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrNameRequired
	}

	industry := strings.TrimSpace(req.IndustryType)
	if industry == "" {
		industry = "pos_general"
	}
	if !validIndustries[industry] {
		return nil, ErrInvalidIndustry
	}

	currency := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if len(currency) != 3 {
		currency = "USD"
	}
	timezone := strings.TrimSpace(req.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}

	slug, err := s.uniqueSlug(ctx, name)
	if err != nil {
		return nil, err
	}

	w := &Workspace{
		Name:         name,
		Slug:         slug,
		IndustryType: industry,
		Status:       "active",
		CurrencyCode: currency,
		Timezone:     timezone,
	}
	if err := s.repo.CreateWithOwner(ctx, w, ownerUserID); err != nil {
		return nil, err
	}
	return w, nil
}

// List returns the workspaces the user belongs to.
func (s *Service) List(ctx context.Context, userID string) ([]Workspace, error) {
	return s.repo.ListForUser(ctx, userID)
}

// ResolveMembership verifies that the user is an active member of the tenant and
// that the tenant is usable. A suspended workspace resolves to a distinct error
// so the UI can explain the situation rather than pretending it doesn't exist.
func (s *Service) ResolveMembership(ctx context.Context, userID, tenantID string) (*Membership, error) {
	m, err := s.repo.GetMembership(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	if m.TenantStatus == "suspended" || m.TenantStatus == "cancelled" {
		return nil, ErrTenantSuspended
	}
	return m, nil
}

// uniqueSlug derives a URL-safe slug from the name, appending a short random
// suffix if the base slug is already taken.
func (s *Service) uniqueSlug(ctx context.Context, name string) (string, error) {
	base := slugify(name)
	if base == "" {
		base = "workspace"
	}

	exists, err := s.repo.SlugExists(ctx, base)
	if err != nil {
		return "", err
	}
	if !exists {
		return base, nil
	}

	for i := 0; i < 5; i++ {
		candidate := base + "-" + randSuffix(5)
		exists, err := s.repo.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return base + "-" + randSuffix(8), nil
}

// slugify lowercases the input and replaces runs of non-alphanumerics with a
// single hyphen, trimming leading/trailing hyphens.
func slugify(s string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// randSuffix returns n lowercase-alphanumeric random characters for slug uniqueness.
func randSuffix(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "x"
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}
