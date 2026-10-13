package sales

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

func canCheckout(role string) bool {
	return role == "owner" || role == "admin" || role == "member"
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
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
	sale, created, err := h.svc.Checkout(r.Context(), t.TenantID, u.ID, in)
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

func (h *Handler) Receipt(w http.ResponseWriter, r *http.Request) {
	t, _ := tenant.FromContext(r.Context())
	if !canCheckout(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner to grant you sales access.")
		return
	}
	sale, err := h.svc.Receipt(r.Context(), t.TenantID, chi.URLParam(r, "orderID"))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"sale": sale})
}
