package workspace

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/pkg/response"
)

type StaffHandler struct {
	svc     *StaffService
	auditor *audit.Recorder
}

func NewStaffHandler(svc *StaffService, auditor *audit.Recorder) *StaffHandler {
	return &StaffHandler{svc: svc, auditor: auditor}
}

func (h *StaffHandler) writeErr(w http.ResponseWriter, err error) {
	if e, ok := err.(*Error); ok {
		response.Error(w, e.Status, e.Code, e.Message)
		return
	}
	response.Error(w, http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
}

func (h *StaffHandler) actor(r *http.Request) (string, string) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	return t.TenantID, u.ID
}

func (h *StaffHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, actorID := h.actor(r)
	members, invitations, err := h.svc.List(r.Context(), tenantID, actorID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"members": members, "invitations": invitations})
}

func (h *StaffHandler) Invite(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decodeJSON(r, &input); err != nil {
		h.writeErr(w, err)
		return
	}
	tenantID, actorID := h.actor(r)
	invitation, err := h.svc.Invite(r.Context(), tenantID, actorID, input.Email, input.Role)
	if invitation != nil {
		h.auditor.Record(r.Context(), audit.Entry{UserID: &actorID, Action: "staff.invited", EntityType: "staff_invitation", EntityID: invitation.ID, Metadata: map[string]any{"email": invitation.Email, "role": invitation.Role}})
	}
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"invitation": invitation})
}

func (h *StaffHandler) Resend(w http.ResponseWriter, r *http.Request) {
	tenantID, actorID := h.actor(r)
	invitation, err := h.svc.Resend(r.Context(), tenantID, actorID, chi.URLParam(r, "invitationID"))
	if invitation != nil {
		h.auditor.Record(r.Context(), audit.Entry{UserID: &actorID, Action: "staff.invitation_resent", EntityType: "staff_invitation", EntityID: invitation.ID})
	}
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"invitation": invitation})
}

func (h *StaffHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	tenantID, actorID := h.actor(r)
	invitationID := chi.URLParam(r, "invitationID")
	if err := h.svc.Revoke(r.Context(), tenantID, actorID, invitationID); err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &actorID, Action: "staff.invitation_revoked", EntityType: "staff_invitation", EntityID: invitationID})
	response.JSON(w, http.StatusOK, map[string]bool{"revoked": true})
}

func (h *StaffHandler) Accept(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &input); err != nil {
		h.writeErr(w, err)
		return
	}
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		h.writeErr(w, ErrUnauthenticated)
		return
	}
	tenantID, err := h.svc.Accept(r.Context(), input.Token, u.ID, u.Email, u.IsEmailVerified())
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &u.ID, TenantID: tenantID, Action: "staff.invitation_accepted", EntityType: "tenant_membership", EntityID: u.ID})
	response.JSON(w, http.StatusOK, map[string]any{"tenantId": tenantID})
}

func (h *StaffHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &input); err != nil {
		h.writeErr(w, err)
		return
	}
	tenantID, actorID := h.actor(r)
	userID := chi.URLParam(r, "userID")
	if err := h.svc.ChangeRole(r.Context(), tenantID, actorID, userID, input.Role); err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &actorID, Action: "staff.role_changed", EntityType: "tenant_membership", EntityID: userID, Metadata: map[string]any{"role": input.Role}})
	response.JSON(w, http.StatusOK, map[string]bool{"updated": true})
}

func (h *StaffHandler) Remove(w http.ResponseWriter, r *http.Request) {
	tenantID, actorID := h.actor(r)
	userID := chi.URLParam(r, "userID")
	if err := h.svc.Remove(r.Context(), tenantID, actorID, userID); err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &actorID, Action: "staff.removed", EntityType: "tenant_membership", EntityID: userID})
	response.JSON(w, http.StatusOK, map[string]bool{"removed": true})
}

func StaffRoutes(r chi.Router, h *StaffHandler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Post("/invitations/accept", h.Accept)
		g.Group(func(t chi.Router) {
			t.Use(requireTenant)
			t.Get("/staff", h.List)
			t.Post("/staff/invitations", h.Invite)
			t.Post("/staff/invitations/{invitationID}/resend", h.Resend)
			t.Delete("/staff/invitations/{invitationID}", h.Revoke)
			t.Patch("/staff/{userID}", h.ChangeRole)
			t.Delete("/staff/{userID}", h.Remove)
		})
	})
}
