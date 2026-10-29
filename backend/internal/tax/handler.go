package tax

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
		g.Get("/tax-rates", h.List)
		g.Post("/tax-rates", h.Create)
		g.Patch("/tax-rates/{rateID}", h.Update)
	})
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	if e, ok := AsError(err); ok {
		response.Error(w, e.Status, e.Code, e.Message)
		return
	}
	h.log.Error("tax request failed", "error", err)
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

// List is open to every member: cashiers see which rate a product carries.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	rates, err := h.svc.List(r.Context(), t.TenantID, r.URL.Query().Get("include") == "archived")
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"taxRates": rates})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	var in RateInput
	if !manages(t.Role) {
		h.writeErr(w, ErrForbidden)
		return
	}
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	rate, err := h.svc.Create(r.Context(), t.TenantID, t.Role, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "tax_rate.created", EntityType: "tax_rate", EntityID: rate.ID,
		Metadata: map[string]any{"name": rate.Name, "rateBps": rate.RateBps, "inclusive": rate.Inclusive}})
	response.JSON(w, http.StatusCreated, map[string]any{"taxRate": rate})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	var in RateUpdate
	if !manages(t.Role) {
		h.writeErr(w, ErrForbidden)
		return
	}
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	rate, err := h.svc.Update(r.Context(), t.TenantID, t.Role, chi.URLParam(r, "rateID"), in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "tax_rate.updated", EntityType: "tax_rate", EntityID: rate.ID,
		Metadata: map[string]any{"name": rate.Name, "rateBps": rate.RateBps, "inclusive": rate.Inclusive, "status": rate.Status}})
	response.JSON(w, http.StatusOK, map[string]any{"taxRate": rate})
}
