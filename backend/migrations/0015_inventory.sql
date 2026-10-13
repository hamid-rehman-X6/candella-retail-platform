-- +goose Up
CREATE TABLE stock_levels (
  id TEXT PRIMARY KEY CHECK (id ~ '^stk_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  store_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, store_id, variant_id),
  FOREIGN KEY (tenant_id, store_id) REFERENCES stores(tenant_id, id),
  FOREIGN KEY (tenant_id, variant_id) REFERENCES product_variants(tenant_id, id)
);
CREATE TRIGGER stock_levels_set_updated_at BEFORE UPDATE ON stock_levels
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE stock_movements (
  id TEXT PRIMARY KEY CHECK (id ~ '^mov_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  store_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('receive', 'count')),
  delta BIGINT NOT NULL,
  quantity_before BIGINT NOT NULL CHECK (quantity_before >= 0),
  quantity_after BIGINT NOT NULL CHECK (quantity_after >= 0),
  note TEXT NOT NULL DEFAULT '',
  created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (tenant_id, store_id) REFERENCES stores(tenant_id, id),
  FOREIGN KEY (tenant_id, variant_id) REFERENCES product_variants(tenant_id, id)
);
CREATE INDEX stock_movements_tenant_created_idx ON stock_movements (tenant_id, created_at DESC, id DESC);
CREATE INDEX stock_movements_variant_idx ON stock_movements (tenant_id, variant_id, created_at DESC);

-- +goose StatementBegin
CREATE FUNCTION prevent_stock_movement_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'stock movements are append-only';
END $$;
-- +goose StatementEnd
CREATE TRIGGER stock_movements_immutable BEFORE UPDATE OR DELETE ON stock_movements
FOR EACH ROW EXECUTE FUNCTION prevent_stock_movement_change();

GRANT SELECT, INSERT, UPDATE ON stock_levels TO candella_catalog;
GRANT SELECT, INSERT ON stock_movements TO candella_catalog;

ALTER TABLE stock_levels ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_levels FORCE ROW LEVEL SECURITY;
CREATE POLICY stock_levels_tenant ON stock_levels USING (tenant_id = current_setting('app.current_tenant_id', true))
WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
ALTER TABLE stock_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_movements FORCE ROW LEVEL SECURITY;
CREATE POLICY stock_movements_tenant ON stock_movements USING (tenant_id = current_setting('app.current_tenant_id', true))
WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

-- +goose Down
DROP TABLE stock_movements;
DROP TABLE stock_levels;
DROP FUNCTION prevent_stock_movement_change();
