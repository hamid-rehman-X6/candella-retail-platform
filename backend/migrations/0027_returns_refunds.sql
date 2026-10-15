-- +goose Up
-- A return takes goods back from a completed sale and pays cash out of the
-- drawer of the counter it is processed on. The original order, its payment
-- and the stock ledger are never edited: the return is a new document with its
-- own lines, a positive stock movement per line, and a cash refund that the
-- shift accounts for at closing.

-- Return numbers (RET-000001) run separately from sale numbers.
ALTER TABLE document_sequences
  ADD COLUMN current_return_number BIGINT NOT NULL DEFAULT 0 CHECK (current_return_number >= 0);

-- Lets a return insist it is processed in the store that made the sale.
ALTER TABLE orders ADD CONSTRAINT orders_tenant_store_id_key UNIQUE (tenant_id, store_id, id);

CREATE TABLE returns (
  id TEXT PRIMARY KEY CHECK (id ~ '^ret_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  store_id TEXT NOT NULL,
  order_id TEXT NOT NULL,
  register_session_id TEXT NOT NULL,
  return_number TEXT NOT NULL,
  idempotency_key TEXT NOT NULL CHECK (length(idempotency_key) BETWEEN 8 AND 100),
  request_hash TEXT NOT NULL CHECK (length(request_hash) = 64),
  reason TEXT NOT NULL CHECK (length(btrim(reason)) BETWEEN 3 AND 500),
  refund_method TEXT NOT NULL DEFAULT 'cash' CHECK (refund_method = 'cash'),
  refund_minor BIGINT NOT NULL CHECK (refund_minor > 0),
  currency_code CHAR(3) NOT NULL,
  created_by TEXT NOT NULL REFERENCES users(id),
  processor_name_snapshot TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, id, order_id),
  UNIQUE (tenant_id, idempotency_key),
  UNIQUE (tenant_id, return_number),
  FOREIGN KEY (tenant_id, store_id, order_id) REFERENCES orders(tenant_id, store_id, id),
  FOREIGN KEY (tenant_id, store_id, register_session_id) REFERENCES register_sessions(tenant_id, store_id, id)
);
CREATE INDEX returns_order_idx ON returns (tenant_id, order_id);
CREATE INDEX returns_shift_idx ON returns (tenant_id, register_session_id);

CREATE TABLE return_items (
  id TEXT PRIMARY KEY CHECK (id ~ '^ritem_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  return_id TEXT NOT NULL,
  order_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  quantity BIGINT NOT NULL CHECK (quantity > 0),
  unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor >= 0),
  line_refund_minor BIGINT NOT NULL CHECK (line_refund_minor = unit_price_minor * quantity),
  UNIQUE (tenant_id, return_id, variant_id),
  -- The line must be from this return's own sale, and that sale must have sold it.
  FOREIGN KEY (tenant_id, return_id, order_id) REFERENCES returns(tenant_id, id, order_id),
  FOREIGN KEY (tenant_id, order_id, variant_id) REFERENCES order_items(tenant_id, order_id, variant_id)
);
CREATE INDEX return_items_order_idx ON return_items (tenant_id, order_id, variant_id);

-- Restocking is a stock movement of its own kind, tied to its return.
ALTER TABLE stock_movements ADD COLUMN return_id TEXT;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_kind_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_kind_check
  CHECK (kind IN ('receive', 'count', 'sale', 'return'));
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_return_check
  CHECK ((kind = 'return') = (return_id IS NOT NULL));
ALTER TABLE stock_movements ADD FOREIGN KEY (tenant_id, return_id) REFERENCES returns(tenant_id, id);

-- Shifts account for refunds: expected cash = opening + cash sales - cash refunds.
-- Like the sales figures, refunds are recorded on the row at closing; while a
-- shift is open they are computed live and these columns stay 0.
ALTER TABLE register_sessions
  ADD COLUMN refunds_count BIGINT NOT NULL DEFAULT 0 CHECK (refunds_count >= 0),
  ADD COLUMN cash_refunds_minor BIGINT NOT NULL DEFAULT 0 CHECK (cash_refunds_minor >= 0);
ALTER TABLE register_sessions DROP CONSTRAINT register_sessions_check;
ALTER TABLE register_sessions ADD CONSTRAINT register_sessions_closing_check CHECK (
  (status = 'open' AND closed_by IS NULL AND closer_name_snapshot IS NULL AND closed_at IS NULL
    AND sales_count IS NULL AND cash_sales_minor IS NULL AND expected_cash_minor IS NULL
    AND counted_cash_minor IS NULL AND variance_minor IS NULL AND variance_reason IS NULL)
  OR
  (status = 'closed' AND closed_by IS NOT NULL AND closer_name_snapshot IS NOT NULL AND closed_at IS NOT NULL
    AND sales_count >= 0 AND cash_sales_minor >= 0
    AND expected_cash_minor = opening_cash_minor + cash_sales_minor - cash_refunds_minor
    AND counted_cash_minor IS NOT NULL
    AND variance_minor = counted_cash_minor - expected_cash_minor
    AND (variance_minor = 0 OR length(btrim(variance_reason)) >= 3))
);
GRANT UPDATE (refunds_count, cash_refunds_minor) ON register_sessions TO candella_catalog;

-- Returns are append-only for the API, like sales.
GRANT SELECT, INSERT ON returns, return_items TO candella_catalog;

ALTER TABLE returns ENABLE ROW LEVEL SECURITY;
ALTER TABLE returns FORCE ROW LEVEL SECURITY;
CREATE POLICY returns_tenant ON returns
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
ALTER TABLE return_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE return_items FORCE ROW LEVEL SECURITY;
CREATE POLICY return_items_tenant ON return_items
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

GRANT SELECT ON returns, return_items TO candella_platform;
CREATE POLICY returns_platform_read ON returns FOR SELECT TO candella_platform USING (true);
CREATE POLICY return_items_platform_read ON return_items FOR SELECT TO candella_platform USING (true);

-- +goose Down
ALTER TABLE register_sessions DROP CONSTRAINT register_sessions_closing_check;
ALTER TABLE register_sessions ADD CONSTRAINT register_sessions_check CHECK (
  (status = 'open' AND closed_by IS NULL AND closer_name_snapshot IS NULL AND closed_at IS NULL
    AND sales_count IS NULL AND cash_sales_minor IS NULL AND expected_cash_minor IS NULL
    AND counted_cash_minor IS NULL AND variance_minor IS NULL AND variance_reason IS NULL)
  OR
  (status = 'closed' AND closed_by IS NOT NULL AND closer_name_snapshot IS NOT NULL AND closed_at IS NOT NULL
    AND sales_count >= 0 AND cash_sales_minor >= 0
    AND expected_cash_minor = opening_cash_minor + cash_sales_minor
    AND counted_cash_minor IS NOT NULL
    AND variance_minor = counted_cash_minor - expected_cash_minor
    AND (variance_minor = 0 OR length(btrim(variance_reason)) >= 3))
) NOT VALID;
ALTER TABLE register_sessions DROP COLUMN cash_refunds_minor, DROP COLUMN refunds_count;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_return_check;
ALTER TABLE stock_movements DROP COLUMN return_id;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_kind_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_kind_check
  CHECK (kind IN ('receive', 'count', 'sale')) NOT VALID;
DROP TABLE return_items;
DROP TABLE returns;
ALTER TABLE orders DROP CONSTRAINT orders_tenant_store_id_key;
ALTER TABLE document_sequences DROP COLUMN current_return_number;
