package register

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

// Anyone who can take a cash sale can work a shift.
func canUse(role string) bool {
	return role == "owner" || role == "admin" || role == "member"
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var storeErr *store.Error
	if errors.As(err, &storeErr) {
		response.Error(w, storeErr.Status, storeErr.Code, storeErr.Message)
		return
	}
	if e, ok := AsError(err); ok {
		response.Error(w, e.Status, e.Code, e.Message)
		return
	}
	h.log.Error("register request failed", "error", err)
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

// request resolves the caller and refuses roles that cannot work a till.
func (h *Handler) request(w http.ResponseWriter, r *http.Request) (string, Actor, bool) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	if !canUse(t.Role) {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner to grant you checkout access.")
		return "", Actor{}, false
	}
	return t.TenantID, Actor{UserID: u.ID, Name: u.FullName, Role: t.Role}, true
}

// inStore scopes the request to an explicitly selected store, which
// store.Resolve then checks the caller may use.
func inStore(r *http.Request, storeID string, actor Actor) *http.Request {
	return r.WithContext(store.WithSelection(r.Context(), storeID, actor.UserID, actor.Role))
}

func (h *Handler) ListRegisters(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := h.request(w, r)
	if !ok {
		return
	}
	storeID := r.URL.Query().Get("storeId")
	if storeID == "" {
		h.writeErr(w, store.ErrSelection)
		return
	}
	registers, err := h.svc.ListRegisters(inStore(r, storeID, actor).Context(), tenantID, storeID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"registers": registers})
}

func (h *Handler) CreateRegister(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := h.request(w, r)
	if !ok {
		return
	}
	var in CreateRegisterInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, ErrInvalidName)
		return
	}
	if in.StoreID == "" {
		h.writeErr(w, store.ErrSelection)
		return
	}
	r = inStore(r, in.StoreID, actor)
	reg, err := h.svc.CreateRegister(r.Context(), tenantID, actor, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &actor.UserID, Action: "register.created", EntityType: "register",
		EntityID: reg.ID, Metadata: map[string]any{"storeId": reg.StoreID, "name": reg.Name}})
	response.JSON(w, http.StatusCreated, map[string]any{"register": reg})
}

func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := h.request(w, r)
	if !ok {
		return
	}
	var in OpenInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	if in.StoreID == "" {
		h.writeErr(w, store.ErrSelection)
		return
	}
	r = inStore(r, in.StoreID, actor)
	shift, err := h.svc.Open(r.Context(), tenantID, actor, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &actor.UserID, Action: "shift.opened", EntityType: "register_session",
		EntityID: shift.ID, Metadata: map[string]any{"registerId": shift.RegisterID, "openingCashMinor": shift.OpeningCashMinor}})
	response.JSON(w, http.StatusCreated, map[string]any{"shift": shift})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := h.request(w, r)
	if !ok {
		return
	}
	shift, err := h.svc.Get(r.Context(), tenantID, actor, chi.URLParam(r, "shiftID"))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"shift": shift})
}

func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := h.request(w, r)
	if !ok {
		return
	}
	var in CloseInput
	if err := decode(r, &in); err != nil {
		h.writeErr(w, err)
		return
	}
	shift, err := h.svc.Close(r.Context(), tenantID, actor, chi.URLParam(r, "shiftID"), in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &actor.UserID, Action: "shift.closed", EntityType: "register_session",
		EntityID: shift.ID, Metadata: map[string]any{"expectedCashMinor": shift.ExpectedCashMinor,
			"countedCashMinor": shift.CountedCashMinor, "varianceMinor": shift.VarianceMinor}})
	response.JSON(w, http.StatusOK, map[string]any{"shift": shift})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := h.request(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	filter := ShiftFilter{StoreID: query.Get("storeId"), Status: query.Get("status")}
	if filter.StoreID == "" {
		h.writeErr(w, store.ErrSelection)
		return
	}
	var err error
	if filter.Limit, err = atoi(query.Get("limit")); err == nil {
		filter.Offset, err = atoi(query.Get("offset"))
	}
	if err != nil {
		h.writeErr(w, ErrInvalidFilter)
		return
	}
	shifts, hasMore, err := h.svc.List(inStore(r, filter.StoreID, actor).Context(), tenantID, actor, filter)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"shifts": shifts, "hasMore": hasMore})
}

func atoi(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, ErrInvalidFilter
	}
	return n, nil
}
