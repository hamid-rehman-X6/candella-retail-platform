-- +goose Up
CREATE TABLE reorder_settings (
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  store_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  threshold BIGINT NOT NULL CHECK (threshold BETWEEN 0 AND 1000000000),
  preferred_supplier_id TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, store_id, variant_id),
  FOREIGN KEY (tenant_id, store_id) REFERENCES stores(tenant_id, id),
  FOREIGN KEY (tenant_id, variant_id) REFERENCES product_variants(tenant_id, id),
  FOREIGN KEY (tenant_id, preferred_supplier_id) REFERENCES suppliers(tenant_id, id)
);
CREATE INDEX reorder_settings_store_idx ON reorder_settings (tenant_id, store_id, threshold);
GRANT SELECT, INSERT, UPDATE, DELETE ON reorder_settings TO candella_catalog;
GRANT SELECT (tenant_id, store_id, variant_id, threshold) ON reorder_settings TO candella_platform;
ALTER TABLE reorder_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE reorder_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY reorder_settings_tenant ON reorder_settings
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY reorder_settings_platform_read ON reorder_settings FOR SELECT TO candella_platform USING (true);

-- +goose Down
DROP TABLE reorder_settings;
