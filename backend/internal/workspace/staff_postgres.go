package workspace

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/pkg/id"
)

type staffPostgres struct{ db *pgxpool.Pool }

func NewStaffPostgresRepository(db *pgxpool.Pool) StaffRepository { return &staffPostgres{db: db} }

func staffActor(ctx context.Context, tx pgx.Tx, tenantID, actorID string) (string, error) {
	var status, role string
	if err := tx.QueryRow(ctx, `SELECT status FROM tenants WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, tenantID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrStaffNotFound
		}
		return "", err
	}
	if status != "active" && status != "trialing" {
		return "", ErrStaffForbidden
	}
	if err := tx.QueryRow(ctx, `SELECT role FROM tenant_memberships WHERE tenant_id=$1 AND user_id=$2 AND status='active'`, tenantID, actorID).Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrStaffForbidden
		}
		return "", err
	}
	if role != "owner" && role != "admin" {
		return "", ErrStaffForbidden
	}
	return role, nil
}

func (r *staffPostgres) ListStaff(ctx context.Context, tenantID, actorID string) ([]StaffMember, []StaffInvitation, error) {
	members := []StaffMember{}
	invitations := []StaffInvitation{}
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := staffActor(ctx, tx, tenantID, actorID); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT m.user_id, u.email, u.full_name, m.role, m.status
			FROM tenant_memberships m JOIN users u ON u.id=m.user_id
			WHERE m.tenant_id=$1 AND m.status IN ('active','suspended') ORDER BY u.full_name, m.user_id`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var member StaffMember
			if err := rows.Scan(&member.UserID, &member.Email, &member.FullName, &member.Role, &member.Status); err != nil {
				rows.Close()
				return err
			}
			members = append(members, member)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		rows, err = tx.Query(ctx, `SELECT id,email,role,status,expires_at FROM staff_invitations
			WHERE tenant_id=$1 AND status='pending' ORDER BY created_at DESC`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var invitation StaffInvitation
			if err := rows.Scan(&invitation.ID, &invitation.Email, &invitation.Role, &invitation.Status, &invitation.ExpiresAt); err != nil {
				return err
			}
			invitations = append(invitations, invitation)
		}
		return rows.Err()
	})
	return members, invitations, err
}

func (r *staffPostgres) CreateInvitation(ctx context.Context, tenantID, actorID, email, role, hash string, expires time.Time) (*StaffInvitation, error) {
	invitation := &StaffInvitation{ID: id.New("invite"), Email: email, Role: role, Status: "pending", ExpiresAt: expires}
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		actorRole, err := staffActor(ctx, tx, tenantID, actorID)
		if err != nil {
			return err
		}
		if role == "owner" && actorRole != "owner" {
			return ErrStaffForbidden
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tenant_memberships m JOIN users u ON u.id=m.user_id
			WHERE m.tenant_id=$1 AND u.email=$2 AND m.status='active')`, tenantID, email).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrStaffConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE staff_invitations SET status='expired'
			WHERE tenant_id=$1 AND email=$2 AND status='pending' AND expires_at <= now()`, tenantID, email); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO staff_invitations
			(id,tenant_id,email,role,token_hash,invited_by,expires_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, invitation.ID, tenantID, email, role, hash, actorID, expires)
		return err
	})
	if isUnique(err) {
		return nil, ErrStaffConflict
	}
	return invitation, err
}

func (r *staffPostgres) ResendInvitation(ctx context.Context, tenantID, actorID, invitationID, hash string, expires time.Time) (*StaffInvitation, error) {
	var invitation StaffInvitation
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		actorRole, err := staffActor(ctx, tx, tenantID, actorID)
		if err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `SELECT id,email,role,status FROM staff_invitations
			WHERE tenant_id=$1 AND id=$2 AND status='pending' FOR UPDATE`, tenantID, invitationID).
			Scan(&invitation.ID, &invitation.Email, &invitation.Role, &invitation.Status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStaffNotFound
		}
		if err != nil {
			return err
		}
		if invitation.Role == "owner" && actorRole != "owner" {
			return ErrStaffForbidden
		}
		invitation.ExpiresAt = expires
		_, err = tx.Exec(ctx, `UPDATE staff_invitations SET token_hash=$3, expires_at=$4 WHERE tenant_id=$1 AND id=$2`, tenantID, invitationID, hash, expires)
		return err
	})
	return &invitation, err
}

func (r *staffPostgres) RevokeInvitation(ctx context.Context, tenantID, actorID, invitationID string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		actorRole, err := staffActor(ctx, tx, tenantID, actorID)
		if err != nil {
			return err
		}
		var role string
		err = tx.QueryRow(ctx, `SELECT role FROM staff_invitations WHERE tenant_id=$1 AND id=$2 AND status='pending' FOR UPDATE`, tenantID, invitationID).Scan(&role)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStaffNotFound
		}
		if err != nil {
			return err
		}
		if role == "owner" && actorRole != "owner" {
			return ErrStaffForbidden
		}
		_, err = tx.Exec(ctx, `UPDATE staff_invitations SET status='revoked' WHERE tenant_id=$1 AND id=$2`, tenantID, invitationID)
		return err
	})
}

func (r *staffPostgres) AcceptInvitation(ctx context.Context, hash, userID, email string) (string, error) {
	var tenantID string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Resolve the tenant before taking locks. Every staff writer then locks
		// the tenant first, which serializes acceptance with role and invite edits.
		if err := tx.QueryRow(ctx, `SELECT tenant_id FROM staff_invitations WHERE token_hash=$1`, hash).Scan(&tenantID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvitationInvalid
			}
			return err
		}
		var tenantStatus string
		if err := tx.QueryRow(ctx, `SELECT status FROM tenants WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, tenantID).Scan(&tenantStatus); err != nil {
			return ErrInvitationInvalid
		}
		if tenantStatus != "active" && tenantStatus != "trialing" {
			return ErrInvitationInvalid
		}
		var invitationEmail, role, status string
		var expires time.Time
		var invitationID string
		err := tx.QueryRow(ctx, `SELECT id,tenant_id,email,role,status,expires_at
			FROM staff_invitations WHERE token_hash=$1 FOR UPDATE`, hash).
			Scan(&invitationID, &tenantID, &invitationEmail, &role, &status, &expires)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvitationInvalid
		}
		if err != nil {
			return err
		}
		if status != "pending" || !expires.After(time.Now()) {
			return ErrInvitationInvalid
		}
		var actualEmail, userStatus string
		var verifiedAt *time.Time
		err = tx.QueryRow(ctx, `SELECT email,status,email_verified_at FROM users WHERE id=$1 AND deleted_at IS NULL`, userID).
			Scan(&actualEmail, &userStatus, &verifiedAt)
		if err != nil {
			return ErrInvitationEmail
		}
		if verifiedAt == nil || userStatus != "active" || !strings.EqualFold(actualEmail, invitationEmail) || !strings.EqualFold(email, invitationEmail) {
			return ErrInvitationEmail
		}
		var membershipStatus string
		err = tx.QueryRow(ctx, `SELECT status FROM tenant_memberships WHERE tenant_id=$1 AND user_id=$2 FOR UPDATE`, tenantID, userID).Scan(&membershipStatus)
		if err == nil && membershipStatus == "active" {
			return ErrStaffConflict
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO tenant_memberships (id,tenant_id,user_id,role,status)
			VALUES ($1,$2,$3,$4,'active') ON CONFLICT (tenant_id,user_id)
			DO UPDATE SET role=EXCLUDED.role,status='active'`, id.New(id.TenantMembership), tenantID, userID, role)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE staff_invitations SET status='accepted', accepted_by=$2 WHERE id=$1`, invitationID, userID)
		return err
	})
	return tenantID, err
}

func (r *staffPostgres) ChangeStaffRole(ctx context.Context, tenantID, actorID, userID, role string) error {
	return r.mutateMember(ctx, tenantID, actorID, userID, role, false)
}

func (r *staffPostgres) RemoveStaff(ctx context.Context, tenantID, actorID, userID string) error {
	return r.mutateMember(ctx, tenantID, actorID, userID, "", true)
}

func (r *staffPostgres) mutateMember(ctx context.Context, tenantID, actorID, userID, nextRole string, remove bool) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		actorRole, err := staffActor(ctx, tx, tenantID, actorID)
		if err != nil {
			return err
		}
		var currentRole string
		err = tx.QueryRow(ctx, `SELECT role FROM tenant_memberships WHERE tenant_id=$1 AND user_id=$2 AND status='active' FOR UPDATE`, tenantID, userID).Scan(&currentRole)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStaffNotFound
		}
		if err != nil {
			return err
		}
		if actorRole == "admin" && (currentRole == "owner" || nextRole == "owner" || actorID == userID) {
			return ErrStaffForbidden
		}
		if currentRole == "owner" && (remove || nextRole != "owner") {
			var owners int
			if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM tenant_memberships WHERE tenant_id=$1 AND role='owner' AND status='active'`, tenantID).Scan(&owners); err != nil {
				return err
			}
			if owners <= 1 {
				return ErrLastOwner
			}
		}
		if remove {
			_, err = tx.Exec(ctx, `UPDATE tenant_memberships SET status='removed' WHERE tenant_id=$1 AND user_id=$2`, tenantID, userID)
		} else {
			_, err = tx.Exec(ctx, `UPDATE tenant_memberships SET role=$3 WHERE tenant_id=$1 AND user_id=$2`, tenantID, userID, nextRole)
		}
		return err
	})
}

func isUnique(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
