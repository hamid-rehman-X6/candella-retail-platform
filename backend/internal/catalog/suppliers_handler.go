package catalog

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/pkg/response"
)

func (h *Handler) ListSuppliers(w http.ResponseWriter, r *http.Request) {
	t, _ := scope(r)
	limit, offset := page(r)
	suppliers, err := h.svc.ListSuppliers(r.Context(), t.TenantID, r.URL.Query().Get("search"), limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"suppliers": suppliers})
}

func (h *Handler) CreateSupplier(w http.ResponseWriter, r *http.Request) {
	t, u := scope(r)
	if rejectReadOnly(w, t) {
		return
	}
	var in SupplierInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	supplier, err := h.svc.CreateSupplier(r.Context(), t.TenantID, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "supplier.created", EntityType: "supplier", EntityID: supplier.ID})
	response.JSON(w, http.StatusCreated, map[string]any{"supplier": supplier})
}

func (h *Handler) UpdateSupplier(w http.ResponseWriter, r *http.Request) {
	t, u := scope(r)
	if rejectReadOnly(w, t) {
		return
	}
	var in SupplierUpdateInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	supplier, err := h.svc.UpdateSupplier(r.Context(), t.TenantID, chi.URLParam(r, "supplierID"), in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	action := "supplier.updated"
	if in.Status != nil && *in.Status == "archived" {
		action = "supplier.archived"
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: action, EntityType: "supplier", EntityID: supplier.ID})
	response.JSON(w, http.StatusOK, map[string]any{"supplier": supplier})
}

func (h *Handler) ListSupplierSources(w http.ResponseWriter, r *http.Request) {
	t, _ := scope(r)
	limit, offset := page(r)
	sources, err := h.svc.ListSupplierSources(r.Context(), t.TenantID, chi.URLParam(r, "supplierID"), limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"sources": sources})
}

func (h *Handler) SaveSupplierSource(w http.ResponseWriter, r *http.Request) {
	t, u := scope(r)
	if rejectReadOnly(w, t) {
		return
	}
	var in SupplierSourceInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	source, err := h.svc.SaveSupplierSource(r.Context(), t.TenantID, chi.URLParam(r, "supplierID"), in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "supplier.source_saved", EntityType: "supplier_variant",
		EntityID: source.SupplierID + ":" + source.VariantID,
		Metadata: map[string]any{"supplierId": source.SupplierID, "variantId": source.VariantID}})
	response.JSON(w, http.StatusOK, map[string]any{"source": source})
}
