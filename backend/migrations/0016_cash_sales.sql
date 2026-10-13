-- +goose Up
CREATE TABLE document_sequences (
  tenant_id TEXT NOT NULL PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  current_number BIGINT NOT NULL DEFAULT 0 CHECK (current_number >= 0)
);

CREATE TABLE orders (
  id TEXT PRIMARY KEY CHECK (id ~ '^ord_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  store_id TEXT NOT NULL,
  order_number TEXT NOT NULL,
  idempotency_key TEXT NOT NULL CHECK (length(idempotency_key) BETWEEN 8 AND 100),
  request_hash TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'completed' CHECK (status = 'completed'),
  currency_code CHAR(3) NOT NULL,
  subtotal_minor BIGINT NOT NULL CHECK (subtotal_minor >= 0),
  grand_total_minor BIGINT NOT NULL CHECK (grand_total_minor >= 0),
  created_by TEXT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, order_number),
  UNIQUE (tenant_id, idempotency_key),
  FOREIGN KEY (tenant_id, store_id) REFERENCES stores(tenant_id, id)
);
CREATE INDEX orders_tenant_created_idx ON orders (tenant_id, created_at DESC, id DESC);

CREATE TABLE order_items (
  id TEXT PRIMARY KEY CHECK (id ~ '^oitem_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  order_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  product_name_snapshot TEXT NOT NULL,
  variant_name_snapshot TEXT NOT NULL,
  sku_snapshot TEXT NOT NULL,
  quantity BIGINT NOT NULL CHECK (quantity > 0),
  unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor >= 0),
  line_total_minor BIGINT NOT NULL CHECK (line_total_minor >= 0),
  FOREIGN KEY (tenant_id, order_id) REFERENCES orders(tenant_id, id),
  FOREIGN KEY (tenant_id, variant_id) REFERENCES product_variants(tenant_id, id),
  UNIQUE (tenant_id, order_id, variant_id)
);
CREATE INDEX order_items_tenant_order_idx ON order_items (tenant_id, order_id);

CREATE TABLE payments (
  id TEXT PRIMARY KEY CHECK (id ~ '^pay_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  order_id TEXT NOT NULL,
  method TEXT NOT NULL CHECK (method = 'cash'),
  status TEXT NOT NULL CHECK (status = 'completed'),
  amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
  cash_received_minor BIGINT NOT NULL CHECK (cash_received_minor >= amount_minor),
  change_minor BIGINT NOT NULL CHECK (change_minor = cash_received_minor - amount_minor),
  currency_code CHAR(3) NOT NULL,
  paid_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (tenant_id, order_id) REFERENCES orders(tenant_id, id),
  UNIQUE (tenant_id, order_id)
);

ALTER TABLE stock_movements ADD COLUMN order_id TEXT;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_order_tenant_fk
  FOREIGN KEY (tenant_id, order_id) REFERENCES orders(tenant_id, id);
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_kind_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_kind_check
  CHECK (kind IN ('receive', 'count', 'sale'));
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_sale_order_check
  CHECK ((kind = 'sale' AND order_id IS NOT NULL) OR (kind <> 'sale' AND order_id IS NULL));
CREATE INDEX stock_movements_order_idx ON stock_movements (tenant_id, order_id) WHERE order_id IS NOT NULL;

GRANT SELECT, INSERT, UPDATE ON document_sequences TO candella_catalog;
GRANT SELECT, INSERT ON orders, order_items, payments TO candella_catalog;

ALTER TABLE document_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_sequences FORCE ROW LEVEL SECURITY;
CREATE POLICY document_sequences_tenant ON document_sequences
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders FORCE ROW LEVEL SECURITY;
CREATE POLICY orders_tenant ON orders
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
ALTER TABLE order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_items FORCE ROW LEVEL SECURITY;
CREATE POLICY order_items_tenant ON order_items
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE payments FORCE ROW LEVEL SECURITY;
CREATE POLICY payments_tenant ON payments
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

-- +goose Down
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_sale_order_check;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_kind_check;
-- This check deliberately prevents rollback after sales exist: historical sale movements
-- must never be silently deleted or rewritten.
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_kind_check
  CHECK (kind IN ('receive', 'count'));
DROP INDEX stock_movements_order_idx;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_order_tenant_fk;
ALTER TABLE stock_movements DROP COLUMN order_id;
DROP TABLE payments;
DROP TABLE order_items;
DROP TABLE orders;
DROP TABLE document_sequences;
