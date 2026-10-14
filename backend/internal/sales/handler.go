package sales

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
	"candella-ecosystem/backend/internal/store"
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

func canCheckout(role string) bool {
	return role == "owner" || role == "admin" || role == "member"
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var storeErr *store.Error
	if errors.As(err, &storeErr) {
		response.Error(w, storeErr.Status, storeErr.Code, storeErr.Message)
		return
	}
	if e, ok := asSaleError(err); ok {
		response.Error(w, e.Status, e.Code, e.Message)
		return
	}
	h.log.Error("sales request failed", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
}

func decodeCheckout(r *http.Request, dst *CheckoutInput) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return ErrInvalid
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrInvalid
	}
	return nil
}

func parseSalesInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, ErrInvalidFilter
	}
	return n, nil
}

func parseOptionalAmount(raw string) (*int64, error) {
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, ErrInvalidFilter
	}
	return &n, nil
}

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !canCheckout(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner to grant you checkout access.")
		return
	}
	var in CheckoutInput
	if err := decodeCheckout(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	if in.StoreID == "" {
		h.writeErr(w, store.ErrSelection)
		return
	}
	r = r.WithContext(store.WithSelection(r.Context(), in.StoreID, u.ID, t.Role))
	sale, created, err := h.svc.Checkout(r.Context(), t.TenantID, u.ID, u.FullName, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if created {
		h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, Action: "sale.completed", EntityType: "order", EntityID: sale.ID,
			Metadata: map[string]any{"orderNumber": sale.OrderNumber, "totalMinor": sale.GrandTotalMinor}})
		response.JSON(w, http.StatusCreated, map[string]any{"sale": sale})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"sale": sale})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !canCheckout(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner to grant you sales access.")
		return
	}
	query := r.URL.Query()
	filter := SalesFilter{
		FromDate: query.Get("fromDate"), ToDate: query.Get("toDate"),
		OrderNumber: query.Get("orderNumber"), Cashier: query.Get("cashier"),
	}
	var err error
	filter.Limit, err = parseSalesInt(query.Get("limit"), 50)
	if err == nil {
		filter.Offset, err = parseSalesInt(query.Get("offset"), 0)
	}
	if err == nil {
		filter.MinAmountMinor, err = parseOptionalAmount(query.Get("minAmountMinor"))
	}
	if err == nil {
		filter.MaxAmountMinor, err = parseOptionalAmount(query.Get("maxAmountMinor"))
	}
	if err != nil {
		h.writeErr(w, ErrInvalidFilter)
		return
	}
	list, hasMore, err := h.svc.Search(store.WithActor(r.Context(), u.ID, t.Role), t.TenantID, filter)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"sales": list, "hasMore": hasMore})
}

func (h *Handler) Receipt(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !canCheckout(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner to grant you sales access.")
		return
	}
	sale, err := h.svc.Receipt(store.WithActor(r.Context(), u.ID, t.Role), t.TenantID, chi.URLParam(r, "orderID"))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"sale": sale})
}
