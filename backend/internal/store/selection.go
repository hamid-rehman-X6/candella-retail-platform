package store

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrSelection = &Error{http.StatusBadRequest, "store_required", "Select a store before viewing stock or checking out."}
	ErrAccess    = &Error{http.StatusForbidden, "store_access_denied", "You cannot use this store. Ask a workspace admin for access."}
	ErrNotFound  = &Error{http.StatusNotFound, "store_not_found", "This store was not found in your workspace."}
	ErrInvalid   = &Error{http.StatusBadRequest, "invalid_store", "Enter a store name between 1 and 100 characters."}
	ErrConflict  = &Error{http.StatusConflict, "store_conflict", "A store with this name already exists in your workspace."}
)

type selectionKey struct{}
type selection struct{ ID, UserID, Role string }

func WithActor(ctx context.Context, userID, role string) context.Context {
	return context.WithValue(ctx, selectionKey{}, selection{UserID: userID, Role: role})
}

func Actor(ctx context.Context) (string, string, bool) {
	selected, ok := ctx.Value(selectionKey{}).(selection)
	return selected.UserID, selected.Role, ok
}

func WithSelection(ctx context.Context, storeID, userID, role string) context.Context {
	return context.WithValue(ctx, selectionKey{}, selection{storeID, userID, role})
}

type Location struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsDefault  bool   `json:"isDefault"`
	Accessible bool   `json:"accessible"`
}

// Resolve runs inside a tenant-scoped transaction. HTTP handlers require an explicit
// store; the empty selection keeps older internal service callers on the default.
func Resolve(ctx context.Context, tx pgx.Tx, tenantID string) (Location, error) {
	sel, selected := ctx.Value(selectionKey{}).(selection)
	var loc Location
	var err error
	if selected {
		if sel.ID == "" {
			return loc, ErrSelection
		}
		err = tx.QueryRow(ctx, `SELECT id,name,is_default FROM stores WHERE tenant_id=$1 AND id=$2`, tenantID, sel.ID).Scan(&loc.ID, &loc.Name, &loc.IsDefault)
	} else {
		err = tx.QueryRow(ctx, `SELECT id,name,is_default FROM stores WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&loc.ID, &loc.Name, &loc.IsDefault)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return loc, ErrNotFound
	}
	if err != nil {
		return loc, err
	}
	if selected && sel.Role == "member" && !loc.IsDefault {
		var allowed bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM store_access WHERE tenant_id=$1 AND store_id=$2 AND user_id=$3)`, tenantID, loc.ID, sel.UserID).Scan(&allowed); err != nil {
			return loc, err
		}
		if !allowed {
			return loc, ErrAccess
		}
	}
	loc.Accessible = true
	return loc, nil
}
