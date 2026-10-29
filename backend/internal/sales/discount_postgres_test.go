package sales

import (
	"errors"
	"testing"
	"time"

	"candella-ecosystem/backend/internal/catalog"
	"candella-ecosystem/backend/internal/promotion"
	"candella-ecosystem/backend/internal/register"
	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/internal/tax"
	"candella-ecosystem/backend/pkg/id"
)

// errorCode is the stable code of a sales, promotion or register error.
func errorCode(err error) string {
	var s *Error
	var p *promotion.Error
	var r *register.Error
	switch {
	case errors.As(err, &s):
		return s.Code
	case errors.As(err, &p):
		return p.Code
	case errors.As(err, &r):
		return r.Code
	}
	return ""
}

func TestDiscountsPostgres(t *testing.T) {
	ctx, pool, appPool := openSalesTestDB(t)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}

	ownerID, cashierID := id.New(id.User), id.New(id.User)
	tenantA, tenantB := id.New(id.Tenant), id.New(id.Tenant)
	storeA, storeB := id.New(id.Store), id.New(id.Store)
	for _, u := range []string{ownerID, cashierID} {
		exec(`INSERT INTO users (id,email,full_name) VALUES ($1,$1 || '@example.test',$1)`, u)
	}
	for _, tenantID := range []string{tenantA, tenantB} {
		exec(`INSERT INTO tenants (id,name,slug,industry_type,currency_code,timezone) VALUES ($1,'Shop',$1,'pos_general','PKR','Asia/Karachi')`, tenantID)
	}
	exec(`INSERT INTO tenant_memberships (id,tenant_id,user_id,role,status) VALUES ($1,$2,$3,'owner','active'),($4,$2,$5,'member','active')`,
		id.New(id.TenantMembership), tenantA, ownerID, id.New(id.TenantMembership), cashierID)
	exec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$4,'Main',true)`, storeA, tenantA, storeB, tenantB)
	counter, counterB := id.New(id.Register), id.New(id.Register)
	exec(`INSERT INTO registers (id,tenant_id,store_id,name) VALUES ($1,$2,$3,'Counter 1'),($4,$5,$6,'Counter 1')`,
		counter, tenantA, storeA, counterB, tenantB, storeB)
	exec(`INSERT INTO register_sessions (id,tenant_id,store_id,register_id,register_name_snapshot,opened_by,opener_name_snapshot,opening_cash_minor)
		VALUES ($1,$2,$3,$4,'Counter 1',$5,'Owner',10000000),($6,$7,$8,$9,'Counter 1',$5,'Owner',0)`,
		id.New(id.RegisterSession), tenantA, storeA, counter, ownerID, id.New(id.RegisterSession), tenantB, storeB, counterB)

	promos := promotion.NewService(promotion.NewPostgresRepository(appPool))
	rates := tax.NewService(tax.NewPostgresRepository(appPool))
	products := catalog.NewService(catalog.NewPostgresRepository(appPool))
	svc := NewService(NewPostgresRepository(appPool))
	asOwner := store.WithSelection(ctx, storeA, ownerID, "owner")
	asCashier := store.WithSelection(ctx, storeA, cashierID, "member")

	gst, err := rates.Create(ctx, tenantA, "owner", tax.RateInput{Name: "GST", RateBps: 1800, Inclusive: true})
	if err != nil {
		t.Fatal(err)
	}
	salesTax, err := rates.Create(ctx, tenantA, "owner", tax.RateInput{Name: "Sales tax", RateBps: 1700})
	if err != nil {
		t.Fatal(err)
	}
	product := func(name string, price int64, rateID string) string {
		t.Helper()
		p, err := products.CreateProduct(asOwner, tenantA, ownerID, catalog.CreateProductInput{Name: name, TaxRateID: rateID,
			Variant: catalog.CreateVariantInput{PriceMinor: price}, InitialQuantity: ptr(int64(500)), StoreID: storeA})
		if err != nil {
			t.Fatal(err)
		}
		return p.Variants[0].ID
	}
	shirt := product("Shirt", 100000, "")              // Rs 1,000, untaxed
	shampoo := product("Shampoo", 100000, salesTax.ID) // Rs 1,000 + 17%
	chai := product("Chai", 118000, gst.ID)             // Rs 1,180 with 18% inside
	promo := func(in promotion.Input) *promotion.Promotion {
		t.Helper()
		p, err := promos.Create(ctx, tenantA, "owner", in)
		if err != nil {
			t.Fatalf("promotion %s: %v", in.Name, err)
		}
		return p
	}
	pkt := time.FixedZone("PKT", 5*60*60)
	yesterday := time.Now().In(pkt).AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := time.Now().In(pkt).AddDate(0, 0, 1).Format("2006-01-02")
	eid := promo(promotion.Input{Name: "Eid 10%", Kind: "percent", PercentBps: ptr(int64(1000)), AppliesTo: "order"})
	bigBill := promo(promotion.Input{Name: "Rs 500 off Rs 1,500", Kind: "amount", AmountMinor: ptr(int64(50000)),
		AppliesTo: "order", MinSubtotalMinor: 150000})
	shirtDeal := promo(promotion.Input{Name: "Rs 200 off each", Kind: "amount", AmountMinor: ptr(int64(20000)), AppliesTo: "item"})
	expired := promo(promotion.Input{Name: "Old sale", Kind: "percent", PercentBps: ptr(int64(500)), AppliesTo: "order", EndsOn: &yesterday})
	future := promo(promotion.Input{Name: "Next week", Kind: "percent", PercentBps: ptr(int64(500)), AppliesTo: "order", StartsOn: &tomorrow})
	huge := promo(promotion.Input{Name: "Rs 10,000 off", Kind: "amount", AmountMinor: ptr(int64(1000000)), AppliesTo: "order"})

	sell := func(asWho string, discount *DiscountInput, items ...CartItem) (*Sale, error) {
		c := asOwner
		userID, name := ownerID, "Owner"
		if asWho == "cashier" {
			c, userID, name = asCashier, cashierID, "Cashier"
		}
		sale, _, err := svc.Checkout(c, tenantA, userID, name, CheckoutInput{StoreID: storeA, RegisterID: counter,
			IdempotencyKey: "discount-sale-" + id.New(id.Order), CashReceivedMinor: 10_000_000,
			Items: items, Discount: discount})
		return sale, err
	}

	var eidSale *Sale
	t.Run("an order promotion comes off before tax", func(t *testing.T) {
		var err error
		eidSale, err = sell("owner", &DiscountInput{PromotionID: eid.ID},
			CartItem{VariantID: shirt, Quantity: 1}, CartItem{VariantID: shampoo, Quantity: 1})
		// 10% of 200000 = 20000, 10000 per line; 17% of the shampoo's 90000 = 15300.
		if err != nil || eidSale.SubtotalMinor != 200000 || eidSale.DiscountMinor != 20000 || eidSale.TaxMinor != 15300 ||
			eidSale.GrandTotalMinor != 195300 || *eidSale.DiscountName != "Eid 10%" || *eidSale.DiscountKind != "promotion" {
			t.Fatalf("eid sale = %+v err=%v", eidSale, err)
		}
		for _, item := range eidSale.Items {
			if item.DiscountMinor != 10000 {
				t.Fatalf("%s discount = %d, want its 10000 share", item.ProductName, item.DiscountMinor)
			}
		}
	})

	t.Run("an item promotion takes an amount off each unit", func(t *testing.T) {
		sale, err := sell("cashier", nil, CartItem{VariantID: shirt, Quantity: 2, PromotionID: shirtDeal.ID})
		if err != nil || sale.DiscountMinor != 40000 || sale.GrandTotalMinor != 160000 ||
			sale.Items[0].PromotionName == nil || *sale.Items[0].PromotionName != "Rs 200 off each" || sale.DiscountKind != nil {
			t.Fatalf("item promotion sale = %+v err=%v", sale, err)
		}
	})

	t.Run("a minimum bill is enforced", func(t *testing.T) {
		if _, err := sell("owner", &DiscountInput{PromotionID: bigBill.ID}, CartItem{VariantID: shirt, Quantity: 1}); errorCode(err) != "promotion_minimum" {
			t.Fatalf("below the minimum: %v", err)
		}
		sale, err := sell("owner", &DiscountInput{PromotionID: bigBill.ID}, CartItem{VariantID: shirt, Quantity: 2})
		if err != nil || sale.DiscountMinor != 50000 || sale.GrandTotalMinor != 150000 {
			t.Fatalf("at the minimum: %+v err=%v", sale, err)
		}
	})

	t.Run("invalid and incompatible discounts are refused, and nothing is sold", func(t *testing.T) {
		var before int64
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE tenant_id=$1`, tenantA).Scan(&before); err != nil {
			t.Fatal(err)
		}
		cases := []struct {
			name, code string
			discount   *DiscountInput
			items      []CartItem
		}{
			{"expired", "promotion_expired", &DiscountInput{PromotionID: expired.ID}, []CartItem{{VariantID: shirt, Quantity: 1}}},
			{"not started", "promotion_not_started", &DiscountInput{PromotionID: future.ID}, []CartItem{{VariantID: shirt, Quantity: 1}}},
			{"item promotion used on the order", "promotion_wrong_scope", &DiscountInput{PromotionID: shirtDeal.ID}, []CartItem{{VariantID: shirt, Quantity: 1}}},
			{"order promotion used on an item", "promotion_wrong_scope", nil, []CartItem{{VariantID: shirt, Quantity: 1, PromotionID: eid.ID}}},
			{"order and item together", "discount_combination", &DiscountInput{PromotionID: eid.ID},
				[]CartItem{{VariantID: shirt, Quantity: 1, PromotionID: shirtDeal.ID}}},
			{"discount above the total", "discount_too_large", &DiscountInput{PromotionID: huge.ID}, []CartItem{{VariantID: shirt, Quantity: 1}}},
			{"unknown promotion", "promotion_unavailable", &DiscountInput{PromotionID: "promo_AAAAAAAAAAAA"}, []CartItem{{VariantID: shirt, Quantity: 1}}},
			{"manual without a reason", "invalid_discount", &DiscountInput{PercentBps: 500}, []CartItem{{VariantID: shirt, Quantity: 1}}},
			{"promotion and manual together", "invalid_discount", &DiscountInput{PromotionID: eid.ID, PercentBps: 500, Reason: "Both"},
				[]CartItem{{VariantID: shirt, Quantity: 1}}},
		}
		for _, tc := range cases {
			if _, err := sell("owner", tc.discount, tc.items...); errorCode(err) != tc.code {
				t.Errorf("%s: err=%v, want %s", tc.name, err, tc.code)
			}
		}
		var after int64
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE tenant_id=$1`, tenantA).Scan(&after); err != nil || after != before {
			t.Fatalf("refused sales left orders: %d -> %d", before, after)
		}
	})

	t.Run("cashiers stay within the owner's manual limit", func(t *testing.T) {
		manual := func(bps int64) *DiscountInput { return &DiscountInput{PercentBps: bps, Reason: "Regular customer"} }
		if _, err := sell("cashier", manual(500), CartItem{VariantID: shirt, Quantity: 1}); errorCode(err) != "discount_needs_manager" {
			t.Fatalf("cashier discount with no limit set: %v", err)
		}
		if _, err := promos.SaveSettings(ctx, tenantA, "member", promotion.Settings{CashierLimitBps: 10000}); !errors.Is(err, promotion.ErrForbidden) {
			t.Fatalf("cashier raised their own limit: %v", err)
		}
		if _, err := promos.SaveSettings(ctx, tenantA, "owner", promotion.Settings{CashierLimitBps: 1000}); err != nil {
			t.Fatal(err)
		}
		sale, err := sell("cashier", manual(500), CartItem{VariantID: shirt, Quantity: 1})
		if err != nil || sale.DiscountMinor != 5000 || *sale.DiscountReason != "Regular customer" || *sale.DiscountName != "Manual discount 5%" {
			t.Fatalf("cashier 5%%: %+v err=%v", sale, err)
		}
		if _, err := sell("cashier", manual(1500), CartItem{VariantID: shirt, Quantity: 1}); errorCode(err) != "discount_needs_manager" {
			t.Fatalf("cashier above the limit: %v", err)
		}
		if sale, err := sell("owner", manual(10000), CartItem{VariantID: shirt, Quantity: 1}); err != nil || sale.GrandTotalMinor != 0 {
			t.Fatalf("owner 100%%: %+v err=%v", sale, err)
		}
	})

	t.Run("a refund pays back the discounted, taxed price", func(t *testing.T) {
		sale, err := sell("owner", &DiscountInput{PercentBps: 5000, Reason: "Damaged box"}, CartItem{VariantID: chai, Quantity: 2})
		// Half of 2 x 1180 = 118000 paid, 18000 of it tax.
		if err != nil || sale.GrandTotalMinor != 118000 || sale.TaxMinor != 18000 {
			t.Fatalf("discounted chai: %+v err=%v", sale, err)
		}
		ret, _, err := svc.Return(asOwner, tenantA, ownerID, "Owner", sale.ID, ReturnInput{StoreID: storeA, RegisterID: counter,
			IdempotencyKey: "discount-return-1", Reason: "Changed mind", Items: []CartItem{{VariantID: chai, Quantity: 1}}})
		if err != nil || ret.RefundMinor != 59000 || ret.TaxMinor != 9000 {
			t.Fatalf("refund %+v err=%v, want 59000 with 9000 tax", ret, err)
		}
	})

	t.Run("retries keep the same discount", func(t *testing.T) {
		in := CheckoutInput{StoreID: storeA, RegisterID: counter, IdempotencyKey: "discount-retry-01", CashReceivedMinor: 1_000_000,
			Items: []CartItem{{VariantID: shirt, Quantity: 1}}, Discount: &DiscountInput{PromotionID: eid.ID}}
		first, created, err := svc.Checkout(asOwner, tenantA, ownerID, "Owner", in)
		if err != nil || !created {
			t.Fatal(err)
		}
		again, created, err := svc.Checkout(asOwner, tenantA, ownerID, "Owner", in)
		if err != nil || created || again.ID != first.ID || again.DiscountMinor != 10000 {
			t.Fatalf("replay %+v created=%v err=%v", again, created, err)
		}
		in.Discount = &DiscountInput{PercentBps: 500, Reason: "Different"}
		if _, _, err := svc.Checkout(asOwner, tenantA, ownerID, "Owner", in); !errors.Is(err, ErrKeyReuse) {
			t.Fatalf("same key, different discount: %v", err)
		}
	})

	t.Run("changing a promotion never changes a past receipt", func(t *testing.T) {
		archived := "archived"
		if _, err := promos.Update(ctx, tenantA, "owner", eid.ID, promotion.Input{Name: "Eid 20%", Kind: "percent",
			PercentBps: ptr(int64(2000)), AppliesTo: "order"}, &archived); err != nil {
			t.Fatal(err)
		}
		receipt, err := svc.Receipt(ctx, tenantA, eidSale.ID)
		if err != nil || receipt.DiscountMinor != 20000 || *receipt.DiscountName != "Eid 10%" || *receipt.DiscountPercentBps != 1000 ||
			receipt.GrandTotalMinor != 195300 {
			t.Fatalf("past receipt changed: %+v err=%v", receipt, err)
		}
		if _, err := sell("owner", &DiscountInput{PromotionID: eid.ID}, CartItem{VariantID: shirt, Quantity: 1}); errorCode(err) != "promotion_unavailable" {
			t.Fatalf("archived promotion used: %v", err)
		}
	})

	t.Run("the daily report totals discounts", func(t *testing.T) {
		today := time.Now().In(pkt).Format("2006-01-02")
		report, err := svc.DailyReport(asOwner, tenantA, ReportFilter{FromDate: today, ToDate: today})
		// 20000 + 40000 + 50000 + 5000 + 100000 + 118000 + 10000
		if err != nil || report.Totals.DiscountMinor != 343000 {
			t.Fatalf("report discounts = %d err=%v, want 343000", report.Totals.DiscountMinor, err)
		}
	})

	t.Run("only managers run promotions, and only in their workspace", func(t *testing.T) {
		if _, err := promos.Create(ctx, tenantA, "member", promotion.Input{Name: "Mine", Kind: "percent",
			PercentBps: ptr(int64(100)), AppliesTo: "order"}); !errors.Is(err, promotion.ErrForbidden) {
			t.Fatalf("cashier created a promotion: %v", err)
		}
		list, err := promos.List(ctx, tenantB, true)
		if err != nil || len(list) != 0 {
			t.Fatalf("tenant B sees %d promotions, err=%v", len(list), err)
		}
		if _, _, err := svc.Checkout(store.WithSelection(ctx, storeB, ownerID, "owner"), tenantB, ownerID, "Owner",
			CheckoutInput{StoreID: storeB, RegisterID: counterB, IdempotencyKey: "foreign-promo-1", CashReceivedMinor: 100,
				Items: []CartItem{{VariantID: shirt, Quantity: 1}}, Discount: &DiscountInput{PromotionID: bigBill.ID}}); err == nil {
			t.Fatal("tenant B used tenant A's promotion")
		}
		for _, bad := range []promotion.Input{
			{Name: "", Kind: "percent", PercentBps: ptr(int64(100)), AppliesTo: "order"},
			{Name: "Zero", Kind: "percent", PercentBps: ptr(int64(0)), AppliesTo: "order"},
			{Name: "Over", Kind: "percent", PercentBps: ptr(int64(10001)), AppliesTo: "order"},
			{Name: "Backwards", Kind: "amount", AmountMinor: ptr(int64(100)), AppliesTo: "order", StartsOn: &tomorrow, EndsOn: &yesterday},
			{Name: "Nowhere", Kind: "amount", AmountMinor: ptr(int64(100)), AppliesTo: "shelf"},
		} {
			if _, err := promos.Create(ctx, tenantA, "owner", bad); !errors.Is(err, promotion.ErrInvalid) {
				t.Errorf("invalid promotion %q accepted: %v", bad.Name, err)
			}
		}
	})
}
