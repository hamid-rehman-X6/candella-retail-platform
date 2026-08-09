package platform

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/pkg/response"
)

// Handler adapts HTTP requests to the platform Service.
type Handler struct {
	svc     *Service
	auditor *audit.Recorder
	log     *slog.Logger
}

// NewHandler builds the platform HTTP handler.
func NewHandler(svc *Service, auditor *audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{svc: svc, auditor: auditor, log: log}
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var de *Error
	if errors.As(err, &de) {
		response.Error(w, de.Status, de.Code, de.Message)
		return
	}
	h.log.Error("platform request failed", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
}

// GET /platform/me — the signed-in admin's identity and role.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	admin, ok := AdminFromContext(r.Context())
	if !ok {
		h.writeErr(w, ErrNotFound)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"admin": toAdminResponse(admin)})
}

// GET /platform/overview — headline platform numbers.
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.Overview(r.Context())
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"overview": overviewResponse{
		Tenants: statusCountsResponse{
			Total: o.Tenants.Total, Active: o.Tenants.Active,
			Trialing: o.Tenants.Trialing, Suspended: o.Tenants.Suspended,
			Cancelled: o.Tenants.Cancelled,
		},
		TotalUsers:     o.TotalUsers,
		NewUsers7d:     o.NewUsers7d,
		NewTenants7d:   o.NewTenants7d,
		ActiveSessions: o.ActiveSessions,
		RecentActivity: toActivityResponse(o.RecentActivity),
	}})
}

// GET /platform/tenants — filtered, paged list of every tenant.
func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list, total, err := h.svc.Tenants(r.Context(), TenantQuery{
		Search:   q.Get("search"),
		Status:   q.Get("status"),
		Industry: q.Get("industry"),
		Limit:    atoiDefault(q.Get("limit"), defaultPageSize),
		Offset:   atoiDefault(q.Get("offset"), 0),
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}

	out := make([]tenantResponse, 0, len(list))
	for _, t := range list {
		out = append(out, toTenantResponse(t))
	}
	response.JSON(w, http.StatusOK, map[string]any{"tenants": out, "total": total})
}

// GET /platform/tenants/{tenantID} — one tenant, its members and its activity.
func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")

	detail, err := h.svc.Tenant(r.Context(), tenantID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	// Opening a customer's record exposes their members' names and emails, so
	// this read is audited — unlike list/overview, which are aggregates.
	if admin, ok := AdminFromContext(r.Context()); ok {
		if user, ok := auth.UserFromContext(r.Context()); ok {
			h.auditor.Record(r.Context(), audit.Entry{
				UserID:     &user.ID,
				ActorType:  audit.ActorPlatformAdmin,
				Action:     "platform.tenant.viewed",
				EntityType: "tenant",
				EntityID:   tenantID,
				TenantID:   tenantID,
				Metadata:   map[string]any{"adminRole": admin.Role},
			})
		}
	}

	response.JSON(w, http.StatusOK, map[string]any{"tenant": toDetailResponse(detail)})
}

// atoiDefault parses a query-string integer, falling back when absent/invalid.
func atoiDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
