// Package audit writes the append-only security trail.
//
// It is the single writer to audit_logs. Every feature slice records through
// this one Recorder so that the tenant, the actor kind, the IP and the user
// agent are filled in from the request context automatically — a call site
// physically cannot forget them.
//
// This is the *audit* log, not analytics and not telemetry. It answers "who did
// what, to what, from where, when". Usage metrics belong in rollup tables and
// operational metrics belong in logs/Sentry; mixing them here produces a slow
// audit table and untrustworthy analytics.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/pkg/id"
)

// ActorType distinguishes who performed an action. Platform admins are users
// too, so without this a staff action and a customer action look identical.
type ActorType string

const (
	ActorUser          ActorType = "user"
	ActorPlatformAdmin ActorType = "platform_admin"
	ActorSystem        ActorType = "system"
)

// Entry is a single security-relevant event.
type Entry struct {
	// UserID is the human who acted; nil for system-initiated events.
	UserID *string
	// ActorType defaults to ActorUser when empty.
	ActorType ActorType
	// Action is a dotted past-tense event name, e.g. "user.login", "tenant.suspended".
	Action string
	// EntityType / EntityID identify what was acted upon.
	EntityType string
	EntityID   string
	// TenantID overrides the tenant taken from the request context. Leave empty
	// for tenant-scoped requests; set it explicitly for platform actions that
	// target a tenant the actor isn't a member of.
	TenantID string
	// Metadata carries context for the event, including any before/after diff.
	Metadata map[string]any
}

// Recorder appends entries to audit_logs.
type Recorder struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewRecorder builds the audit recorder.
func NewRecorder(db *pgxpool.Pool, log *slog.Logger) *Recorder {
	return &Recorder{db: db, log: log}
}

// Record appends an entry, filling tenant / IP / user agent from the request
// context. It is best-effort by design: a failed audit write is logged loudly
// but never fails the user's request, because refusing a completed action
// because we couldn't describe it is worse than a gap in the trail.
func (r *Recorder) Record(ctx context.Context, e Entry) {
	if e.ActorType == "" {
		e.ActorType = ActorUser
	}
	if e.TenantID == "" {
		e.TenantID = tenant.IDFromContext(ctx)
	}

	metadata := []byte("{}")
	if len(e.Metadata) > 0 {
		if encoded, err := json.Marshal(e.Metadata); err == nil {
			metadata = encoded
		}
	}

	req := RequestFromContext(ctx)

	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs
			(id, tenant_id, user_id, actor_type, action, entity_type, entity_id,
			 metadata, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id.New(id.AuditLog), nullable(e.TenantID), e.UserID, string(e.ActorType),
		e.Action, e.EntityType, e.EntityID, metadata,
		nullable(req.IP), nullable(req.UserAgent))
	if err != nil {
		r.log.Error("audit write failed",
			"action", e.Action, "entity", e.EntityType, "error", err)
	}
}

// nullable maps the empty string to a SQL NULL.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
