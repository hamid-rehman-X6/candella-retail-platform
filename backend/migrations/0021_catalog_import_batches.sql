-- +goose Up
CREATE TABLE catalog_import_batches (
  id TEXT PRIMARY KEY CHECK (id ~ '^cimp_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  idempotency_key TEXT NOT NULL CHECK (length(idempotency_key) BETWEEN 8 AND 100),
  request_hash TEXT NOT NULL,
  mode TEXT NOT NULL CHECK (mode IN ('create','update')),
  created_count INTEGER NOT NULL DEFAULT 0,
  updated_count INTEGER NOT NULL DEFAULT 0,
  result_rows JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_by TEXT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, idempotency_key)
);
GRANT SELECT, INSERT ON catalog_import_batches TO candella_catalog;
ALTER TABLE catalog_import_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_import_batches FORCE ROW LEVEL SECURITY;
CREATE POLICY catalog_import_batches_tenant ON catalog_import_batches
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

-- +goose Down
DROP TABLE catalog_import_batches;
