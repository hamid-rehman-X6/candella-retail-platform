package purchase

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/store"
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
func canManage(role string) bool { return role == "owner" || role == "admin" }
func (h *Handler) allowed(w http.ResponseWriter, r *http.Request) bool {
	t, _ := tenant.FromContext(r.Context())
	if canManage(t.Role) {
		return true
	}
	response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner for purchase order access.")
	return false
}
func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var se *store.Error
	if errors.As(err, &se) {
		response.Error(w, se.Status, se.Code, se.Message)
		return
	}
	var pe *Error
	if errors.As(err, &pe) {
		response.Error(w, pe.Status, pe.Code, pe.Message)
		return
	}
	h.log.Error("purchase request failed", "error", err)
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
			return 0, 0, ErrInvalid
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return 0, 0, ErrInvalid
		}
	}
	return limit, offset, nil
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r) {
		return
	}
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	storeID := r.URL.Query().Get("storeId")
	if storeID == "" {
		h.writeErr(w, store.ErrSelection)
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	list, err := h.svc.List(store.WithSelection(r.Context(), storeID, u.ID, t.Role), t.TenantID, limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"orders": list})
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r) {
		return
	}
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	var in CreateInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	order, created, err := h.svc.Create(r.Context(), t.TenantID, u.ID, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if created {
		h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "purchase_order.created", EntityType: "purchase_order", EntityID: order.ID})
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	response.JSON(w, status, map[string]any{"order": order})
}
func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r) {
		return
	}
	t, _ := tenant.FromContext(r.Context())
	order, err := h.svc.Find(r.Context(), t.TenantID, chi.URLParam(r, "orderID"))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"order": order})
}
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r) {
		return
	}
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	order, changed, err := h.svc.Approve(r.Context(), t.TenantID, chi.URLParam(r, "orderID"), u.ID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if changed {
		h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "purchase_order.approved", EntityType: "purchase_order", EntityID: order.ID})
	}
	response.JSON(w, http.StatusOK, map[string]any{"order": order})
}
func (h *Handler) Receive(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(w, r) {
		return
	}
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	var in ReceiveInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	order, receipt, created, err := h.svc.Receive(r.Context(), t.TenantID, chi.URLParam(r, "orderID"), u.ID, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if created {
		h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "purchase_order.received", EntityType: "goods_receipt", EntityID: receipt.ID,
			Metadata: map[string]any{"purchaseOrderId": order.ID, "storeId": order.StoreID}})
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	response.JSON(w, status, map[string]any{"order": order, "receipt": receipt})
}
func Routes(r chi.Router, h *Handler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Use(requireTenant)
		g.Get("/purchase-orders", h.List)
		g.Post("/purchase-orders", h.Create)
		g.Get("/purchase-orders/{orderID}", h.Find)
		g.Post("/purchase-orders/{orderID}/approve", h.Approve)
		g.Post("/purchase-orders/{orderID}/receipts", h.Receive)
	})
}
