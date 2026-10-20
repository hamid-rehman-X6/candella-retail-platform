package customer

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/pkg/response"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc     *Service
	auditor *audit.Recorder
	log     *slog.Logger
}

func NewHandler(svc *Service, auditor *audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{svc, auditor, log}
}
func canRead(role string) bool   { return role == "owner" || role == "admin" || role == "member" }
func canManage(role string) bool { return role == "owner" || role == "admin" }
func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var e *Error
	if errors.As(err, &e) {
		response.Error(w, e.Status, e.Code, e.Message)
		return
	}
	h.log.Error("customer request failed", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
}
func decode(r *http.Request, dst any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
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
func pagination(r *http.Request) (int, int, error) {
	limit, offset := 50, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return 0, 0, ErrInvalidSearch
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return 0, 0, ErrInvalidSearch
		}
	}
	return limit, offset, nil
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	if !canRead(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner for customer access.")
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	list, err := h.svc.List(r.Context(), t.TenantID, r.URL.Query().Get("search"), canManage(t.Role), limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"customers": list})
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !canManage(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner to manage customers.")
		return
	}
	var in Input
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	c, err := h.svc.Create(r.Context(), t.TenantID, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "customer.created", EntityType: "customer", EntityID: c.ID})
	response.JSON(w, http.StatusCreated, map[string]any{"customer": c})
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !canManage(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner to manage customers.")
		return
	}
	var in UpdateInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	c, err := h.svc.Update(r.Context(), t.TenantID, chi.URLParam(r, "customerID"), in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	action := "customer.updated"
	if in.Status != nil && *in.Status == "archived" {
		action = "customer.archived"
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: action, EntityType: "customer", EntityID: c.ID})
	response.JSON(w, http.StatusOK, map[string]any{"customer": c})
}
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	if !canManage(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner for customer history.")
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	list, err := h.svc.History(r.Context(), t.TenantID, chi.URLParam(r, "customerID"), limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"sales": list})
}
func Routes(r chi.Router, h *Handler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Use(requireTenant)
		g.Get("/customers", h.List)
		g.Post("/customers", h.Create)
		g.Patch("/customers/{customerID}", h.Update)
		g.Get("/customers/{customerID}/sales", h.History)
	})
}
