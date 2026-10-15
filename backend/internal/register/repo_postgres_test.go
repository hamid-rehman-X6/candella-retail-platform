package register_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/internal/register"
	"candella-ecosystem/backend/internal/sales"
	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/internal/workspace"
	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

// Only a named local disposable database or this repository's CI PostgreSQL
// service may be used; the test creates tenants and migrates the schema.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("REGISTER_TEST_DATABASE_URL")
	ciDatabase := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn = os.Getenv("DATABASE_URL")
		ciDatabase = true
	}
	if dsn == "" {
		t.Skip("set REGISTER_TEST_DATABASE_URL for a disposable candella_register_test database")
	}
	u, err := url.Parse(dsn)
	expectedDatabase := "/candella_register_test"
	if ciDatabase {
		expectedDatabase = "/candella"
	}
	if err != nil || u.Path != expectedDatabase || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") ||
		(ciDatabase && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("register integration test requires a local disposable PostgreSQL database")
	}
	return dsn
}

func migrate(ctx context.Context, t *testing.T, dsn string) {
	t.Helper()
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext('candella_integration_migrations'))`); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("migrations on disposable test DB: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_unlock(hashtext('candella_integration_migrations'))`); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterShiftsPostgres(t *testing.T) {
	dsn := testDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	migrate(ctx, t, dsn)

	// Seed as the login; run the code under test as the API role.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	appPool, err := database.Connect(ctx, dsn, database.AppRole)
	if err != nil {
		t.Fatal(err)
	}
	defer appPool.Close()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}

	ownerID, cashierID, otherCashierID := id.New(id.User), id.New(id.User), id.New(id.User)
	tenantA, tenantB := id.New(id.Tenant), id.New(id.Tenant)
	storeA, branchA, storeB := id.New(id.Store), id.New(id.Store), id.New(id.Store)
	counterA, counterB := id.New(id.Register), id.New(id.Register)
	variant := id.New(id.ProductVariant)
	for _, userID := range []string{ownerID, cashierID, otherCashierID} {
		exec(`INSERT INTO users (id,email,full_name) VALUES ($1,$1 || '@example.test',$1)`, userID)
	}
	for _, tenantID := range []string{tenantA, tenantB} {
		exec(`INSERT INTO tenants (id,name,slug,industry_type,currency_code) VALUES ($1,'Shop',$1,'pos_general','PKR')`, tenantID)
	}
	for _, m := range []struct{ userID, role string }{{ownerID, "owner"}, {cashierID, "member"}, {otherCashierID, "member"}} {
		exec(`INSERT INTO tenant_memberships (id,tenant_id,user_id,role,status) VALUES ($1,$2,$3,$4,'active')`,
			id.New(id.TenantMembership), tenantA, m.userID, m.role)
	}
	exec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$2,'Branch',false),($4,$5,'Main',true)`,
		storeA, tenantA, branchA, storeB, tenantB)
	exec(`INSERT INTO registers (id,tenant_id,store_id,name) VALUES ($1,$2,$3,'Counter 1'),($4,$5,$6,'Counter 1')`,
		counterA, tenantA, storeA, counterB, tenantB, storeB)
	productID := id.New(id.Product)
	exec(`INSERT INTO products (id,tenant_id,name) VALUES ($1,$2,'Chai')`, productID, tenantA)
	exec(`INSERT INTO product_variants (id,tenant_id,product_id,sku,price_minor) VALUES ($1,$2,$3,'CHAI',15000)`, variant, tenantA, productID)
	exec(`INSERT INTO stock_levels (id,tenant_id,store_id,variant_id,quantity) VALUES ($1,$2,$3,$4,1000)`, id.New(id.StockLevel), tenantA, storeA, variant)

	owner := register.Actor{UserID: ownerID, Name: "Owner", Role: "owner"}
	cashier := register.Actor{UserID: cashierID, Name: "Ayesha", Role: "member"}
	otherCashier := register.Actor{UserID: otherCashierID, Name: "Bilal", Role: "member"}
	in := func(actor register.Actor, storeID string) context.Context {
		return store.WithSelection(ctx, storeID, actor.UserID, actor.Role)
	}
	svc := register.NewService(register.NewPostgresRepository(appPool))
	salesSvc := sales.NewService(sales.NewPostgresRepository(appPool))
	sell := func(registerID, key string, quantity int64) (*sales.Sale, error) {
		sale, _, err := salesSvc.Checkout(store.WithSelection(ctx, storeA, cashierID, "member"), tenantA, cashierID, "Ayesha",
			sales.CheckoutInput{StoreID: storeA, RegisterID: registerID, IdempotencyKey: key, CashReceivedMinor: 15000 * quantity,
				Items: []sales.CartItem{{VariantID: variant, Quantity: quantity}}})
		return sale, err
	}

	// Opening a shift.
	registers, err := svc.ListRegisters(in(cashier, storeA), tenantA, storeA)
	if err != nil || len(registers) != 1 || registers[0].ID != counterA || registers[0].OpenSession != nil {
		t.Fatalf("registers before opening = %+v, err=%v", registers, err)
	}
	if _, err := sell(counterA, "before-open-1", 1); !errors.Is(err, register.ErrNoOpenShift) {
		t.Fatalf("sale before the shift opened: %v", err)
	}
	shift, err := svc.Open(in(cashier, storeA), tenantA, cashier, register.OpenInput{StoreID: storeA, RegisterID: counterA, OpeningCashMinor: 500000})
	if err != nil || shift.Status != "open" || shift.OpeningCashMinor != 500000 || shift.ExpectedCashMinor != 500000 ||
		shift.OpenerName != "Ayesha" || shift.RegisterName != "Counter 1" || shift.CurrencyCode != "PKR" {
		t.Fatalf("opened shift = %+v, err=%v", shift, err)
	}
	if _, err := svc.Open(in(owner, storeA), tenantA, owner, register.OpenInput{StoreID: storeA, RegisterID: counterA}); !errors.Is(err, register.ErrAlreadyOpen) {
		t.Fatalf("second open on the same counter: %v", err)
	}
	if _, err := svc.Open(in(owner, storeA), tenantA, owner, register.OpenInput{StoreID: storeA, RegisterID: counterB}); !errors.Is(err, register.ErrRegisterNotFound) {
		t.Fatalf("open on another workspace's counter: %v", err)
	}
	if _, err := svc.Open(in(cashier, branchA), tenantA, cashier, register.OpenInput{StoreID: branchA, RegisterID: counterA}); !errors.Is(err, store.ErrAccess) {
		t.Fatalf("cashier opened a shift in a store they cannot use: %v", err)
	}
	registers, err = svc.ListRegisters(in(otherCashier, storeA), tenantA, storeA)
	if err != nil || registers[0].OpenSession == nil || registers[0].OpenSession.ID != shift.ID || registers[0].OpenSession.OpenerName != "Ayesha" {
		t.Fatalf("other cashiers must see whose shift is open: %+v, err=%v", registers, err)
	}

	t.Run("concurrent opens on one counter", func(t *testing.T) {
		counter, err := svc.CreateRegister(in(owner, storeA), tenantA, owner, register.CreateRegisterInput{StoreID: storeA, Name: "Counter 2"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateRegister(in(owner, storeA), tenantA, owner, register.CreateRegisterInput{StoreID: storeA, Name: "counter 2"}); !errors.Is(err, register.ErrRegisterConflict) {
			t.Fatalf("duplicate counter name: %v", err)
		}
		var wg sync.WaitGroup
		results := make(chan error, 8)
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := svc.Open(in(owner, storeA), tenantA, owner, register.OpenInput{StoreID: storeA, RegisterID: counter.ID, OpeningCashMinor: 100})
				results <- err
			}()
		}
		wg.Wait()
		close(results)
		opened, refused := 0, 0
		for err := range results {
			switch {
			case err == nil:
				opened++
			case errors.Is(err, register.ErrAlreadyOpen):
				refused++
			default:
				t.Fatalf("concurrent open: %v", err)
			}
		}
		if opened != 1 || refused != 7 {
			t.Fatalf("opened=%d refused=%d, want 1 and 7", opened, refused)
		}
	})

	// Sales land in the shift, and the expected cash follows them.
	first, err := sell(counterA, "shift-sale-01", 2)
	if err != nil || first.ShiftID != shift.ID {
		t.Fatalf("first sale = %+v, err=%v", first, err)
	}
	if _, err := sell(counterA, "shift-sale-02", 1); err != nil {
		t.Fatal(err)
	}
	live, err := svc.Get(ctx, tenantA, cashier, shift.ID)
	if err != nil || live.SalesCount != 2 || live.CashSalesMinor != 45000 || live.ExpectedCashMinor != 545000 {
		t.Fatalf("live shift = %+v, err=%v", live, err)
	}
	if _, err := svc.Get(ctx, tenantA, otherCashier, shift.ID); !errors.Is(err, register.ErrNotFound) {
		t.Fatalf("another cashier read this drawer: %v", err)
	}
	if _, err := svc.Get(ctx, tenantB, owner, shift.ID); !errors.Is(err, register.ErrNotFound) {
		t.Fatalf("another workspace read this shift: %v", err)
	}

	// Closing.
	if _, err := svc.Close(ctx, tenantA, otherCashier, shift.ID, register.CloseInput{CountedCashMinor: 545000, ExpectedCashMinor: 545000}); !errors.Is(err, register.ErrNotYours) {
		t.Fatalf("another cashier closed this shift: %v", err)
	}
	if _, err := svc.Close(ctx, tenantB, owner, shift.ID, register.CloseInput{CountedCashMinor: 545000, ExpectedCashMinor: 545000}); !errors.Is(err, register.ErrNotFound) {
		t.Fatalf("another workspace closed this shift: %v", err)
	}
	if _, err := svc.Close(ctx, tenantA, cashier, shift.ID, register.CloseInput{CountedCashMinor: 530000, ExpectedCashMinor: 530000}); !errors.Is(err, register.ErrExpectedChanged) {
		t.Fatalf("close against a stale expected amount: %v", err)
	}
	if _, err := svc.Close(ctx, tenantA, cashier, shift.ID, register.CloseInput{CountedCashMinor: 540000, ExpectedCashMinor: 545000}); !errors.Is(err, register.ErrReasonRequired) {
		t.Fatalf("short drawer closed without a reason: %v", err)
	}
	closed, err := svc.Close(ctx, tenantA, cashier, shift.ID, register.CloseInput{CountedCashMinor: 540000, ExpectedCashMinor: 545000, VarianceReason: "Gave Rs 50 extra change"})
	if err != nil || closed.Status != "closed" || closed.SalesCount != 2 || closed.ExpectedCashMinor != 545000 ||
		*closed.CountedCashMinor != 540000 || *closed.VarianceMinor != -5000 || *closed.VarianceReason != "Gave Rs 50 extra change" ||
		*closed.CloserName != "Ayesha" || closed.ClosedAt == nil {
		t.Fatalf("closed shift = %+v, err=%v", closed, err)
	}
	if _, err := svc.Close(ctx, tenantA, owner, shift.ID, register.CloseInput{CountedCashMinor: 545000, ExpectedCashMinor: 545000}); !errors.Is(err, register.ErrClosed) {
		t.Fatalf("closed a shift twice: %v", err)
	}
	if _, err := sell(counterA, "after-close-1", 1); !errors.Is(err, register.ErrNoOpenShift) {
		t.Fatalf("sale after the shift closed: %v", err)
	}
	if replay, err := sell(counterA, "shift-sale-01", 2); err != nil || replay.ID != first.ID {
		t.Fatalf("a retry of a sale made before closing must still return its receipt: %+v, %v", replay, err)
	}

	t.Run("a closed shift cannot be changed", func(t *testing.T) {
		for query, wantCode := range map[string]string{
			`UPDATE register_sessions SET variance_reason='edited later' WHERE tenant_id=$1 AND id=$2`: "",
			`UPDATE register_sessions SET opening_cash_minor=0 WHERE tenant_id=$1 AND id=$2`:           "42501",
			`DELETE FROM register_sessions WHERE tenant_id=$1 AND id=$2`:                               "42501",
		} {
			var affected int64
			err := database.WithTenantTx(ctx, appPool, tenantA, func(tx pgx.Tx) error {
				tag, err := tx.Exec(ctx, query, tenantA, shift.ID)
				affected = tag.RowsAffected()
				return err
			})
			var pgErr *pgconn.PgError
			if wantCode == "" && (err != nil || affected != 0) {
				t.Errorf("%s: affected=%d err=%v, want 0 rows", query, affected, err)
			}
			if wantCode != "" && (!errors.As(err, &pgErr) || pgErr.Code != wantCode) {
				t.Errorf("%s: err=%v, want %s", query, err, wantCode)
			}
		}
		again, err := svc.Get(ctx, tenantA, owner, shift.ID)
		if err != nil || *again.VarianceReason != "Gave Rs 50 extra change" || again.OpeningCashMinor != 500000 {
			t.Fatalf("closed shift changed: %+v, err=%v", again, err)
		}
	})

	t.Run("closing waits for sales in flight", func(t *testing.T) {
		race, err := svc.Open(in(cashier, storeA), tenantA, cashier, register.OpenInput{StoreID: storeA, RegisterID: counterA, OpeningCashMinor: 1000})
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		saleErrs := make(chan error, 12)
		for i := range 12 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := sell(counterA, fmt.Sprintf("race-sale-%03d", i), 1)
				saleErrs <- err
			}()
		}
		var final *register.Shift
		for attempt := 0; attempt < 50 && final == nil; attempt++ {
			current, err := svc.Get(ctx, tenantA, cashier, race.ID)
			if err != nil {
				t.Fatal(err)
			}
			final, err = svc.Close(ctx, tenantA, cashier, race.ID, register.CloseInput{
				CountedCashMinor: current.ExpectedCashMinor, ExpectedCashMinor: current.ExpectedCashMinor})
			if err != nil && !errors.Is(err, register.ErrExpectedChanged) {
				t.Fatal(err)
			}
		}
		wg.Wait()
		close(saleErrs)
		if final == nil {
			t.Fatal("could not close the shift while sales were running")
		}
		sold := int64(0)
		for err := range saleErrs {
			switch {
			case err == nil:
				sold++
			case !errors.Is(err, register.ErrNoOpenShift):
				t.Fatalf("racing sale: %v", err)
			}
		}
		var linked int64
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE tenant_id=$1 AND register_session_id=$2`, tenantA, race.ID).Scan(&linked); err != nil {
			t.Fatal(err)
		}
		if final.SalesCount != linked || linked != sold || final.ExpectedCashMinor != 1000+15000*sold {
			t.Fatalf("closing counted %d sales (expected %d) but %d were committed and %d linked",
				final.SalesCount, final.ExpectedCashMinor, sold, linked)
		}
	})

	t.Run("a new workspace starts with a counter", func(t *testing.T) {
		ws := &workspace.Workspace{Name: "New shop", Slug: "new-" + ownerID[4:], IndustryType: "pos_general",
			CurrencyCode: "PKR", Timezone: "Asia/Karachi"}
		if err := workspace.NewPostgresRepository(appPool).CreateWithOwner(ctx, ws, ownerID); err != nil {
			t.Fatal(err)
		}
		var counters int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM registers r JOIN stores s ON s.tenant_id=r.tenant_id AND s.id=r.store_id
			WHERE r.tenant_id=$1 AND s.is_default AND r.name='Counter 1'`, ws.ID).Scan(&counters); err != nil || counters != 1 {
			t.Fatalf("new workspace counters=%d err=%v", counters, err)
		}
	})

	t.Run("history", func(t *testing.T) {
		all, _, err := svc.List(in(owner, storeA), tenantA, owner, register.ShiftFilter{StoreID: storeA})
		if err != nil || len(all) != 3 {
			t.Fatalf("owner history = %d shifts, err=%v; want 3", len(all), err)
		}
		own, _, err := svc.List(in(cashier, storeA), tenantA, cashier, register.ShiftFilter{StoreID: storeA})
		if err != nil || len(own) != 2 {
			t.Fatalf("cashier history = %d shifts, err=%v; want their 2", len(own), err)
		}
		for _, s := range own {
			if s.OpenedBy != cashierID {
				t.Fatalf("cashier saw %s's shift", s.OpenerName)
			}
		}
		closedOnly, more, err := svc.List(in(owner, storeA), tenantA, owner, register.ShiftFilter{StoreID: storeA, Status: "closed", Limit: 1})
		if err != nil || len(closedOnly) != 1 || !more || closedOnly[0].Status != "closed" {
			t.Fatalf("closed page = %+v more=%v err=%v", closedOnly, more, err)
		}
		if _, _, err := svc.List(in(owner, storeA), tenantB, owner, register.ShiftFilter{StoreID: storeA}); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("another workspace listed this store's shifts: %v", err)
		}
	})
}
