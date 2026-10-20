package purchase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db} }
func (r *postgresRepository) tx(ctx context.Context, tenantID string, fn func(pgx.Tx, string) error) error {
	return database.WithTenantTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
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
func hash(value any) string {
	payload, _ := json.Marshal(value)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
func createHash(userID string, in CreateInput) string {
	return hash(struct {
		UserID string
		Input  CreateInput
	}{userID, in})
}
func receiveHash(userID, orderID string, in ReceiveInput) string {
	return hash(struct {
		UserID, OrderID string
		Items           []ReceiveLineInput
	}{userID, orderID, in.Items})
}
func lockKey(ctx context.Context, tx pgx.Tx, tenantID, key string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1),hashtext($2))`, tenantID, key)
	return err
}

func (r *postgresRepository) Create(ctx context.Context, tenantID, userID string, in CreateInput) (*Order, bool, error) {
	var out *Order
	created := false
	requestHash := createHash(userID, in)
	err := r.tx(ctx, tenantID, func(tx pgx.Tx, currency string) error {
		location, err := store.Resolve(store.WithSelection(ctx, in.StoreID, userID, "admin"), tx, tenantID)
		if err != nil {
			return err
		}
		if err := lockKey(ctx, tx, tenantID, in.IdempotencyKey); err != nil {
			return err
		}
		var existingID, existingHash string
		err = tx.QueryRow(ctx, `SELECT id,request_hash FROM purchase_orders WHERE tenant_id=$1 AND idempotency_key=$2`, tenantID, in.IdempotencyKey).Scan(&existingID, &existingHash)
		if err == nil {
			if existingHash != requestHash {
				return ErrKeyReuse
			}
			out, err = loadOrder(ctx, tx, tenantID, existingID)
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var supplierName, status string
		err = tx.QueryRow(ctx, `SELECT name,status FROM suppliers WHERE tenant_id=$1 AND id=$2 FOR SHARE`, tenantID, in.SupplierID).Scan(&supplierName, &status)
		if errors.Is(err, pgx.ErrNoRows) || status == "archived" {
			return ErrSupplierUnavailable
		}
		if err != nil {
			return err
		}
		items := make([]Item, 0, len(in.Items))
		var total int64
		for _, line := range in.Items {
			var item Item
			item.VariantID = line.VariantID
			item.Quantity = line.Quantity
			item.UnitCostMinor = line.UnitCostMinor
			item.LineCostMinor = line.Quantity * line.UnitCostMinor
			err = tx.QueryRow(ctx, `SELECT p.name,v.name,v.sku,p.status FROM product_variants v JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
				WHERE v.tenant_id=$1 AND v.id=$2 FOR SHARE OF p,v`, tenantID, line.VariantID).Scan(&item.ProductName, &item.VariantName, &item.SKU, &status)
			if errors.Is(err, pgx.ErrNoRows) || status == "archived" {
				return ErrVariantUnavailable
			}
			if err != nil {
				return err
			}
			total += item.LineCostMinor
			items = append(items, item)
		}
		var number int64
		if err := tx.QueryRow(ctx, `INSERT INTO document_sequences (tenant_id,current_purchase_number) VALUES ($1,1)
			ON CONFLICT (tenant_id) DO UPDATE SET current_purchase_number=document_sequences.current_purchase_number+1
			RETURNING current_purchase_number`, tenantID).Scan(&number); err != nil {
			return err
		}
		orderID := id.New(id.PurchaseOrder)
		_, err = tx.Exec(ctx, `INSERT INTO purchase_orders (id,tenant_id,store_id,supplier_id,number,idempotency_key,request_hash,
			store_name_snapshot,supplier_name_snapshot,currency_code,total_cost_minor,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, orderID, tenantID, location.ID, in.SupplierID,
			fmt.Sprintf("PO-%06d", number), in.IdempotencyKey, requestHash, location.Name, supplierName, currency, total, userID)
		if err != nil {
			return err
		}
		for _, item := range items {
			_, err = tx.Exec(ctx, `INSERT INTO purchase_order_items (id,tenant_id,purchase_order_id,variant_id,product_name_snapshot,
				variant_name_snapshot,sku_snapshot,quantity,unit_cost_minor,line_cost_minor)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id.New(id.PurchaseItem), tenantID, orderID, item.VariantID,
				item.ProductName, item.VariantName, item.SKU, item.Quantity, item.UnitCostMinor, item.LineCostMinor)
			if err != nil {
				return err
			}
		}
		out, err = loadOrder(ctx, tx, tenantID, orderID)
		if err == nil {
			created = true
		}
		return err
	})
	return out, created, err
}

func (r *postgresRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]Summary, error) {
	out := []Summary{}
	err := r.tx(ctx, tenantID, func(tx pgx.Tx, _ string) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT id,number,store_id,supplier_name_snapshot,status,currency_code,total_cost_minor,created_at
			FROM purchase_orders WHERE tenant_id=$1 AND store_id=$2 ORDER BY created_at DESC,id DESC LIMIT $3 OFFSET $4`, tenantID, location.ID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var s Summary
			if err := rows.Scan(&s.ID, &s.Number, &s.StoreID, &s.SupplierName, &s.Status, &s.CurrencyCode, &s.TotalCostMinor, &s.CreatedAt); err != nil {
				return err
			}
			out = append(out, s)
		}
		return rows.Err()
	})
	return out, err
}
func (r *postgresRepository) Find(ctx context.Context, tenantID, orderID string) (*Order, error) {
	var out *Order
	err := r.tx(ctx, tenantID, func(tx pgx.Tx, _ string) error {
		var err error
		out, err = loadOrder(ctx, tx, tenantID, orderID)
		return err
	})
	return out, err
}
func (r *postgresRepository) Approve(ctx context.Context, tenantID, orderID, userID string) (*Order, bool, error) {
	var out *Order
	changed := false
	err := r.tx(ctx, tenantID, func(tx pgx.Tx, _ string) error {
		var status, supplierID string
		err := tx.QueryRow(ctx, `SELECT status,supplier_id FROM purchase_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenantID, orderID).Scan(&status, &supplierID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == "draft" {
			var supplierStatus string
			if err := tx.QueryRow(ctx, `SELECT status FROM suppliers WHERE tenant_id=$1 AND id=$2 FOR SHARE`, tenantID, supplierID).Scan(&supplierStatus); err != nil {
				return err
			}
			if supplierStatus != "active" {
				return ErrSupplierUnavailable
			}
			_, err = tx.Exec(ctx, `UPDATE purchase_orders SET status='approved',approved_by=$3,approved_at=now() WHERE tenant_id=$1 AND id=$2`, tenantID, orderID, userID)
			if err != nil {
				return err
			}
			changed = true
		}
		out, err = loadOrder(ctx, tx, tenantID, orderID)
		return err
	})
	return out, changed, err
}

func (r *postgresRepository) Receive(ctx context.Context, tenantID, orderID, userID string, in ReceiveInput) (*Order, *Receipt, bool, error) {
	var out *Order
	var receipt *Receipt
	created := false
	requestHash := receiveHash(userID, orderID, in)
	err := r.tx(ctx, tenantID, func(tx pgx.Tx, _ string) error {
		if err := lockKey(ctx, tx, tenantID, in.IdempotencyKey); err != nil {
			return err
		}
		var existingID, existingOrder, existingHash string
		err := tx.QueryRow(ctx, `SELECT id,purchase_order_id,request_hash FROM goods_receipts WHERE tenant_id=$1 AND idempotency_key=$2`, tenantID, in.IdempotencyKey).Scan(&existingID, &existingOrder, &existingHash)
		if err == nil {
			if existingOrder != orderID || existingHash != requestHash {
				return ErrKeyReuse
			}
			out, err = loadOrder(ctx, tx, tenantID, orderID)
			if err != nil {
				return err
			}
			for i := range out.Receipts {
				if out.Receipts[i].ID == existingID {
					receipt = &out.Receipts[i]
					return nil
				}
			}
			return ErrNotFound
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var status, storeID string
		err = tx.QueryRow(ctx, `SELECT status,store_id FROM purchase_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenantID, orderID).Scan(&status, &storeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == "draft" {
			return ErrApprovalRequired
		}
		if status == "received" {
			return ErrFullyReceived
		}
		type line struct {
			variantID                  string
			quantity, unitCost, before int64
		}
		lines := make([]line, 0, len(in.Items))
		var total int64
		for _, input := range in.Items {
			var ordered, already, cost int64
			var productStatus string
			err = tx.QueryRow(ctx, `SELECT pi.quantity,pi.received_quantity,pi.unit_cost_minor,p.status FROM purchase_order_items pi
				JOIN product_variants v ON v.tenant_id=pi.tenant_id AND v.id=pi.variant_id
				JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
				WHERE pi.tenant_id=$1 AND pi.purchase_order_id=$2 AND pi.variant_id=$3 FOR UPDATE OF pi FOR SHARE OF p`, tenantID, orderID, input.VariantID).
				Scan(&ordered, &already, &cost, &productStatus)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOverReceive
			}
			if err != nil {
				return err
			}
			if productStatus != "active" {
				return ErrVariantUnavailable
			}
			if input.Quantity > ordered-already {
				return ErrOverReceive
			}
			if cost > maxCostMinor/input.Quantity || cost*input.Quantity > maxCostMinor-total {
				return ErrInvalid
			}
			total += cost * input.Quantity
			lines = append(lines, line{variantID: input.VariantID, quantity: input.Quantity, unitCost: cost})
		}
		// Stock is locked in variant ID order for every receipt, avoiding deadlocks.
		for i := range lines {
			_, err = tx.Exec(ctx, `INSERT INTO stock_levels (id,tenant_id,store_id,variant_id,quantity) VALUES ($1,$2,$3,$4,0)
				ON CONFLICT (tenant_id,store_id,variant_id) DO NOTHING`, id.New(id.StockLevel), tenantID, storeID, lines[i].variantID)
			if err != nil {
				return err
			}
			err = tx.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3 FOR UPDATE`, tenantID, storeID, lines[i].variantID).Scan(&lines[i].before)
			if err != nil {
				return err
			}
			if lines[i].quantity > 1_000_000_000-lines[i].before {
				return ErrStockLimit
			}
		}
		receiptID := id.New(id.GoodsReceipt)
		_, err = tx.Exec(ctx, `INSERT INTO goods_receipts (id,tenant_id,purchase_order_id,store_id,idempotency_key,request_hash,total_cost_minor,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, receiptID, tenantID, orderID, storeID, in.IdempotencyKey, requestHash, total, userID)
		if err != nil {
			return err
		}
		for _, line := range lines {
			_, err = tx.Exec(ctx, `INSERT INTO goods_receipt_items (id,tenant_id,goods_receipt_id,purchase_order_id,variant_id,quantity,unit_cost_minor,line_cost_minor)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, id.New(id.GoodsReceiptItem), tenantID, receiptID, orderID, line.variantID, line.quantity, line.unitCost, line.unitCost*line.quantity)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE purchase_order_items SET received_quantity=received_quantity+$4
				WHERE tenant_id=$1 AND purchase_order_id=$2 AND variant_id=$3`, tenantID, orderID, line.variantID, line.quantity)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE stock_levels SET quantity=$4 WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`, tenantID, storeID, line.variantID, line.before+line.quantity)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `INSERT INTO stock_movements (id,tenant_id,store_id,variant_id,kind,delta,quantity_before,quantity_after,note,created_by,goods_receipt_id)
				VALUES ($1,$2,$3,$4,'purchase',$5,$6,$7,'Purchase order receipt',$8,$9)`, id.New(id.StockMovement), tenantID, storeID, line.variantID, line.quantity, line.before, line.before+line.quantity, userID, receiptID)
			if err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `UPDATE purchase_orders SET status=CASE WHEN EXISTS(
			SELECT 1 FROM purchase_order_items WHERE tenant_id=$1 AND purchase_order_id=$2 AND received_quantity<quantity)
			THEN 'partially_received' ELSE 'received' END WHERE tenant_id=$1 AND id=$2`, tenantID, orderID)
		if err != nil {
			return err
		}
		out, err = loadOrder(ctx, tx, tenantID, orderID)
		if err != nil {
			return err
		}
		for i := range out.Receipts {
			if out.Receipts[i].ID == receiptID {
				receipt = &out.Receipts[i]
				created = true
				return nil
			}
		}
		return ErrNotFound
	})
	return out, receipt, created, err
}

func loadOrder(ctx context.Context, tx pgx.Tx, tenantID, orderID string) (*Order, error) {
	o := &Order{Items: []Item{}, Receipts: []Receipt{}}
	err := tx.QueryRow(ctx, `SELECT id,number,store_id,store_name_snapshot,supplier_id,supplier_name_snapshot,status,currency_code,total_cost_minor,created_at,approved_at
		FROM purchase_orders WHERE tenant_id=$1 AND id=$2`, tenantID, orderID).
		Scan(&o.ID, &o.Number, &o.StoreID, &o.StoreName, &o.SupplierID, &o.SupplierName, &o.Status, &o.CurrencyCode, &o.TotalCostMinor, &o.CreatedAt, &o.ApprovedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT variant_id,product_name_snapshot,variant_name_snapshot,sku_snapshot,quantity,received_quantity,unit_cost_minor,line_cost_minor
		FROM purchase_order_items WHERE tenant_id=$1 AND purchase_order_id=$2 ORDER BY variant_id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.VariantID, &item.ProductName, &item.VariantName, &item.SKU, &item.Quantity, &item.ReceivedQuantity, &item.UnitCostMinor, &item.LineCostMinor); err != nil {
			rows.Close()
			return nil, err
		}
		o.Items = append(o.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.Query(ctx, `SELECT id,total_cost_minor,created_at FROM goods_receipts WHERE tenant_id=$1 AND purchase_order_id=$2 ORDER BY created_at,id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var receipt Receipt
		receipt.PurchaseOrderID = orderID
		receipt.Items = []ReceiptItem{}
		if err := rows.Scan(&receipt.ID, &receipt.TotalCostMinor, &receipt.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		o.Receipts = append(o.Receipts, receipt)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range o.Receipts {
		rows, err = tx.Query(ctx, `SELECT variant_id,quantity,unit_cost_minor,line_cost_minor FROM goods_receipt_items
			WHERE tenant_id=$1 AND goods_receipt_id=$2 ORDER BY variant_id`, tenantID, o.Receipts[i].ID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var line ReceiptItem
			if err := rows.Scan(&line.VariantID, &line.Quantity, &line.UnitCostMinor, &line.LineCostMinor); err != nil {
				rows.Close()
				return nil, err
			}
			o.Receipts[i].Items = append(o.Receipts[i].Items, line)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return o, nil
}
