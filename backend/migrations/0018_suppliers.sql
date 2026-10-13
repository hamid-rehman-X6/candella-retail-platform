-- +goose Up
CREATE TABLE suppliers (
  id TEXT PRIMARY KEY CHECK (id ~ '^sup_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 160),
  phone TEXT NOT NULL DEFAULT '' CHECK (length(phone) <= 40),
  email TEXT NOT NULL DEFAULT '' CHECK (length(email) <= 254),
  address TEXT NOT NULL DEFAULT '' CHECK (length(address) <= 500),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id)
);
CREATE UNIQUE INDEX suppliers_active_name_idx ON suppliers (tenant_id, lower(name)) WHERE status = 'active';
CREATE INDEX suppliers_tenant_name_idx ON suppliers (tenant_id, lower(name));
CREATE TRIGGER suppliers_set_updated_at BEFORE UPDATE ON suppliers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE supplier_variants (
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  supplier_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  supplier_sku TEXT NOT NULL DEFAULT '' CHECK (length(supplier_sku) <= 80),
  cost_minor BIGINT CHECK (cost_minor BETWEEN 0 AND 1000000000000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, supplier_id, variant_id),
  FOREIGN KEY (tenant_id, supplier_id) REFERENCES suppliers(tenant_id, id),
  FOREIGN KEY (tenant_id, variant_id) REFERENCES product_variants(tenant_id, id)
);
CREATE UNIQUE INDEX supplier_variants_sku_idx ON supplier_variants
  (tenant_id, supplier_id, lower(supplier_sku)) WHERE supplier_sku <> '';
CREATE INDEX supplier_variants_variant_idx ON supplier_variants (tenant_id, variant_id);
CREATE TRIGGER supplier_variants_set_updated_at BEFORE UPDATE ON supplier_variants
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

GRANT SELECT, INSERT, UPDATE ON suppliers, supplier_variants TO candella_catalog;
ALTER TABLE suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE suppliers FORCE ROW LEVEL SECURITY;
CREATE POLICY suppliers_tenant ON suppliers
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
ALTER TABLE supplier_variants ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_variants FORCE ROW LEVEL SECURITY;
CREATE POLICY supplier_variants_tenant ON supplier_variants
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

-- +goose Down
DROP TABLE supplier_variants;
DROP TABLE suppliers;
