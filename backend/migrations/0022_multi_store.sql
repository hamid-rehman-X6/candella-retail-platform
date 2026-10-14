-- +goose Up
CREATE UNIQUE INDEX stores_tenant_name_key ON stores (tenant_id, lower(name));

CREATE TABLE store_access (
  tenant_id TEXT NOT NULL,
  store_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, store_id, user_id),
  FOREIGN KEY (tenant_id, store_id) REFERENCES stores(tenant_id, id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, user_id) REFERENCES tenant_memberships(tenant_id, user_id) ON DELETE CASCADE
);
CREATE INDEX store_access_user_idx ON store_access (tenant_id, user_id);
GRANT SELECT, INSERT, DELETE ON store_access TO candella_catalog;
ALTER TABLE store_access ENABLE ROW LEVEL SECURITY;
ALTER TABLE store_access FORCE ROW LEVEL SECURITY;
CREATE POLICY store_access_tenant ON store_access
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

ALTER TABLE orders ADD COLUMN store_name_snapshot TEXT;
UPDATE orders o SET store_name_snapshot=s.name FROM stores s
  WHERE s.tenant_id=o.tenant_id AND s.id=o.store_id;
ALTER TABLE orders ALTER COLUMN store_name_snapshot SET NOT NULL;

-- +goose Down
ALTER TABLE orders DROP COLUMN store_name_snapshot;
DROP TABLE store_access;
DROP INDEX stores_tenant_name_key;
