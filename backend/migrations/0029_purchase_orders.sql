-- +goose Up
ALTER TABLE document_sequences ADD COLUMN current_purchase_number BIGINT NOT NULL DEFAULT 0 CHECK (current_purchase_number >= 0);

CREATE TABLE purchase_orders (
  id TEXT PRIMARY KEY CHECK (id ~ '^po_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  store_id TEXT NOT NULL,
  supplier_id TEXT NOT NULL,
  number TEXT NOT NULL,
  idempotency_key TEXT NOT NULL CHECK (length(idempotency_key) BETWEEN 8 AND 100),
  request_hash TEXT NOT NULL CHECK (length(request_hash) = 64),
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','approved','partially_received','received')),
  store_name_snapshot TEXT NOT NULL,
  supplier_name_snapshot TEXT NOT NULL,
  currency_code CHAR(3) NOT NULL,
  total_cost_minor BIGINT NOT NULL CHECK (total_cost_minor BETWEEN 0 AND 1000000000000),
  created_by TEXT NOT NULL REFERENCES users(id),
  approved_by TEXT REFERENCES users(id),
  approved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, id, store_id),
  UNIQUE (tenant_id, number),
  UNIQUE (tenant_id, idempotency_key),
  FOREIGN KEY (tenant_id, store_id) REFERENCES stores(tenant_id, id),
  FOREIGN KEY (tenant_id, supplier_id) REFERENCES suppliers(tenant_id, id),
  CHECK ((status='draft' AND approved_by IS NULL AND approved_at IS NULL) OR
         (status<>'draft' AND approved_by IS NOT NULL AND approved_at IS NOT NULL))
);
CREATE INDEX purchase_orders_tenant_store_created_idx ON purchase_orders (tenant_id,store_id,created_at DESC,id DESC);

CREATE TABLE purchase_order_items (
  id TEXT PRIMARY KEY CHECK (id ~ '^poitem_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  purchase_order_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  product_name_snapshot TEXT NOT NULL,
  variant_name_snapshot TEXT NOT NULL,
  sku_snapshot TEXT NOT NULL,
  quantity BIGINT NOT NULL CHECK (quantity BETWEEN 1 AND 1000000),
  received_quantity BIGINT NOT NULL DEFAULT 0 CHECK (received_quantity >= 0 AND received_quantity <= quantity),
  unit_cost_minor BIGINT NOT NULL CHECK (unit_cost_minor BETWEEN 0 AND 1000000000000),
  line_cost_minor BIGINT NOT NULL CHECK (line_cost_minor = quantity * unit_cost_minor AND line_cost_minor <= 1000000000000),
  UNIQUE (tenant_id,purchase_order_id,variant_id),
  FOREIGN KEY (tenant_id,purchase_order_id) REFERENCES purchase_orders(tenant_id,id),
  FOREIGN KEY (tenant_id,variant_id) REFERENCES product_variants(tenant_id,id)
);

CREATE TABLE goods_receipts (
  id TEXT PRIMARY KEY CHECK (id ~ '^gr_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  purchase_order_id TEXT NOT NULL,
  store_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL CHECK (length(idempotency_key) BETWEEN 8 AND 100),
  request_hash TEXT NOT NULL CHECK (length(request_hash) = 64),
  total_cost_minor BIGINT NOT NULL CHECK (total_cost_minor BETWEEN 0 AND 1000000000000),
  created_by TEXT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id,id),
  UNIQUE (tenant_id,id,store_id),
  UNIQUE (tenant_id,id,purchase_order_id),
  UNIQUE (tenant_id,idempotency_key),
  FOREIGN KEY (tenant_id,purchase_order_id,store_id) REFERENCES purchase_orders(tenant_id,id,store_id),
  FOREIGN KEY (tenant_id,store_id) REFERENCES stores(tenant_id,id)
);
CREATE INDEX goods_receipts_order_idx ON goods_receipts (tenant_id,purchase_order_id,created_at,id);

CREATE TABLE goods_receipt_items (
  id TEXT PRIMARY KEY CHECK (id ~ '^gritem_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  goods_receipt_id TEXT NOT NULL,
  purchase_order_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  quantity BIGINT NOT NULL CHECK (quantity BETWEEN 1 AND 1000000),
  unit_cost_minor BIGINT NOT NULL CHECK (unit_cost_minor BETWEEN 0 AND 1000000000000),
  line_cost_minor BIGINT NOT NULL CHECK (line_cost_minor = quantity * unit_cost_minor),
  UNIQUE (tenant_id,goods_receipt_id,variant_id),
  FOREIGN KEY (tenant_id,goods_receipt_id,purchase_order_id) REFERENCES goods_receipts(tenant_id,id,purchase_order_id),
  FOREIGN KEY (tenant_id,purchase_order_id,variant_id) REFERENCES purchase_order_items(tenant_id,purchase_order_id,variant_id)
);

ALTER TABLE stock_movements ADD COLUMN goods_receipt_id TEXT;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_goods_receipt_fk
  FOREIGN KEY (tenant_id,goods_receipt_id,store_id) REFERENCES goods_receipts(tenant_id,id,store_id);
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_kind_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_kind_check
  CHECK (kind IN ('receive','count','sale','return','purchase'));
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_purchase_check
  CHECK ((kind='purchase') = (goods_receipt_id IS NOT NULL));
CREATE UNIQUE INDEX stock_movements_receipt_variant_idx ON stock_movements (tenant_id,goods_receipt_id,variant_id)
  WHERE goods_receipt_id IS NOT NULL;

GRANT SELECT,INSERT ON purchase_orders,purchase_order_items TO candella_catalog;
GRANT UPDATE (status,approved_by,approved_at) ON purchase_orders TO candella_catalog;
GRANT UPDATE (received_quantity) ON purchase_order_items TO candella_catalog;
GRANT SELECT,INSERT ON goods_receipts,goods_receipt_items TO candella_catalog;
GRANT SELECT (id,tenant_id,store_id,supplier_id,status,created_at) ON purchase_orders TO candella_platform;
GRANT SELECT (id,tenant_id,purchase_order_id,variant_id) ON purchase_order_items TO candella_platform;
GRANT SELECT (id,tenant_id,purchase_order_id,store_id,created_at) ON goods_receipts TO candella_platform;
GRANT SELECT (id,tenant_id,goods_receipt_id,purchase_order_id,variant_id) ON goods_receipt_items TO candella_platform;

ALTER TABLE purchase_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_orders FORCE ROW LEVEL SECURITY;
CREATE POLICY purchase_orders_tenant ON purchase_orders USING (tenant_id=current_setting('app.current_tenant_id',true)) WITH CHECK (tenant_id=current_setting('app.current_tenant_id',true));
CREATE POLICY purchase_orders_platform_read ON purchase_orders FOR SELECT TO candella_platform USING (true);
ALTER TABLE purchase_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_items FORCE ROW LEVEL SECURITY;
CREATE POLICY purchase_order_items_tenant ON purchase_order_items USING (tenant_id=current_setting('app.current_tenant_id',true)) WITH CHECK (tenant_id=current_setting('app.current_tenant_id',true));
CREATE POLICY purchase_order_items_platform_read ON purchase_order_items FOR SELECT TO candella_platform USING (true);
ALTER TABLE goods_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE goods_receipts FORCE ROW LEVEL SECURITY;
CREATE POLICY goods_receipts_tenant ON goods_receipts USING (tenant_id=current_setting('app.current_tenant_id',true)) WITH CHECK (tenant_id=current_setting('app.current_tenant_id',true));
CREATE POLICY goods_receipts_platform_read ON goods_receipts FOR SELECT TO candella_platform USING (true);
ALTER TABLE goods_receipt_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE goods_receipt_items FORCE ROW LEVEL SECURITY;
CREATE POLICY goods_receipt_items_tenant ON goods_receipt_items USING (tenant_id=current_setting('app.current_tenant_id',true)) WITH CHECK (tenant_id=current_setting('app.current_tenant_id',true));
CREATE POLICY goods_receipt_items_platform_read ON goods_receipt_items FOR SELECT TO candella_platform USING (true);

-- +goose Down
DROP INDEX stock_movements_receipt_variant_idx;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_purchase_check;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_goods_receipt_fk;
ALTER TABLE stock_movements DROP COLUMN goods_receipt_id;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_kind_check;
-- The ledger is immutable. Refuse rollback if purchase movements already exist.
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_kind_check CHECK (kind IN ('receive','count','sale','return'));
DROP TABLE goods_receipt_items;
DROP TABLE goods_receipts;
DROP TABLE purchase_order_items;
DROP TABLE purchase_orders;
ALTER TABLE document_sequences DROP COLUMN current_purchase_number;
