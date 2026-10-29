package sales

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"candella-ecosystem/backend/internal/catalog"
	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/internal/tax"
	"candella-ecosystem/backend/pkg/id"
)

func TestTaxPostgres(t *testing.T) {
	ctx, pool, appPool := openSalesTestDB(t)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}

	ownerID, tenantA, tenantB := id.New(id.User), id.New(id.Tenant), id.New(id.Tenant)
	storeA, storeB := id.New(id.Store), id.New(id.Store)
	exec(`INSERT INTO users (id,email,full_name) VALUES ($1,$1 || '@example.test','Owner')`, ownerID)
	for _, tenantID := range []string{tenantA, tenantB} {
		exec(`INSERT INTO tenants (id,name,slug,industry_type,currency_code,timezone) VALUES ($1,'Shop',$1,'pos_general','PKR','Asia/Karachi')`, tenantID)
	}
	exec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$4,'Main',true)`, storeA, tenantA, storeB, tenantB)
	counter, shift := id.New(id.Register), id.New(id.RegisterSession)
	exec(`INSERT INTO registers (id,tenant_id,store_id,name) VALUES ($1,$2,$3,'Counter 1')`, counter, tenantA, storeA)
	exec(`INSERT INTO register_sessions (id,tenant_id,store_id,register_id,register_name_snapshot,opened_by,opener_name_snapshot,opening_cash_minor)
		VALUES ($1,$2,$3,$4,'Counter 1',$5,'Owner',0)`, shift, tenantA, storeA, counter, ownerID)

	rates := tax.NewService(tax.NewPostgresRepository(appPool))
	products := catalog.NewService(catalog.NewPostgresRepository(appPool))
	svc := NewService(NewPostgresRepository(appPool))
	atMain := store.WithSelection(ctx, storeA, ownerID, "owner")

	gst, err := rates.Create(ctx, tenantA, "owner", tax.RateInput{Name: "GST 18%", RateBps: 1800, Inclusive: true})
	if err != nil {
		t.Fatal(err)
	}
	salesTax, err := rates.Create(ctx, tenantA, "admin", tax.RateInput{Name: "Sales tax 17%", RateBps: 1700, Inclusive: false})
	if err != nil {
		t.Fatal(err)
	}
	product := func(name string, priceMinor int64, rateID string) string {
		t.Helper()
		p, err := products.CreateProduct(atMain, tenantA, ownerID, catalog.CreateProductInput{
			Name: name, TaxRateID: rateID, Variant: catalog.CreateVariantInput{PriceMinor: priceMinor},
			InitialQuantity: ptr(int64(100)), StoreID: storeA})
		if err != nil {
			t.Fatalf("product %s: %v", name, err)
		}
		return p.Variants[0].ID
	}
	chai := product("Chai", 45000, gst.ID)           // Rs 450, 18% inside the price
	shampoo := product("Shampoo", 89000, salesTax.ID) // Rs 890, 17% added
	water := product("Water", 9000, "")               // Rs 90, not taxed
	sell := func(key string, items ...CartItem) *Sale {
		t.Helper()
		sale, _, err := svc.Checkout(atMain, tenantA, ownerID, "Owner", CheckoutInput{StoreID: storeA, RegisterID: counter,
			IdempotencyKey: key, CashReceivedMinor: 10_000_000, Items: items})
		if err != nil {
			t.Fatalf("sale %s: %v", key, err)
		}
		return sale
	}

	t.Run("an untaxed cart charges no tax", func(t *testing.T) {
		sale := sell("tax-untaxed-01", CartItem{VariantID: water, Quantity: 2})
		if sale.TaxMinor != 0 || sale.SubtotalMinor != 18000 || sale.GrandTotalMinor != 18000 || sale.Items[0].TaxName != nil {
			t.Fatalf("untaxed sale = %+v", sale)
		}
	})

	var mixed *Sale
	t.Run("a mixed cart adds only exclusive tax to the total", func(t *testing.T) {
		mixed = sell("tax-mixed-0001",
			CartItem{VariantID: chai, Quantity: 1}, CartItem{VariantID: shampoo, Quantity: 1}, CartItem{VariantID: water, Quantity: 1})
		// Chai: 45000 x 18/118 = 6864.4 -> 6864, inside the price.
		// Shampoo: 89000 x 17% = 15130, added on top.
		if mixed.SubtotalMinor != 143000 || mixed.TaxMinor != 21994 || mixed.GrandTotalMinor != 158130 {
			t.Fatalf("mixed totals subtotal=%d tax=%d grand=%d, want 143000, 21994, 158130",
				mixed.SubtotalMinor, mixed.TaxMinor, mixed.GrandTotalMinor)
		}
		want := map[string]struct {
			name string
			bps  int64
			incl bool
			tax  int64
		}{chai: {"GST 18%", 1800, true, 6864}, shampoo: {"Sales tax 17%", 1700, false, 15130}}
		for _, item := range mixed.Items {
			w, taxed := want[item.VariantID]
			switch {
			case !taxed && (item.TaxName != nil || item.TaxMinor != 0):
				t.Errorf("%s should be untaxed: %+v", item.ProductName, item)
			case taxed && (item.TaxName == nil || *item.TaxName != w.name || *item.TaxRateBps != w.bps ||
				*item.TaxInclusive != w.incl || item.TaxMinor != w.tax):
				t.Errorf("%s tax snapshot = %+v, want %+v", item.ProductName, item, w)
			}
		}
		var paid int64
		if err := pool.QueryRow(ctx, `SELECT amount_minor FROM payments WHERE tenant_id=$1 AND order_id=$2`, tenantA, mixed.ID).Scan(&paid); err != nil || paid != 158130 {
			t.Fatalf("payment=%d err=%v, want the grand total", paid, err)
		}
	})

	t.Run("each line is rounded once, half up", func(t *testing.T) {
		tiny := product("Sweet", 33, salesTax.ID) // 33 x 17% = 5.61 -> 6
		sale := sell("tax-rounding-1", CartItem{VariantID: tiny, Quantity: 1})
		if sale.TaxMinor != 6 || sale.GrandTotalMinor != 39 {
			t.Fatalf("rounded sale tax=%d grand=%d, want 6, 39", sale.TaxMinor, sale.GrandTotalMinor)
		}
	})

	t.Run("changing a rate never changes a past sale", func(t *testing.T) {
		name, bps := "GST 25%", int64(2500)
		if _, err := rates.Update(ctx, tenantA, "owner", gst.ID, tax.RateUpdate{Name: &name, RateBps: &bps}); err != nil {
			t.Fatal(err)
		}
		receipt, err := svc.Receipt(ctx, tenantA, mixed.ID)
		if err != nil || receipt.TaxMinor != 21994 || receipt.GrandTotalMinor != 158130 {
			t.Fatalf("past sale changed: %+v err=%v", receipt, err)
		}
		for _, item := range receipt.Items {
			if item.VariantID == chai && (*item.TaxName != "GST 18%" || *item.TaxRateBps != 1800 || item.TaxMinor != 6864) {
				t.Fatalf("past chai line changed: %+v", item)
			}
		}
		next := sell("tax-new-rate-1", CartItem{VariantID: chai, Quantity: 1})
		// 45000 x 25/125 = 9000
		if next.TaxMinor != 9000 || *next.Items[0].TaxName != "GST 25%" {
			t.Fatalf("new sale did not use the new rate: %+v", next.Items[0])
		}
	})

	t.Run("refunds return the tax that was charged", func(t *testing.T) {
		three := sell("tax-refund-sale", CartItem{VariantID: shampoo, Quantity: 3})
		// 3 x 89000 = 267000; 17% = 45390; customer paid 312390.
		if three.GrandTotalMinor != 312390 {
			t.Fatalf("three shampoos grand = %d", three.GrandTotalMinor)
		}
		var refunded, refundedTax int64
		for i := range 3 {
			ret, _, err := svc.Return(atMain, tenantA, ownerID, "Owner", three.ID, ReturnInput{StoreID: storeA, RegisterID: counter,
				IdempotencyKey: fmt.Sprintf("tax-return-%d", i), Reason: "Allergic", Items: []CartItem{{VariantID: shampoo, Quantity: 1}}})
			if err != nil {
				t.Fatal(err)
			}
			if ret.Items[0].LineRefundMinor < 104129 || ret.Items[0].LineRefundMinor > 104131 {
				t.Fatalf("one shampoo refund = %d, want about 104130", ret.Items[0].LineRefundMinor)
			}
			refunded += ret.RefundMinor
			refundedTax += ret.TaxMinor
		}
		if refunded != 312390 || refundedTax != 45390 {
			t.Fatalf("three single returns refunded %d with %d tax, want exactly 312390 and 45390", refunded, refundedTax)
		}
	})

	t.Run("the daily report shows tax charged and refunded", func(t *testing.T) {
		today := time.Now().In(time.FixedZone("PKT", 5*60*60)).Format("2006-01-02")
		report, err := svc.DailyReport(atMain, tenantA, ReportFilter{FromDate: today, ToDate: today})
		if err != nil {
			t.Fatal(err)
		}
		// 0 + 21994 + 6 + 9000 + 45390 charged; 45390 refunded.
		if report.Totals.TaxMinor != 76390 || report.Totals.RefundedTaxMinor != 45390 || report.Totals.NetTaxMinor != 31000 {
			t.Fatalf("report tax = %d charged, %d refunded, %d net; want 76390, 45390, 31000",
				report.Totals.TaxMinor, report.Totals.RefundedTaxMinor, report.Totals.NetTaxMinor)
		}
	})

	t.Run("a rate in use cannot be archived", func(t *testing.T) {
		archived := "archived"
		_, err := rates.Update(ctx, tenantA, "owner", salesTax.ID, tax.RateUpdate{Status: &archived})
		if e, ok := tax.AsError(err); !ok || e.Code != "tax_rate_in_use" {
			t.Fatalf("archived a rate in use: %v", err)
		}
		spare, err := rates.Create(ctx, tenantA, "owner", tax.RateInput{Name: "Old rate", RateBps: 500})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := rates.Update(ctx, tenantA, "owner", spare.ID, tax.RateUpdate{Status: &archived}); err != nil {
			t.Fatalf("archive an unused rate: %v", err)
		}
		if _, err := products.CreateProduct(atMain, tenantA, ownerID, catalog.CreateProductInput{
			Name: "Late", TaxRateID: spare.ID, Variant: catalog.CreateVariantInput{PriceMinor: 100}}); !errors.Is(err, catalog.ErrTaxRate) {
			t.Fatalf("product given an archived rate: %v", err)
		}
	})

	t.Run("a product can be moved to no tax", func(t *testing.T) {
		none := ""
		list, err := products.ListProducts(atMain, tenantA, "Water", 10, 0)
		if err != nil || len(list) != 1 {
			t.Fatalf("water lookup: %v %v", list, err)
		}
		p, err := products.UpdateProduct(atMain, tenantA, list[0].ID, catalog.UpdateProductInput{TaxRateID: &salesTax.ID})
		if err != nil || p.TaxRateName == nil || *p.TaxRateName != "Sales tax 17%" {
			t.Fatalf("give water a rate: %+v %v", p, err)
		}
		p, err = products.UpdateProduct(atMain, tenantA, list[0].ID, catalog.UpdateProductInput{TaxRateID: &none})
		if err != nil || p.TaxRateID != nil {
			t.Fatalf("remove water's rate: %+v %v", p, err)
		}
	})

	t.Run("only owners and admins manage rates", func(t *testing.T) {
		if _, err := rates.Create(ctx, tenantA, "member", tax.RateInput{Name: "Sneaky", RateBps: 0}); !errors.Is(err, tax.ErrForbidden) {
			t.Fatalf("member created a rate: %v", err)
		}
		bps := int64(0)
		if _, err := rates.Update(ctx, tenantA, "member", gst.ID, tax.RateUpdate{RateBps: &bps}); !errors.Is(err, tax.ErrForbidden) {
			t.Fatalf("member changed a rate: %v", err)
		}
		for _, bad := range []tax.RateInput{{Name: "", RateBps: 100}, {Name: "Too high", RateBps: 10001}, {Name: "Negative", RateBps: -1}} {
			if _, err := rates.Create(ctx, tenantA, "owner", bad); !errors.Is(err, tax.ErrInvalid) {
				t.Fatalf("invalid rate %+v accepted: %v", bad, err)
			}
		}
		if _, err := rates.Create(ctx, tenantA, "owner", tax.RateInput{Name: "gst 25%", RateBps: 100}); !errors.Is(err, tax.ErrConflict) {
			t.Fatalf("duplicate active name: %v", err)
		}
	})

	t.Run("another workspace cannot see or use these rates", func(t *testing.T) {
		list, err := rates.List(ctx, tenantB, true)
		if err != nil || len(list) != 0 {
			t.Fatalf("tenant B sees rates %+v err=%v", list, err)
		}
		atB := store.WithSelection(ctx, storeB, ownerID, "owner")
		if _, err := products.CreateProduct(atB, tenantB, ownerID, catalog.CreateProductInput{
			Name: "Borrowed", TaxRateID: gst.ID, Variant: catalog.CreateVariantInput{PriceMinor: 100}}); !errors.Is(err, catalog.ErrTaxRate) {
			t.Fatalf("tenant B used tenant A's rate: %v", err)
		}
		name := "Hijacked"
		if _, err := rates.Update(ctx, tenantB, "owner", gst.ID, tax.RateUpdate{Name: &name}); !errors.Is(err, tax.ErrNotFound) {
			t.Fatalf("tenant B renamed tenant A's rate: %v", err)
		}
	})
}

func ptr[T any](v T) *T { return &v }
