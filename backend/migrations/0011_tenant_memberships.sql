-- +goose Up
-- Links a global user to a tenant. The `role` column is a lightweight RBAC for now
-- (owner/admin/member); a full roles/permissions system can replace it later.
CREATE TABLE tenant_memberships (
  id         TEXT PRIMARY KEY CHECK (id ~ '^tmem_[0-9A-Za-z]{12}$'),
  tenant_id  TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role       TEXT NOT NULL DEFAULT 'owner' CHECK (role IN ('owner','admin','member')),
  status     TEXT NOT NULL DEFAULT 'active'
               CHECK (status IN ('active','invited','suspended','removed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, user_id)
);

CREATE INDEX tenant_memberships_user_id_idx ON tenant_memberships (user_id);

CREATE TRIGGER tenant_memberships_set_updated_at
BEFORE UPDATE ON tenant_memberships
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE tenant_memberships;
