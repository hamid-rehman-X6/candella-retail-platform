package promotion

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/pkg/response"
)

type Handler struct {
	svc     *Service
	auditor *audit.Recorder
	log     *slog.Logger
}

func NewHandler(svc *Service, auditor *audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{svc: svc, auditor: auditor, log: log}
}

func Routes(r chi.Router, h *Handler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Use(requireTenant)
		g.Get("/promotions", h.List)
		g.Post("/promotions", h.Create)
		g.Put("/promotions/{promotionID}", h.Update)
		g.Get("/discount-settings", h.Settings)
		g.Put("/discount-settings", h.SaveSettings)
	})
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	if e, ok := AsError(err); ok {
		response.Error(w, e.Status, e.Code, e.Message)
		return
	}
	h.log.Error("promotion request failed", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
}

func decode(r *http.Request, dst any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<16))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return ErrInvalid
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrInvalid
	}
	return nil
}

// List is open to every member: the till offers the active promotions.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	list, err := h.svc.List(r.Context(), t.TenantID, r.URL.Query().Get("include") == "archived")
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"promotions": list})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !Manages(t.Role) {
		h.writeErr(w, ErrForbidden)
		return
	}
	var in Input
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	p, err := h.svc.Create(r.Context(), t.TenantID, t.Role, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "promotion.created", EntityType: "promotion", EntityID: p.ID,
		Metadata: map[string]any{"name": p.Name, "kind": p.Kind, "percentBps": p.PercentBps, "amountMinor": p.AmountMinor, "appliesTo": p.AppliesTo}})
	response.JSON(w, http.StatusCreated, map[string]any{"promotion": p})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !Manages(t.Role) {
		h.writeErr(w, ErrForbidden)
		return
	}
	var body struct {
		Input
		Status *string `json:"status"`
	}
	if err := decode(r, &body); err != nil {
		h.writeErr(w, err)
		return
	}
	p, err := h.svc.Update(r.Context(), t.TenantID, t.Role, chi.URLParam(r, "promotionID"), body.Input, body.Status)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "promotion.updated", EntityType: "promotion", EntityID: p.ID,
		Metadata: map[string]any{"name": p.Name, "kind": p.Kind, "percentBps": p.PercentBps, "amountMinor": p.AmountMinor, "status": p.Status}})
	response.JSON(w, http.StatusOK, map[string]any{"promotion": p})
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	s, err := h.svc.Settings(r.Context(), t.TenantID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"settings": s})
}

func (h *Handler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !Manages(t.Role) {
		h.writeErr(w, ErrForbidden)
		return
	}
	var in Settings
	if err := decode(r, &in); err != nil {
		h.writeErr(w, ErrSettings)
		return
	}
	s, err := h.svc.SaveSettings(r.Context(), t.TenantID, t.Role, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "discount_limit.updated", EntityType: "tenant", EntityID: t.TenantID,
		Metadata: map[string]any{"cashierLimitBps": s.CashierLimitBps}})
	response.JSON(w, http.StatusOK, map[string]any{"settings": s})
}
