package register

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }

// The currency comes from the workspace; register rows are then read and
// written as candella_catalog, like the other tenant slices.
func withRegisterTx(ctx context.Context, db *pgxpool.Pool, tenantID string, fn func(pgx.Tx, string) error) error {
	return database.WithTenantTx(ctx, db, tenantID, func(tx pgx.Tx) error {
		var currency string
		if err := tx.QueryRow(ctx, `SELECT currency_code FROM tenants WHERE id=$1`, tenantID).Scan(&currency); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		return fn(tx, currency)
	})
}

// LockOpenShift returns the open shift on a register and holds it until the
// caller's transaction ends, so the shift cannot close half way through a
// sale and every committed sale is counted at closing. It runs inside a
// tenant transaction that has already resolved storeID for the caller.
func LockOpenShift(ctx context.Context, tx pgx.Tx, tenantID, storeID, registerID string) (string, error) {
	if registerID == "" {
		return "", ErrRegisterRequired
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM registers WHERE tenant_id=$1 AND store_id=$2 AND id=$3)`,
		tenantID, storeID, registerID).Scan(&exists); err != nil {
		return "", err
	}
	if !exists {
		return "", ErrRegisterNotFound
	}
	var shiftID string
	err := tx.QueryRow(ctx, `SELECT id FROM register_sessions
		WHERE tenant_id=$1 AND store_id=$2 AND register_id=$3 AND status='open' FOR SHARE`,
		tenantID, storeID, registerID).Scan(&shiftID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNoOpenShift
	}
	return shiftID, err
}

func (r *postgresRepository) ListRegisters(ctx context.Context, tenantID, _ string) ([]Register, error) {
	registers := []Register{}
	err := withRegisterTx(ctx, r.db, tenantID, func(tx pgx.Tx, _ string) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT r.id, r.store_id, r.name, s.id, s.opened_by, s.opener_name_snapshot, s.opened_at
			FROM registers r
			LEFT JOIN register_sessions s ON s.tenant_id=r.tenant_id AND s.register_id=r.id AND s.status='open'
			WHERE r.tenant_id=$1 AND r.store_id=$2
			ORDER BY lower(r.name), r.id`, tenantID, location.ID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var reg Register
			var shiftID, openedBy, opener *string
			var openedAt *time.Time
			if err := rows.Scan(&reg.ID, &reg.StoreID, &reg.Name, &shiftID, &openedBy, &opener, &openedAt); err != nil {
				return err
			}
			if shiftID != nil {
				reg.OpenSession = &ShiftSummary{ID: *shiftID, OpenedBy: *openedBy, OpenerName: *opener, OpenedAt: *openedAt}
			}
			registers = append(registers, reg)
		}
		return rows.Err()
	})
	return registers, err
}

func (r *postgresRepository) CreateRegister(ctx context.Context, tenantID string, in CreateRegisterInput) (*Register, error) {
	reg := &Register{ID: id.New(id.Register), Name: in.Name}
	err := withRegisterTx(ctx, r.db, tenantID, func(tx pgx.Tx, _ string) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		reg.StoreID = location.ID
		_, err = tx.Exec(ctx, `INSERT INTO registers (id, tenant_id, store_id, name) VALUES ($1,$2,$3,$4)`,
			reg.ID, tenantID, reg.StoreID, reg.Name)
		return err
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return nil, ErrRegisterConflict
	}
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func (r *postgresRepository) Open(ctx context.Context, tenantID string, actor Actor, in OpenInput) (*Shift, error) {
	var shift *Shift
	err := withRegisterTx(ctx, r.db, tenantID, func(tx pgx.Tx, currency string) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		var registerName string
		err = tx.QueryRow(ctx, `SELECT name FROM registers WHERE tenant_id=$1 AND store_id=$2 AND id=$3`,
			tenantID, location.ID, in.RegisterID).Scan(&registerName)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRegisterNotFound
		}
		if err != nil {
			return err
		}
		shiftID := id.New(id.RegisterSession)
		if _, err := tx.Exec(ctx, `INSERT INTO register_sessions
			(id, tenant_id, store_id, register_id, register_name_snapshot, opened_by, opener_name_snapshot, opening_cash_minor)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			shiftID, tenantID, location.ID, in.RegisterID, registerName, actor.UserID, actor.Name, in.OpeningCashMinor); err != nil {
			return err
		}
		shift, err = loadShift(ctx, tx, tenantID, shiftID, currency)
		return err
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return nil, ErrAlreadyOpen
	}
	if err != nil {
		return nil, err
	}
	return shift, nil
}

// visible applies the read rule: cashiers see their own shifts in stores they
// can use; owners and admins see every shift.
func visible(ctx context.Context, tx pgx.Tx, tenantID string, actor Actor, shift *Shift) error {
	if actor.manages() {
		return nil
	}
	if shift.OpenedBy != actor.UserID {
		return ErrNotFound
	}
	if _, err := store.Resolve(store.WithSelection(ctx, shift.StoreID, actor.UserID, actor.Role), tx, tenantID); err != nil {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Get(ctx context.Context, tenantID string, actor Actor, shiftID string) (*Shift, error) {
	var shift *Shift
	err := withRegisterTx(ctx, r.db, tenantID, func(tx pgx.Tx, currency string) error {
		var err error
		shift, err = loadShift(ctx, tx, tenantID, shiftID, currency)
		if err != nil {
			return err
		}
		return visible(ctx, tx, tenantID, actor, shift)
	})
	return shift, err
}

func (r *postgresRepository) Close(ctx context.Context, tenantID string, actor Actor, shiftID string, in CloseInput) (*Shift, error) {
	var shift *Shift
	err := withRegisterTx(ctx, r.db, tenantID, func(tx pgx.Tx, currency string) error {
		// Waits for sales in flight on this shift, which hold it FOR SHARE. A
		// closed shift cannot be locked at all (migration 0026), so it is not
		// found here; tell the caller it is closed if they may see it.
		var status, openedBy, storeID string
		var opening int64
		err := tx.QueryRow(ctx, `SELECT status, opened_by, store_id, opening_cash_minor FROM register_sessions
			WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenantID, shiftID).Scan(&status, &openedBy, &storeID, &opening)
		if errors.Is(err, pgx.ErrNoRows) {
			closed, err := loadShift(ctx, tx, tenantID, shiftID, currency)
			if err != nil {
				return err
			}
			if err := visible(ctx, tx, tenantID, actor, closed); err != nil {
				return err
			}
			return ErrClosed
		}
		if err != nil {
			return err
		}
		if !actor.manages() {
			if _, err := store.Resolve(store.WithSelection(ctx, storeID, actor.UserID, actor.Role), tx, tenantID); err != nil {
				return ErrNotFound
			}
			if openedBy != actor.UserID {
				return ErrNotYours
			}
		}
		if status != "open" {
			return ErrClosed
		}
		count, cashSales, err := shiftTotals(ctx, tx, tenantID, shiftID)
		if err != nil {
			return err
		}
		expected, variance, err := closingFigures(opening, cashSales, in)
		if err != nil {
			return err
		}
		var reason *string
		if in.VarianceReason != "" {
			reason = &in.VarianceReason
		}
		if _, err := tx.Exec(ctx, `UPDATE register_sessions SET status='closed', closed_by=$3, closer_name_snapshot=$4,
			closed_at=now(), sales_count=$5, cash_sales_minor=$6, expected_cash_minor=$7, counted_cash_minor=$8,
			variance_minor=$9, variance_reason=$10
			WHERE tenant_id=$1 AND id=$2 AND status='open'`,
			tenantID, shiftID, actor.UserID, actor.Name, count, cashSales, expected, in.CountedCashMinor,
			variance, reason); err != nil {
			return err
		}
		shift, err = loadShift(ctx, tx, tenantID, shiftID, currency)
		return err
	})
	return shift, err
}

func (r *postgresRepository) List(ctx context.Context, tenantID string, actor Actor, filter ShiftFilter) ([]Shift, bool, error) {
	shifts := []Shift{}
	hasMore := false
	err := withRegisterTx(ctx, r.db, tenantID, func(tx pgx.Tx, currency string) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT id FROM register_sessions
			WHERE tenant_id=$1 AND store_id=$2
			AND ($3::text = '' OR status = $3)
			AND ($4::bool OR opened_by = $5)
			ORDER BY opened_at DESC, id DESC LIMIT $6 OFFSET $7`,
			tenantID, location.ID, filter.Status, actor.manages(), actor.UserID, filter.Limit+1, filter.Offset)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		if len(ids) > filter.Limit {
			hasMore = true
			ids = ids[:filter.Limit]
		}
		for _, shiftID := range ids {
			shift, err := loadShift(ctx, tx, tenantID, shiftID, currency)
			if err != nil {
				return err
			}
			shifts = append(shifts, *shift)
		}
		return nil
	})
	return shifts, hasMore, err
}

// shiftTotals counts the completed cash sales taken into a shift.
func shiftTotals(ctx context.Context, tx pgx.Tx, tenantID, shiftID string) (count, cashMinor int64, err error) {
	err = tx.QueryRow(ctx, `SELECT count(*), COALESCE(sum(p.amount_minor), 0)
		FROM orders o JOIN payments p ON p.tenant_id=o.tenant_id AND p.order_id=o.id AND p.method='cash'
		WHERE o.tenant_id=$1 AND o.register_session_id=$2 AND o.status='completed'`,
		tenantID, shiftID).Scan(&count, &cashMinor)
	return count, cashMinor, err
}

func loadShift(ctx context.Context, tx pgx.Tx, tenantID, shiftID, currency string) (*Shift, error) {
	s := &Shift{CurrencyCode: currency}
	var salesCount, cashSales, expected *int64
	err := tx.QueryRow(ctx, `SELECT rs.id, rs.store_id, st.name, rs.register_id, rs.register_name_snapshot, rs.status,
		rs.opened_by, rs.opener_name_snapshot, rs.opened_at, rs.opening_cash_minor,
		rs.closed_by, rs.closer_name_snapshot, rs.closed_at, rs.sales_count, rs.cash_sales_minor,
		rs.expected_cash_minor, rs.counted_cash_minor, rs.variance_minor, rs.variance_reason
		FROM register_sessions rs JOIN stores st ON st.tenant_id=rs.tenant_id AND st.id=rs.store_id
		WHERE rs.tenant_id=$1 AND rs.id=$2`, tenantID, shiftID).
		Scan(&s.ID, &s.StoreID, &s.StoreName, &s.RegisterID, &s.RegisterName, &s.Status,
			&s.OpenedBy, &s.OpenerName, &s.OpenedAt, &s.OpeningCashMinor,
			&s.ClosedBy, &s.CloserName, &s.ClosedAt, &salesCount, &cashSales,
			&expected, &s.CountedCashMinor, &s.VarianceMinor, &s.VarianceReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if s.Status == "closed" {
		s.SalesCount, s.CashSalesMinor, s.ExpectedCashMinor = *salesCount, *cashSales, *expected
		return s, nil
	}
	s.SalesCount, s.CashSalesMinor, err = shiftTotals(ctx, tx, tenantID, shiftID)
	s.ExpectedCashMinor = s.OpeningCashMinor + s.CashSalesMinor
	return s, err
}
