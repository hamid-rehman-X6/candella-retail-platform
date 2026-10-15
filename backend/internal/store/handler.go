package store

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
	"candella-ecosystem/backend/pkg/response"
)

type Handler struct {
	db      *pgxpool.Pool
	auditor *audit.Recorder
	log     *slog.Logger
}

func NewHandler(db *pgxpool.Pool, auditor *audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{db, auditor, log}
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var e *Error
	if errors.As(err, &e) {
		response.Error(w, e.Status, e.Code, e.Message)
		return
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		response.Error(w, ErrConflict.Status, ErrConflict.Code, ErrConflict.Message)
		return
	}
	h.log.Error("store request failed", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal_error", "Could not save the store. Try again.")
}

func actor(r *http.Request) (string, string, string) {
	t, _ := tenant.FromContext(r.Context())
	u, _ := auth.UserFromContext(r.Context())
	return t.TenantID, u.ID, t.Role
}

func canManage(role string) bool { return role == "owner" || role == "admin" }

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

func (h *Handler) tx(r *http.Request, tenantID string, fn func(pgx.Tx) error) error {
	return database.WithTenantTx(r.Context(), h.db, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		return fn(tx)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, role := actor(r)
	stores := []Location{}
	err := h.tx(r, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(r.Context(), `SELECT s.id,s.name,s.is_default,
			(s.is_default OR $3::bool OR EXISTS(SELECT 1 FROM store_access a WHERE a.tenant_id=s.tenant_id AND a.store_id=s.id AND a.user_id=$2)) AS accessible
			FROM stores s WHERE s.tenant_id=$1 ORDER BY s.is_default DESC,s.name,s.id`, tenantID, userID, canManage(role))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var loc Location
			if err := rows.Scan(&loc.ID, &loc.Name, &loc.IsDefault, &loc.Accessible); err != nil {
				return err
			}
			if loc.Accessible || canManage(role) {
				stores = append(stores, loc)
			}
		}
		return rows.Err()
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"stores": stores})
}

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 100 {
		return "", ErrInvalid
	}
	return name, nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, role := actor(r)
	if !canManage(role) {
		h.writeErr(w, ErrAccess)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := decode(r, &input); err != nil {
		h.writeErr(w, err)
		return
	}
	name, err := validName(input.Name)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	loc := Location{ID: id.New(id.Store), Name: name, Accessible: true}
	err = h.tx(r, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,$3,false)`, loc.ID, tenantID, name); err != nil {
			return err
		}
		return CreateDefaultRegister(r.Context(), tx, tenantID, loc.ID)
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &userID, Action: "store.created", EntityType: "store", EntityID: loc.ID})
	response.JSON(w, http.StatusCreated, map[string]any{"store": loc})
}

func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, role := actor(r)
	if !canManage(role) {
		h.writeErr(w, ErrAccess)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := decode(r, &input); err != nil {
		h.writeErr(w, err)
		return
	}
	name, err := validName(input.Name)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	storeID := chi.URLParam(r, "storeID")
	err = h.tx(r, tenantID, func(tx pgx.Tx) error {
		cmd, err := tx.Exec(r.Context(), `UPDATE stores SET name=$3 WHERE tenant_id=$1 AND id=$2`, tenantID, storeID, name)
		if err == nil && cmd.RowsAffected() == 0 {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &userID, Action: "store.renamed", EntityType: "store", EntityID: storeID})
	response.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *Handler) SetDefault(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, role := actor(r)
	if !canManage(role) {
		h.writeErr(w, ErrAccess)
		return
	}
	storeID := chi.URLParam(r, "storeID")
	err := database.WithTenantTx(r.Context(), h.db, tenantID, func(tx pgx.Tx) error {
		// Serializes default switches, including concurrent requests.
		if _, err := tx.Exec(r.Context(), `SELECT id FROM tenants WHERE id=$1 FOR UPDATE`, tenantID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM stores WHERE tenant_id=$1 AND id=$2)`, tenantID, storeID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		if _, err := tx.Exec(r.Context(), `UPDATE stores SET is_default=false WHERE tenant_id=$1 AND is_default`, tenantID); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `UPDATE stores SET is_default=true WHERE tenant_id=$1 AND id=$2`, tenantID, storeID)
		return err
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &userID, Action: "store.default_changed", EntityType: "store", EntityID: storeID})
	response.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *Handler) Access(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, role := actor(r)
	if !canManage(role) {
		h.writeErr(w, ErrAccess)
		return
	}
	var input struct {
		UserID  string `json:"userId"`
		Allowed bool   `json:"allowed"`
	}
	if err := decode(r, &input); err != nil || input.UserID == "" {
		h.writeErr(w, ErrInvalid)
		return
	}
	storeID := chi.URLParam(r, "storeID")
	err := database.WithTenantTx(r.Context(), h.db, tenantID, func(tx pgx.Tx) error {
		var memberRole string
		if err := tx.QueryRow(r.Context(), `SELECT role FROM tenant_memberships WHERE tenant_id=$1 AND user_id=$2 AND status='active'`, tenantID, input.UserID).Scan(&memberRole); errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if memberRole != "member" {
			return ErrInvalid
		}
		if _, err := tx.Exec(r.Context(), `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		var isDefault bool
		if err := tx.QueryRow(r.Context(), `SELECT is_default FROM stores WHERE tenant_id=$1 AND id=$2`, tenantID, storeID).Scan(&isDefault); errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if isDefault {
			return ErrInvalid
		}
		if input.Allowed {
			_, err := tx.Exec(r.Context(), `INSERT INTO store_access (tenant_id,store_id,user_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, tenantID, storeID, input.UserID)
			return err
		}
		_, err := tx.Exec(r.Context(), `DELETE FROM store_access WHERE tenant_id=$1 AND store_id=$2 AND user_id=$3`, tenantID, storeID, input.UserID)
		return err
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	h.auditor.Record(r.Context(), audit.Entry{UserID: &userID, Action: "store.access_changed", EntityType: "store", EntityID: storeID, Metadata: map[string]any{"userId": input.UserID, "allowed": input.Allowed}})
	response.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *Handler) ListAccess(w http.ResponseWriter, r *http.Request) {
	tenantID, _, role := actor(r)
	if !canManage(role) {
		h.writeErr(w, ErrAccess)
		return
	}
	storeID := chi.URLParam(r, "storeID")
	userIDs := []string{}
	err := h.tx(r, tenantID, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM stores WHERE tenant_id=$1 AND id=$2)`, tenantID, storeID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		rows, err := tx.Query(r.Context(), `SELECT user_id FROM store_access WHERE tenant_id=$1 AND store_id=$2 ORDER BY user_id`, tenantID, storeID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var userID string
			if err := rows.Scan(&userID); err != nil {
				return err
			}
			userIDs = append(userIDs, userID)
		}
		return rows.Err()
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"userIds": userIDs})
}

func Routes(r chi.Router, h *Handler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Use(requireTenant)
		g.Get("/stores", h.List)
		g.Post("/stores", h.Create)
		g.Patch("/stores/{storeID}", h.Rename)
		g.Post("/stores/{storeID}/default", h.SetDefault)
		g.Get("/stores/{storeID}/access", h.ListAccess)
		g.Post("/stores/{storeID}/access", h.Access)
	})
}
