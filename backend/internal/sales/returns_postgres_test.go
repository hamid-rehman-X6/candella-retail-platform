package sales

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"candella-ecosystem/backend/internal/register"
	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

func TestReturnsPostgres(t *testing.T) {
	ctx, pool, appPool := openSalesTestDB(t)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	text := func(query string, args ...any) string {
		t.Helper()
		var v string
		if err := pool.QueryRow(ctx, query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return v
	}
	count := func(query string, args ...any) int64 {
		t.Helper()
		var n int64
		if err := pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return n
	}

	ownerID, tenantA, tenantB := id.New(id.User), id.New(id.Tenant), id.New(id.Tenant)
	storeA, branchA := id.New(id.Store), id.New(id.Store)
	shirt, hat, socks := id.New(id.ProductVariant), id.New(id.ProductVariant), id.New(id.ProductVariant)
	exec(`INSERT INTO users (id,email,full_name) VALUES ($1,$1 || '@example.test','Owner')`, ownerID)
	for _, tenantID := range []string{tenantA, tenantB} {
		exec(`INSERT INTO tenants (id,name,slug,industry_type,currency_code) VALUES ($1,'Shop',$1,'pos_general','PKR')`, tenantID)
	}
	exec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$2,'Branch',false)`, storeA, tenantA, branchA)
	for _, v := range []struct {
		id, sku string
		price   int64
	}{{shirt, "SHIRT", 15000}, {hat, "HAT", 5000}, {socks, "SOCKS", 1000}} {
		productID := id.New(id.Product)
		exec(`INSERT INTO products (id,tenant_id,name) VALUES ($1,$2,$3)`, productID, tenantA, v.sku)
		exec(`INSERT INTO product_variants (id,tenant_id,product_id,sku,price_minor) VALUES ($1,$2,$3,$4,$5)`, v.id, tenantA, productID, v.sku, v.price)
		exec(`INSERT INTO stock_levels (id,tenant_id,store_id,variant_id,quantity) VALUES ($1,$2,$3,$4,20)`, id.New(id.StockLevel), tenantA, storeA, v.id)
	}
	openShift := func(storeID string, opening int64) string {
		t.Helper()
		registerID := id.New(id.Register)
		exec(`INSERT INTO registers (id,tenant_id,store_id,name) VALUES ($1,$2,$3,$1)`, registerID, tenantA, storeID)
		exec(`INSERT INTO register_sessions (id,tenant_id,store_id,register_id,register_name_snapshot,opened_by,opener_name_snapshot,opening_cash_minor)
			VALUES ($1,$2,$3,$4,'Counter',$5,'Owner',$6)`, id.New(id.RegisterSession), tenantA, storeID, registerID, ownerID, opening)
		return registerID
	}
	counter := openShift(storeA, 0)
	secondCounter := openShift(storeA, 100000)
	branchCounter := openShift(branchA, 100000)

	svc := NewService(NewPostgresRepository(appPool))
	shifts := register.NewService(register.NewPostgresRepository(appPool))
	owner := register.Actor{UserID: ownerID, Name: "Owner", Role: "owner"}
	atMain := store.WithSelection(ctx, storeA, ownerID, "owner")
	sale, _, err := svc.Checkout(atMain, tenantA, ownerID, "Owner", CheckoutInput{StoreID: storeA, RegisterID: counter,
		IdempotencyKey: "sale-for-returns", CashReceivedMinor: 60000,
		Items: []CartItem{{VariantID: shirt, Quantity: 3}, {VariantID: hat, Quantity: 2}}})
	if err != nil || sale.GrandTotalMinor != 55000 {
		t.Fatalf("sale=%+v err=%v", sale, err)
	}
	ret := func(ctxStore context.Context, registerID, key, reason string, items ...CartItem) (*Return, bool, error) {
		return svc.Return(ctxStore, tenantA, ownerID, "Owner", sale.ID,
			ReturnInput{StoreID: storeA, RegisterID: registerID, IdempotencyKey: key, Reason: reason, Items: items})
	}
	stock := func(variantID string) int64 {
		return count(`SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`, tenantA, storeA, variantID)
	}
	expected := func() int64 {
		t.Helper()
		shiftID := text(`SELECT id FROM register_sessions WHERE tenant_id=$1 AND register_id=$2 AND status='open'`, tenantA, counter)
		s, err := shifts.Get(ctx, tenantA, owner, shiftID)
		if err != nil {
			t.Fatal(err)
		}
		return s.ExpectedCashMinor
	}

	// Partial return: stock comes back, the refund leaves the drawer, the sale is untouched.
	first, created, err := ret(atMain, counter, "return-shirt-1", "Wrong size", CartItem{VariantID: shirt, Quantity: 1})
	if err != nil || !created || first.ReturnNumber != "RET-000001" || first.RefundMinor != 15000 ||
		len(first.Items) != 1 || first.Items[0].SKU != "SHIRT" || first.OrderNumber != sale.OrderNumber || first.Reason != "Wrong size" {
		t.Fatalf("first return=%+v created=%v err=%v", first, created, err)
	}
	if got := stock(shirt); got != 18 {
		t.Fatalf("shirt stock after returning 1 of 3 sold from 20 = %d, want 18", got)
	}
	if n := count(`SELECT count(*) FROM stock_movements WHERE tenant_id=$1 AND return_id=$2 AND kind='return'
		AND delta=1 AND quantity_before=17 AND quantity_after=18 AND order_id IS NULL`, tenantA, first.ID); n != 1 {
		t.Fatalf("return movements=%d, want one +1 movement", n)
	}
	if got := expected(); got != 40000 {
		t.Fatalf("expected drawer after a 15000 refund on 55000 of sales = %d", got)
	}
	receipt, err := svc.Receipt(ctx, tenantA, sale.ID)
	if err != nil || receipt.GrandTotalMinor != 55000 || receipt.RefundedMinor != 15000 || len(receipt.Returns) != 1 {
		t.Fatalf("receipt after return=%+v err=%v", receipt, err)
	}
	for _, item := range receipt.Items {
		if want := map[string]int64{shirt: 1, hat: 0}[item.VariantID]; item.ReturnedQuantity != want {
			t.Fatalf("%s returned=%d, want %d", item.SKU, item.ReturnedQuantity, want)
		}
	}

	t.Run("a retry returns the same return", func(t *testing.T) {
		again, created, err := ret(atMain, counter, "return-shirt-1", "Wrong size", CartItem{VariantID: shirt, Quantity: 1})
		if err != nil || created || again.ID != first.ID || stock(shirt) != 18 {
			t.Fatalf("replay=%+v created=%v err=%v stock=%d", again, created, err, stock(shirt))
		}
		if _, _, err := ret(atMain, counter, "return-shirt-1", "Changed my mind", CartItem{VariantID: shirt, Quantity: 1}); !errors.Is(err, ErrReturnKeyReuse) {
			t.Fatalf("same key, different return: %v", err)
		}
	})

	t.Run("refused returns change nothing", func(t *testing.T) {
		returns, sequence := count(`SELECT count(*) FROM returns WHERE tenant_id=$1`, tenantA),
			count(`SELECT current_return_number FROM document_sequences WHERE tenant_id=$1`, tenantA)
		drawer := expected()
		shortCounter := openShift(storeA, 1000)
		cases := []struct {
			name, code string
			call       func() error
		}{
			{"more than remains", "return_exceeds_sale", func() error {
				_, _, err := ret(atMain, counter, "return-too-many", "Wrong size", CartItem{VariantID: shirt, Quantity: 3})
				return err
			}},
			{"a valid line with an excessive one", "return_exceeds_sale", func() error {
				_, _, err := ret(atMain, counter, "return-mixed-01", "Wrong size", CartItem{VariantID: hat, Quantity: 1}, CartItem{VariantID: shirt, Quantity: 5})
				return err
			}},
			{"an item not in the sale", "return_item_not_in_sale", func() error {
				_, _, err := ret(atMain, counter, "return-not-sold", "Wrong size", CartItem{VariantID: socks, Quantity: 1})
				return err
			}},
			{"the wrong store", "return_wrong_store", func() error {
				_, _, err := svc.Return(store.WithSelection(ctx, branchA, ownerID, "owner"), tenantA, ownerID, "Owner", sale.ID,
					ReturnInput{StoreID: branchA, RegisterID: branchCounter, IdempotencyKey: "return-branch-1", Reason: "Wrong size",
						Items: []CartItem{{VariantID: hat, Quantity: 1}}})
				return err
			}},
			{"a drawer without enough cash", "drawer_cash_short", func() error {
				_, _, err := ret(atMain, shortCounter, "return-short-01", "Wrong size", CartItem{VariantID: hat, Quantity: 1})
				return err
			}},
			{"a counter without an open shift", "shift_required", func() error {
				closedCounter := id.New(id.Register)
				exec(`INSERT INTO registers (id,tenant_id,store_id,name) VALUES ($1,$2,$3,$1)`, closedCounter, tenantA, storeA)
				_, _, err := ret(atMain, closedCounter, "return-no-shift", "Wrong size", CartItem{VariantID: hat, Quantity: 1})
				return err
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.call()
				var saleErr *Error
				var regErr *register.Error
				code := ""
				if errors.As(err, &saleErr) {
					code = saleErr.Code
				} else if errors.As(err, &regErr) {
					code = regErr.Code
				}
				if code != tc.code {
					t.Fatalf("err=%v, want %s", err, tc.code)
				}
			})
		}
		if count(`SELECT count(*) FROM returns WHERE tenant_id=$1`, tenantA) != returns ||
			count(`SELECT current_return_number FROM document_sequences WHERE tenant_id=$1`, tenantA) != sequence ||
			stock(shirt) != 18 || stock(hat) != 18 || expected() != drawer {
			t.Fatal("a refused return left a trace: return, number, stock or drawer changed")
		}
	})

	t.Run("concurrent returns cannot over-return", func(t *testing.T) {
		// Two counters, so the returns do not simply queue on one drawer's lock.
		counters := []string{counter, secondCounter}
		var wg sync.WaitGroup
		results := make(chan error, 6)
		for i := range 6 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _, err := ret(atMain, counters[i%2], fmt.Sprintf("return-hat-race-%d", i), "Wrong colour", CartItem{VariantID: hat, Quantity: 1})
				results <- err
			}()
		}
		wg.Wait()
		close(results)
		ok, refused := 0, 0
		for err := range results {
			var saleErr *Error
			switch {
			case err == nil:
				ok++
			case errors.As(err, &saleErr) && saleErr.Code == "return_exceeds_sale":
				refused++
			default:
				t.Fatalf("racing return: %v", err)
			}
		}
		if ok != 2 || refused != 4 || stock(hat) != 20 {
			t.Fatalf("2 hats sold: ok=%d refused=%d stock=%d, want 2, 4, 20", ok, refused, stock(hat))
		}
	})

	t.Run("concurrent retries make one return", func(t *testing.T) {
		var wg sync.WaitGroup
		ids := make(chan string, 2)
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				r, _, err := ret(atMain, counter, "return-shirt-retry", "Stained", CartItem{VariantID: shirt, Quantity: 1})
				if err != nil {
					t.Error(err)
					return
				}
				ids <- r.ID
			}()
		}
		wg.Wait()
		close(ids)
		a, b := <-ids, <-ids
		if a == "" || a != b || stock(shirt) != 19 {
			t.Fatalf("retries made %q and %q, shirt stock %d", a, b, stock(shirt))
		}
	})

	t.Run("a fully returned sale accepts no more", func(t *testing.T) {
		if _, _, err := ret(atMain, counter, "return-shirt-last", "Faulty", CartItem{VariantID: shirt, Quantity: 1}); err != nil {
			t.Fatal(err)
		}
		_, _, err := ret(atMain, counter, "return-shirt-more", "Faulty", CartItem{VariantID: shirt, Quantity: 1})
		var saleErr *Error
		if !errors.As(err, &saleErr) || saleErr.Code != "return_exceeds_sale" {
			t.Fatalf("return beyond the sale: %v", err)
		}
		receipt, err := svc.Receipt(ctx, tenantA, sale.ID)
		if err != nil || receipt.RefundedMinor != 55000 || receipt.GrandTotalMinor != 55000 || len(receipt.Returns) != 5 {
			t.Fatalf("fully returned receipt=%+v err=%v", receipt, err)
		}
		if stock(shirt) != 20 || stock(hat) != 20 {
			t.Fatalf("after a full return stock=%d/%d, want 20/20", stock(shirt), stock(hat))
		}
	})

	t.Run("returns, the sale and the ledger cannot be edited", func(t *testing.T) {
		for _, query := range []string{
			`UPDATE returns SET refund_minor = 1 WHERE tenant_id = $1`,
			`DELETE FROM returns WHERE tenant_id = $1`,
			`DELETE FROM return_items WHERE tenant_id = $1`,
			`UPDATE stock_movements SET delta = 0 WHERE tenant_id = $1 AND kind = 'return'`,
			`UPDATE orders SET grand_total_minor = 0 WHERE tenant_id = $1`,
		} {
			err := database.WithTenantTx(ctx, appPool, tenantA, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, query, tenantA)
				return err
			})
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
				t.Errorf("%s: err=%v, want permission denied", query, err)
			}
		}
	})

	t.Run("another workspace cannot see or refund", func(t *testing.T) {
		if _, _, err := svc.Return(ctx, tenantB, ownerID, "Owner", sale.ID,
			ReturnInput{IdempotencyKey: "foreign-return", Reason: "Wrong size", RegisterID: counter,
				Items: []CartItem{{VariantID: shirt, Quantity: 1}}}); !errors.Is(err, store.ErrNotFound) && !errors.Is(err, ErrNotFound) {
			t.Fatalf("tenant B refunded tenant A's sale: %v", err)
		}
		if _, err := svc.ReturnReceipt(ctx, tenantB, first.ID); !errors.Is(err, ErrReturnNotFound) {
			t.Fatalf("tenant B read tenant A's return: %v", err)
		}
	})

	t.Run("closing the shift counts the refunds", func(t *testing.T) {
		// The 55000 of refunds came out of the two drawers that paid them, each
		// closing at opening + sales - its own refunds.
		var refunds, payouts int64
		for _, c := range []struct {
			registerID     string
			opening, sales int64
		}{{counter, 0, 55000}, {secondCounter, 100000, 0}} {
			shiftID := text(`SELECT id FROM register_sessions WHERE tenant_id=$1 AND register_id=$2 AND status='open'`, tenantA, c.registerID)
			open, err := shifts.Get(ctx, tenantA, owner, shiftID)
			if err != nil {
				t.Fatal(err)
			}
			closed, err := shifts.Close(ctx, tenantA, owner, shiftID, register.CloseInput{
				CountedCashMinor: open.ExpectedCashMinor, ExpectedCashMinor: open.ExpectedCashMinor})
			if err != nil || closed.CashSalesMinor != c.sales ||
				closed.ExpectedCashMinor != c.opening+c.sales-closed.CashRefundsMinor || *closed.VarianceMinor != 0 {
				t.Fatalf("closed shift=%+v err=%v", closed, err)
			}
			refunds += closed.CashRefundsMinor
			payouts += closed.RefundsCount
		}
		if refunds != 55000 || payouts != 5 {
			t.Fatalf("shifts recorded %d refunds totalling %d, want 5 totalling 55000", payouts, refunds)
		}
	})
}
