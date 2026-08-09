-- +goose Up
-- Candella staff who can operate the platform console.
--
-- A separate table rather than a `users.is_super_admin` boolean: a flag on a hot
-- table is one bad UPDATE away from catastrophe, leaves no room for role
-- separation, and carries no grant history. Linking to users instead of building
-- a second auth stack means staff inherit Argon2id, TOTP 2FA and sessions —
-- and 2FA is enforced for this table at the middleware layer.
CREATE TABLE platform_admins (
  id             TEXT PRIMARY KEY CHECK (id ~ '^padm_[0-9A-Za-z]{12}$'),
  user_id        TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  -- Least privilege: no single "god" role.
  --   owner       — everything, including granting and revoking admins
  --   support     — read tenants and users, read-only impersonation
  --   billing     — subscriptions, invoices, trial extensions
  --   engineering — system health and feature flags, no PII reveal by default
  role           TEXT NOT NULL CHECK (role IN ('owner', 'support', 'billing', 'engineering')),
  status         TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
  -- NULL for the first admin, which is bootstrapped from the CLI.
  created_by     TEXT REFERENCES platform_admins(id) ON DELETE SET NULL,
  last_access_at TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX platform_admins_status_idx ON platform_admins (status);

CREATE TRIGGER platform_admins_set_updated_at
BEFORE UPDATE ON platform_admins
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE platform_admins;
