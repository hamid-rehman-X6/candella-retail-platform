package catalog

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

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
func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var e *Error
	if errors.As(err, &e) {
		response.Error(w, e.Status, e.Code, e.Message)
		return
	}
	h.log.Error("catalog request failed", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
}
func decode(r *http.Request, dst any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return ErrInvalid
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return ErrInvalid
	}
	return nil
}
func page(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return limit, offset
}
func scope(r *http.Request) (*tenant.Context, *auth.User) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	return t, u
}
func canWrite(t *tenant.Context) bool { return t.Role == "owner" || t.Role == "admin" }
func rejectReadOnly(w http.ResponseWriter, t *tenant.Context) bool {
	if canWrite(t) {
		return false
	}
	response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner or admin to make this change.")
	return true
}
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	t, _ := scope(r)
	limit, offset := page(r)
	list, err := h.svc.ListProducts(r.Context(), t.TenantID, r.URL.Query().Get("search"), limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"products": list})
}
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	t, _ := scope(r)
	summary, err := h.svc.Summary(r.Context(), t.TenantID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"summary": summary})
}
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	t, u := scope(r)
	if rejectReadOnly(w, t) {
		return
	}
	var in CreateProductInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	p, err := h.svc.CreateProduct(r.Context(), t.TenantID, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "product.created", EntityType: "product", EntityID: p.ID})
	response.JSON(w, http.StatusCreated, map[string]any{"product": p})
}
func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	t, u := scope(r)
	if rejectReadOnly(w, t) {
		return
	}
	var in UpdateProductInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	p, err := h.svc.UpdateProduct(r.Context(), t.TenantID, chi.URLParam(r, "productID"), in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "product.updated", EntityType: "product", EntityID: p.ID})
	response.JSON(w, http.StatusOK, map[string]any{"product": p})
}
func (h *Handler) AddVariant(w http.ResponseWriter, r *http.Request) {
	t, u := scope(r)
	if rejectReadOnly(w, t) {
		return
	}
	var in CreateVariantInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	v, err := h.svc.AddVariant(r.Context(), t.TenantID, chi.URLParam(r, "productID"), in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "product.variant_added", EntityType: "product_variant", EntityID: v.ID})
	response.JSON(w, http.StatusCreated, map[string]any{"variant": v})
}
func (h *Handler) UpdateVariant(w http.ResponseWriter, r *http.Request) {
	t, u := scope(r)
	if rejectReadOnly(w, t) {
		return
	}
	var in UpdateVariantInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	v, err := h.svc.UpdateVariant(r.Context(), t.TenantID, chi.URLParam(r, "variantID"), in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "product.variant_updated", EntityType: "product_variant", EntityID: v.ID})
	response.JSON(w, http.StatusOK, map[string]any{"variant": v})
}
func (h *Handler) ListStock(w http.ResponseWriter, r *http.Request) {
	t, _ := scope(r)
	limit, offset := page(r)
	items, storeID, err := h.svc.ListStock(r.Context(), t.TenantID, r.URL.Query().Get("search"), limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items, "storeId": storeID})
}
func (h *Handler) RecordMovement(w http.ResponseWriter, r *http.Request) {
	t, u := scope(r)
	if rejectReadOnly(w, t) {
		return
	}
	var in MovementInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	m, err := h.svc.RecordMovement(r.Context(), t.TenantID, u.ID, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "inventory." + m.Kind, EntityType: "stock_movement", EntityID: m.ID,
		Metadata: map[string]any{"variantId": m.VariantID, "quantityBefore": m.QuantityBefore, "quantityAfter": m.QuantityAfter}})
	response.JSON(w, http.StatusCreated, map[string]any{"movement": m})
}
func (h *Handler) ListMovements(w http.ResponseWriter, r *http.Request) {
	t, _ := scope(r)
	limit, offset := page(r)
	list, err := h.svc.ListMovements(r.Context(), t.TenantID, limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"movements": list})
}
