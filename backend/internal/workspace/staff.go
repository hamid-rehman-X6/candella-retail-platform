package workspace

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	"candella-ecosystem/backend/pkg/crypto"
	"candella-ecosystem/backend/pkg/mailer"
	"candella-ecosystem/backend/pkg/validate"
)

var (
	ErrStaffForbidden    = newError(http.StatusForbidden, "staff_forbidden", "Only a workspace owner or admin can manage staff.")
	ErrStaffNotFound     = newError(http.StatusNotFound, "staff_not_found", "That staff member or invitation was not found in this workspace.")
	ErrStaffConflict     = newError(http.StatusConflict, "staff_conflict", "This person already has access or a pending invitation.")
	ErrLastOwner         = newError(http.StatusConflict, "last_owner", "Assign another owner before removing or changing the last owner.")
	ErrInvitationInvalid = newError(http.StatusBadRequest, "invitation_invalid", "This invitation is invalid, expired, or has already been used.")
	ErrInvitationEmail   = newError(http.StatusForbidden, "invitation_email_mismatch", "Sign in with the verified email address that received this invitation.")
	ErrInviteDelivery    = newError(http.StatusBadGateway, "invitation_delivery_failed", "Invitation saved, but email delivery failed. Open Staff and resend it.")
)

type StaffMember struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type StaffInvitation struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type StaffRepository interface {
	ListStaff(context.Context, string, string) ([]StaffMember, []StaffInvitation, error)
	CreateInvitation(context.Context, string, string, string, string, string, time.Time) (*StaffInvitation, error)
	ResendInvitation(context.Context, string, string, string, string, time.Time) (*StaffInvitation, error)
	RevokeInvitation(context.Context, string, string, string) error
	AcceptInvitation(context.Context, string, string, string) (string, error)
	ChangeStaffRole(context.Context, string, string, string, string) error
	RemoveStaff(context.Context, string, string, string) error
}

type StaffService struct {
	repo       StaffRepository
	mail       mailer.Mailer
	appBaseURL string
}

func NewStaffService(repo StaffRepository, mail mailer.Mailer, appBaseURL string) *StaffService {
	return &StaffService{repo: repo, mail: mail, appBaseURL: strings.TrimRight(appBaseURL, "/")}
}

func (s *StaffService) List(ctx context.Context, tenantID, actorID string) ([]StaffMember, []StaffInvitation, error) {
	return s.repo.ListStaff(ctx, tenantID, actorID)
}

func validStaffRole(role string) bool { return role == "owner" || role == "admin" || role == "member" }

func invitationToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *StaffService) deliver(ctx context.Context, invitation *StaffInvitation, token string) error {
	link := s.appBaseURL + "/invitations/accept?token=" + url.QueryEscape(token)
	return s.mail.Send(ctx, mailer.Email{
		To:      invitation.Email,
		Subject: "Join your Candella workspace",
		Text:    "You have been invited to a Candella workspace as " + invitation.Role + ". Open this link within 7 days, sign in with this email address, and accept: " + link,
	})
}

func (s *StaffService) Invite(ctx context.Context, tenantID, actorID, email, role string) (*StaffInvitation, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validate.Email(email) || len(email) > 320 || !validStaffRole(role) {
		return nil, ErrInvalidInput
	}
	token, err := invitationToken()
	if err != nil {
		return nil, err
	}
	invitation, err := s.repo.CreateInvitation(ctx, tenantID, actorID, email, role, crypto.HashToken(token), time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	if err := s.deliver(ctx, invitation, token); err != nil {
		return invitation, ErrInviteDelivery
	}
	return invitation, nil
}

func (s *StaffService) Resend(ctx context.Context, tenantID, actorID, invitationID string) (*StaffInvitation, error) {
	token, err := invitationToken()
	if err != nil {
		return nil, err
	}
	invitation, err := s.repo.ResendInvitation(ctx, tenantID, actorID, invitationID, crypto.HashToken(token), time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	if err := s.deliver(ctx, invitation, token); err != nil {
		return invitation, ErrInviteDelivery
	}
	return invitation, nil
}

func (s *StaffService) Revoke(ctx context.Context, tenantID, actorID, invitationID string) error {
	return s.repo.RevokeInvitation(ctx, tenantID, actorID, invitationID)
}

func (s *StaffService) Accept(ctx context.Context, token, userID, email string, verified bool) (string, error) {
	if !verified || token == "" || len(token) > 200 {
		return "", ErrInvitationInvalid
	}
	return s.repo.AcceptInvitation(ctx, crypto.HashToken(token), userID, email)
}

func (s *StaffService) ChangeRole(ctx context.Context, tenantID, actorID, userID, role string) error {
	if !validStaffRole(role) {
		return ErrInvalidInput
	}
	return s.repo.ChangeStaffRole(ctx, tenantID, actorID, userID, role)
}

func (s *StaffService) Remove(ctx context.Context, tenantID, actorID, userID string) error {
	return s.repo.RemoveStaff(ctx, tenantID, actorID, userID)
}
