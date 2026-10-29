-- +goose Up
-- Discounts lower what a line is sold for without touching the product's
-- price. They come from promotions an owner or admin approved in advance, or
-- from a manual percentage within a limit the owner sets for cashiers. Tax is
-- worked out on the discounted amount, and every sale keeps a snapshot of the
-- discount it got, so editing or ending a promotion never changes a receipt.
CREATE TABLE promotions (
  id TEXT PRIMARY KEY CHECK (id ~ '^promo_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 80),
  -- percent: percent_bps off (1000 = 10%). amount: amount_minor off the
  -- order, or off each unit for an item promotion.
  kind TEXT NOT NULL CHECK (kind IN ('percent', 'amount')),
  percent_bps INTEGER CHECK (percent_bps BETWEEN 1 AND 10000),
  amount_minor BIGINT CHECK (amount_minor BETWEEN 1 AND 1000000000000),
  applies_to TEXT NOT NULL CHECK (applies_to IN ('order', 'item')),
  min_subtotal_minor BIGINT NOT NULL DEFAULT 0 CHECK (min_subtotal_minor BETWEEN 0 AND 1000000000000),
  -- Inclusive business days in the workspace's timezone; NULL is open-ended.
  starts_on DATE,
  ends_on DATE,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  CHECK ((kind = 'percent' AND percent_bps IS NOT NULL AND amount_minor IS NULL)
      OR (kind = 'amount' AND amount_minor IS NOT NULL AND percent_bps IS NULL)),
  CHECK (starts_on IS NULL OR ends_on IS NULL OR ends_on >= starts_on)
);
CREATE UNIQUE INDEX promotions_active_name_key ON promotions (tenant_id, lower(name)) WHERE status = 'active';
CREATE TRIGGER promotions_set_updated_at BEFORE UPDATE ON promotions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- The largest manual discount a cashier (member) may give on their own.
-- 0 means only owners and admins can give manual discounts.
ALTER TABLE tenants ADD COLUMN cashier_discount_limit_bps INTEGER NOT NULL DEFAULT 0
  CHECK (cashier_discount_limit_bps BETWEEN 0 AND 10000);

-- The order-level discount as given. Line discounts below include each
-- line's share of it, so the lines always add up to the order.
ALTER TABLE orders
  ADD COLUMN discount_minor BIGINT NOT NULL DEFAULT 0 CHECK (discount_minor >= 0),
  ADD COLUMN discount_kind TEXT CHECK (discount_kind IN ('promotion', 'manual')),
  ADD COLUMN discount_name_snapshot TEXT,
  ADD COLUMN discount_percent_bps INTEGER CHECK (discount_percent_bps BETWEEN 1 AND 10000),
  ADD COLUMN discount_reason TEXT CHECK (length(discount_reason) <= 300),
  ADD COLUMN promotion_id TEXT;
ALTER TABLE orders ADD CONSTRAINT orders_promotion_fkey
  FOREIGN KEY (tenant_id, promotion_id) REFERENCES promotions(tenant_id, id);
ALTER TABLE orders ADD CONSTRAINT orders_discount_snapshot_check CHECK (
  (discount_kind IS NULL) = (discount_name_snapshot IS NULL)
  AND (discount_kind = 'promotion') = (promotion_id IS NOT NULL)
  AND (discount_kind IS DISTINCT FROM 'manual' OR length(btrim(discount_reason)) >= 3)
);
ALTER TABLE orders DROP CONSTRAINT orders_grand_total_covers_subtotal;
ALTER TABLE orders ADD CONSTRAINT orders_discount_bounded CHECK (
  discount_minor <= subtotal_minor AND grand_total_minor >= subtotal_minor - discount_minor
);

ALTER TABLE order_items
  ADD COLUMN discount_minor BIGINT NOT NULL DEFAULT 0 CHECK (discount_minor >= 0),
  ADD COLUMN promotion_id TEXT,
  ADD COLUMN promotion_name_snapshot TEXT;
ALTER TABLE order_items ADD CONSTRAINT order_items_promotion_fkey
  FOREIGN KEY (tenant_id, promotion_id) REFERENCES promotions(tenant_id, id);
ALTER TABLE order_items ADD CONSTRAINT order_items_discount_check CHECK (
  discount_minor <= line_total_minor
  AND (promotion_id IS NULL) = (promotion_name_snapshot IS NULL)
  AND (tax_inclusive IS NOT TRUE OR tax_minor <= line_total_minor - discount_minor)
);

GRANT SELECT, INSERT, UPDATE ON promotions TO candella_catalog;
ALTER TABLE promotions ENABLE ROW LEVEL SECURITY;
ALTER TABLE promotions FORCE ROW LEVEL SECURITY;
CREATE POLICY promotions_tenant ON promotions
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

GRANT SELECT ON promotions TO candella_platform;
CREATE POLICY promotions_platform_read ON promotions FOR SELECT TO candella_platform USING (true);

-- +goose Down
ALTER TABLE order_items DROP CONSTRAINT order_items_discount_check;
ALTER TABLE order_items DROP CONSTRAINT order_items_promotion_fkey;
ALTER TABLE order_items DROP COLUMN promotion_name_snapshot, DROP COLUMN promotion_id, DROP COLUMN discount_minor;
ALTER TABLE orders DROP CONSTRAINT orders_discount_bounded;
ALTER TABLE orders ADD CONSTRAINT orders_grand_total_covers_subtotal CHECK (grand_total_minor >= subtotal_minor) NOT VALID;
ALTER TABLE orders DROP CONSTRAINT orders_discount_snapshot_check;
ALTER TABLE orders DROP CONSTRAINT orders_promotion_fkey;
ALTER TABLE orders DROP COLUMN promotion_id, DROP COLUMN discount_reason, DROP COLUMN discount_percent_bps,
  DROP COLUMN discount_name_snapshot, DROP COLUMN discount_kind, DROP COLUMN discount_minor;
ALTER TABLE tenants DROP COLUMN cashier_discount_limit_bps;
DROP TABLE promotions;
