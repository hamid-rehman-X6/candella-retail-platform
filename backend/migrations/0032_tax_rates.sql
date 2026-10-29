-- +goose Up
-- Tax rates a workspace charges, and which products carry them. Checkout
-- copies the rate onto each order line (name, basis points, inclusive or not,
-- and the tax amount), so editing or archiving a rate never changes a past
-- sale, its receipt, a later refund, or a report.
--
-- Rates are in basis points: 1800 = 18%. "Inclusive" means the shelf price
-- already contains the tax (the usual retail price in Pakistan); "exclusive"
-- means the tax is added at checkout.
CREATE TABLE tax_rates (
  id TEXT PRIMARY KEY CHECK (id ~ '^tax_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  rate_bps INTEGER NOT NULL CHECK (rate_bps BETWEEN 0 AND 10000),
  inclusive BOOLEAN NOT NULL,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id)
);
CREATE UNIQUE INDEX tax_rates_active_name_key ON tax_rates (tenant_id, lower(name)) WHERE status = 'active';
CREATE TRIGGER tax_rates_set_updated_at BEFORE UPDATE ON tax_rates
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- No rate means the product is not taxed.
ALTER TABLE products ADD COLUMN tax_rate_id TEXT;
ALTER TABLE products ADD CONSTRAINT products_tax_rate_fkey
  FOREIGN KEY (tenant_id, tax_rate_id) REFERENCES tax_rates(tenant_id, id);
CREATE INDEX products_tax_rate_idx ON products (tenant_id, tax_rate_id) WHERE tax_rate_id IS NOT NULL;

-- Order lines keep the rate as it was when sold.
ALTER TABLE order_items
  ADD COLUMN tax_rate_id TEXT,
  ADD COLUMN tax_name_snapshot TEXT,
  ADD COLUMN tax_rate_bps INTEGER CHECK (tax_rate_bps BETWEEN 0 AND 10000),
  ADD COLUMN tax_inclusive BOOLEAN,
  ADD COLUMN tax_minor BIGINT NOT NULL DEFAULT 0 CHECK (tax_minor >= 0);
ALTER TABLE order_items ADD CONSTRAINT order_items_tax_snapshot_check CHECK (
  (tax_rate_bps IS NULL) = (tax_name_snapshot IS NULL)
  AND (tax_rate_bps IS NULL) = (tax_inclusive IS NULL)
  AND (tax_rate_bps IS NOT NULL OR tax_minor = 0)
  AND (tax_inclusive IS NOT TRUE OR tax_minor <= line_total_minor)
);
ALTER TABLE order_items ADD CONSTRAINT order_items_tax_rate_fkey
  FOREIGN KEY (tenant_id, tax_rate_id) REFERENCES tax_rates(tenant_id, id);

-- An order's tax is the sum of its lines; exclusive tax is what lifts the
-- grand total above the subtotal.
ALTER TABLE orders ADD COLUMN tax_minor BIGINT NOT NULL DEFAULT 0 CHECK (tax_minor >= 0);
ALTER TABLE orders ADD CONSTRAINT orders_grand_total_covers_subtotal CHECK (grand_total_minor >= subtotal_minor);

-- A refund now includes the returned share of the line's tax, so it is no
-- longer simply price x quantity.
ALTER TABLE return_items DROP CONSTRAINT return_items_check;
ALTER TABLE return_items ADD CONSTRAINT return_items_line_refund_minor_check CHECK (line_refund_minor >= 0);
ALTER TABLE return_items ADD COLUMN tax_minor BIGINT NOT NULL DEFAULT 0 CHECK (tax_minor >= 0);
ALTER TABLE returns ADD COLUMN tax_minor BIGINT NOT NULL DEFAULT 0 CHECK (tax_minor >= 0);

-- Tenant access: rates are archived, never deleted, so old lines keep their FK.
GRANT SELECT, INSERT, UPDATE ON tax_rates TO candella_catalog;
ALTER TABLE tax_rates ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_rates FORCE ROW LEVEL SECURITY;
CREATE POLICY tax_rates_tenant ON tax_rates
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

GRANT SELECT ON tax_rates TO candella_platform;
CREATE POLICY tax_rates_platform_read ON tax_rates FOR SELECT TO candella_platform USING (true);

-- +goose Down
ALTER TABLE returns DROP COLUMN tax_minor;
ALTER TABLE return_items DROP COLUMN tax_minor;
ALTER TABLE return_items DROP CONSTRAINT return_items_line_refund_minor_check;
ALTER TABLE return_items ADD CONSTRAINT return_items_check
  CHECK (line_refund_minor = unit_price_minor * quantity) NOT VALID;
ALTER TABLE orders DROP CONSTRAINT orders_grand_total_covers_subtotal;
ALTER TABLE orders DROP COLUMN tax_minor;
ALTER TABLE order_items DROP CONSTRAINT order_items_tax_rate_fkey;
ALTER TABLE order_items DROP CONSTRAINT order_items_tax_snapshot_check;
ALTER TABLE order_items DROP COLUMN tax_minor, DROP COLUMN tax_inclusive, DROP COLUMN tax_rate_bps,
  DROP COLUMN tax_name_snapshot, DROP COLUMN tax_rate_id;
DROP INDEX products_tax_rate_idx;
ALTER TABLE products DROP CONSTRAINT products_tax_rate_fkey;
ALTER TABLE products DROP COLUMN tax_rate_id;
DROP TABLE tax_rates;
