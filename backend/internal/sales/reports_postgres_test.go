package sales

import (
	"errors"
	"testing"
	"time"

	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/pkg/id"
)

func TestDailyReportPostgres(t *testing.T) {
	ctx, admin, app := openSalesTestDB(t)
	must := func(query string, args ...any) {
		t.Helper()
		if _, err := admin.Exec(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	owner, tenantA, tenantB := id.New(id.User), id.New(id.Tenant), id.New(id.Tenant)
	main, branch, other := id.New(id.Store), id.New(id.Store), id.New(id.Store)
	product, variant := id.New(id.Product), id.New(id.ProductVariant)
	must(`INSERT INTO users (id,email,full_name) VALUES ($1,$1 || '@example.test','Owner')`, owner)
	for _, tenantID := range []string{tenantA, tenantB} {
		must(`INSERT INTO tenants (id,name,slug,industry_type,currency_code,timezone) VALUES ($1,'Shop',$1,'pos_general','PKR','Asia/Karachi')`, tenantID)
	}
	must(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$2,'Branch',false),($4,$5,'Other',true)`, main, tenantA, branch, other, tenantB)
	must(`INSERT INTO products (id,tenant_id,name) VALUES ($1,$2,'Soap')`, product, tenantA)
	must(`INSERT INTO product_variants (id,tenant_id,product_id,sku,price_minor) VALUES ($1,$2,$3,'SOAP',5000)`, variant, tenantA, product)
	register, shift := id.New(id.Register), id.New(id.RegisterSession)
	must(`INSERT INTO registers (id,tenant_id,store_id,name) VALUES ($1,$2,$3,'Counter')`, register, tenantA, main)
	must(`INSERT INTO register_sessions (id,tenant_id,store_id,register_id,register_name_snapshot,opened_by,opener_name_snapshot,opening_cash_minor)
	  VALUES ($1,$2,$3,$4,'Counter',$5,'Owner',10000)`, shift, tenantA, main, register, owner)
	orders := []struct {
		id, storeID, number, at string
		amount, quantity        int64
	}{
		{id.New(id.Order), main, "POS-000001", "2026-10-19T18:45:00Z", 5000, 1},
		{id.New(id.Order), main, "POS-000002", "2026-10-19T19:15:00Z", 15000, 3},
		{id.New(id.Order), main, "POS-000003", "2026-10-20T19:30:00Z", 2000, 0}, // Oct 21 in Karachi
		{id.New(id.Order), branch, "POS-000004", "2026-10-19T19:15:00Z", 7000, 0},
	}
	for _, sale := range orders {
		at, err := time.Parse(time.RFC3339, sale.at)
		if err != nil {
			t.Fatal(err)
		}
		must(`INSERT INTO orders (id,tenant_id,store_id,order_number,idempotency_key,request_hash,currency_code,subtotal_minor,grand_total_minor,created_by,store_name_snapshot,created_at)
		  VALUES ($1,$2,$3,$4,$1,repeat('a',64),'PKR',$5,$5,$6,'Main',$7)`, sale.id, tenantA, sale.storeID, sale.number, sale.amount, owner, at)
		must(`INSERT INTO payments (id,tenant_id,order_id,method,status,amount_minor,cash_received_minor,change_minor,currency_code,paid_at)
		  VALUES ($1,$2,$3,'cash','completed',$4,$4,0,'PKR',$5)`, id.New(id.Payment), tenantA, sale.id, sale.amount, at)
		if sale.quantity > 0 {
			must(`INSERT INTO order_items (id,tenant_id,order_id,variant_id,product_name_snapshot,variant_name_snapshot,sku_snapshot,quantity,unit_price_minor,line_total_minor)
			  VALUES ($1,$2,$3,$4,'Soap','Standard','SOAP',$5,5000,$6)`, id.New(id.OrderItem), tenantA, sale.id, variant, sale.quantity, sale.amount)
		}
	}
	returnID := id.New(id.Return)
	must(`INSERT INTO returns (id,tenant_id,store_id,order_id,register_session_id,return_number,idempotency_key,request_hash,reason,refund_minor,currency_code,created_by,processor_name_snapshot,created_at)
	  VALUES ($1,$2,$3,$4,$5,'RET-000001',$1,repeat('r',64),'Returned',5000,'PKR',$6,'Owner','2026-10-20T18:30:00Z')`, returnID, tenantA, main, orders[0].id, shift, owner)
	must(`INSERT INTO return_items (id,tenant_id,return_id,order_id,variant_id,quantity,unit_price_minor,line_refund_minor)
	  VALUES ($1,$2,$3,$4,$5,1,5000,5000)`, id.New(id.ReturnItem), tenantA, returnID, orders[0].id, variant)
	must(`INSERT INTO stock_movements (id,tenant_id,store_id,variant_id,kind,delta,quantity_before,quantity_after,created_at)
	  VALUES ($1,$2,$3,$4,'receive',5,0,5,'2026-10-19T19:20:00Z')`, id.New(id.StockMovement), tenantA, main, variant)
	must(`INSERT INTO stock_movements (id,tenant_id,store_id,variant_id,kind,delta,quantity_before,quantity_after,order_id,created_at)
	  VALUES ($1,$2,$3,$4,'sale',-3,5,2,$5,'2026-10-19T19:25:00Z')`, id.New(id.StockMovement), tenantA, main, variant, orders[1].id)
	must(`INSERT INTO stock_movements (id,tenant_id,store_id,variant_id,kind,delta,quantity_before,quantity_after,return_id,created_at)
	  VALUES ($1,$2,$3,$4,'return',1,2,3,$5,'2026-10-20T18:30:00Z')`, id.New(id.StockMovement), tenantA, main, variant, returnID)
	svc := NewService(NewPostgresRepository(app))
	report, err := svc.DailyReport(store.WithSelection(ctx, main, owner, "admin"), tenantA, ReportFilter{"2026-10-19", "2026-10-20"})
	if err != nil {
		t.Fatal(err)
	}
	if report.StoreID != main || report.CurrencyCode != "PKR" || report.Timezone != "Asia/Karachi" || len(report.Days) != 2 {
		t.Fatalf("report identity/days: %+v", report)
	}
	if report.Days[0].SalesCount != 1 || report.Days[0].GrossCashMinor != 5000 || report.Days[0].RefundsMinor != 0 ||
		report.Days[1].SalesCount != 1 || report.Days[1].GrossCashMinor != 15000 || report.Days[1].RefundsMinor != 5000 || report.Days[1].NetSalesMinor != 10000 {
		t.Fatalf("business-day totals: %+v", report.Days)
	}
	if report.Totals.SalesCount != 2 || report.Totals.GrossCashMinor != 20000 || report.Totals.RefundsMinor != 5000 || report.Totals.NetSalesMinor != 15000 {
		t.Fatalf("totals: %+v", report.Totals)
	}
	if len(report.TopProducts) != 1 || report.TopProducts[0].UnitsSold != 4 || report.TopProducts[0].GrossSalesMinor != 20000 {
		t.Fatalf("top products: %+v", report.TopProducts)
	}
	if len(report.StockMovements) != 3 {
		t.Fatalf("stock movements: %+v", report.StockMovements)
	}
	if _, err := svc.DailyReport(store.WithSelection(ctx, other, owner, "admin"), tenantA, ReportFilter{"2026-10-19", "2026-10-20"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-tenant store: %v", err)
	}
	if b, err := svc.DailyReport(store.WithSelection(ctx, other, owner, "admin"), tenantB, ReportFilter{"2026-10-19", "2026-10-20"}); err != nil || b.Totals.GrossCashMinor != 0 {
		t.Fatalf("tenant B: %+v %v", b, err)
	}
	if branchReport, err := svc.DailyReport(store.WithSelection(ctx, branch, owner, "admin"), tenantA, ReportFilter{"2026-10-19", "2026-10-20"}); err != nil || branchReport.Totals.GrossCashMinor != 7000 {
		t.Fatalf("branch: %+v %v", branchReport, err)
	}
}
