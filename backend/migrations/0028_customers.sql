-- +goose Up
CREATE TABLE customers (
  id TEXT PRIMARY KEY CHECK (id ~ '^cus_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 160),
  phone TEXT NOT NULL DEFAULT '' CHECK (length(phone) <= 40),
  email TEXT NOT NULL DEFAULT '' CHECK (length(email) <= 254),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id)
);
CREATE UNIQUE INDEX customers_active_phone_idx ON customers (tenant_id, phone) WHERE status='active' AND phone<>'';
CREATE UNIQUE INDEX customers_active_email_idx ON customers (tenant_id, lower(email)) WHERE status='active' AND email<>'';
CREATE INDEX customers_tenant_name_idx ON customers (tenant_id, lower(name));
CREATE TRIGGER customers_set_updated_at BEFORE UPDATE ON customers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
GRANT SELECT, INSERT, UPDATE ON customers TO candella_catalog;
ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE customers FORCE ROW LEVEL SECURITY;
CREATE POLICY customers_tenant ON customers
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
-- Platform operators may count customer records without reading contact data.
GRANT SELECT (id, tenant_id, status, created_at, updated_at) ON customers TO candella_platform;
CREATE POLICY customers_platform_read ON customers FOR SELECT TO candella_platform USING (true);

ALTER TABLE orders ADD COLUMN customer_id TEXT;
ALTER TABLE orders ADD COLUMN customer_name_snapshot TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ADD CONSTRAINT orders_customer_tenant_fk
  FOREIGN KEY (tenant_id, customer_id) REFERENCES customers(tenant_id, id);
CREATE INDEX orders_customer_history_idx ON orders (tenant_id, customer_id, created_at DESC, id DESC)
  WHERE customer_id IS NOT NULL;

-- +goose Down
DROP INDEX orders_customer_history_idx;
ALTER TABLE orders DROP CONSTRAINT orders_customer_tenant_fk;
ALTER TABLE orders DROP COLUMN customer_name_snapshot;
ALTER TABLE orders DROP COLUMN customer_id;
DROP TABLE customers;
