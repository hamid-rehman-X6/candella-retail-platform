-- +goose Up
CREATE TABLE staff_invitations (
  id TEXT PRIMARY KEY CHECK (id ~ '^invite_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email CITEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('owner','admin','member')),
  token_hash TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','revoked','expired')),
  invited_by TEXT NOT NULL REFERENCES users(id),
  accepted_by TEXT REFERENCES users(id),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id)
);
CREATE UNIQUE INDEX staff_invitations_one_pending_email
  ON staff_invitations (tenant_id, email) WHERE status='pending';
CREATE INDEX staff_invitations_tenant_created_idx
  ON staff_invitations (tenant_id, created_at DESC);
CREATE TRIGGER staff_invitations_set_updated_at
BEFORE UPDATE ON staff_invitations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE staff_invitations;
