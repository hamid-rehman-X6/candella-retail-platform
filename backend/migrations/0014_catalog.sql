-- +goose Up
-- One default selling/stock location for each workspace. More stores can be added later.
CREATE TABLE stores (
  id TEXT PRIMARY KEY CHECK (id ~ '^str_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  is_default BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id)
);
CREATE UNIQUE INDEX stores_one_default_per_tenant ON stores (tenant_id) WHERE is_default;
INSERT INTO stores (id, tenant_id, name, is_default)
SELECT 'str_' || substr(md5(id), 1, 12), id, 'Main store', true FROM tenants;

CREATE TABLE products (
  id TEXT PRIMARY KEY CHECK (id ~ '^prd_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 160),
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id)
);
CREATE INDEX products_tenant_name_idx ON products (tenant_id, lower(name));
CREATE TRIGGER products_set_updated_at BEFORE UPDATE ON products
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE product_variants (
  id TEXT PRIMARY KEY CHECK (id ~ '^var_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  product_id TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT 'Standard' CHECK (length(btrim(name)) BETWEEN 1 AND 100),
  sku TEXT NOT NULL CHECK (length(btrim(sku)) BETWEEN 1 AND 80),
  barcode TEXT CHECK (barcode IS NULL OR length(btrim(barcode)) BETWEEN 1 AND 80),
  price_minor BIGINT NOT NULL CHECK (price_minor >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, product_id) REFERENCES products(tenant_id, id)
);
CREATE UNIQUE INDEX product_variants_tenant_sku_key ON product_variants (tenant_id, lower(sku));
CREATE UNIQUE INDEX product_variants_tenant_barcode_key ON product_variants (tenant_id, barcode) WHERE barcode IS NOT NULL;
CREATE INDEX product_variants_product_idx ON product_variants (tenant_id, product_id);
CREATE TRIGGER product_variants_set_updated_at BEFORE UPDATE ON product_variants
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- The local Docker DB login is a superuser. A restricted role makes RLS effective
-- for catalog transactions even there. The migration login can SET ROLE into it.
CREATE ROLE candella_catalog NOLOGIN NOBYPASSRLS;
-- +goose StatementBegin
DO $$ BEGIN
  EXECUTE format('GRANT candella_catalog TO %I', current_user);
END $$;
-- +goose StatementEnd
GRANT USAGE ON SCHEMA public TO candella_catalog;
GRANT SELECT, INSERT, UPDATE ON stores, products, product_variants TO candella_catalog;

ALTER TABLE stores ENABLE ROW LEVEL SECURITY;
ALTER TABLE stores FORCE ROW LEVEL SECURITY;
CREATE POLICY stores_tenant ON stores USING (tenant_id = current_setting('app.current_tenant_id', true))
WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
ALTER TABLE products FORCE ROW LEVEL SECURITY;
CREATE POLICY products_tenant ON products USING (tenant_id = current_setting('app.current_tenant_id', true))
WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
ALTER TABLE product_variants ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;
CREATE POLICY product_variants_tenant ON product_variants USING (tenant_id = current_setting('app.current_tenant_id', true))
WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

-- +goose Down
DROP TABLE product_variants;
DROP TABLE products;
DROP TABLE stores;
REVOKE USAGE ON SCHEMA public FROM candella_catalog;
DROP ROLE candella_catalog;
