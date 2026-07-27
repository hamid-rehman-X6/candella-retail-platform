-- +goose Up
CREATE TABLE users (
  id                TEXT PRIMARY KEY CHECK (id ~ '^usr_[0-9A-Za-z]{12}$'),
  email             CITEXT NOT NULL,
  -- NULL for accounts that only sign in through an OAuth provider.
  password_hash     TEXT,
  full_name         TEXT NOT NULL,
  status            TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  -- Login is gated on this being set (email ownership proven).
  email_verified_at TIMESTAMPTZ,
  mfa_enabled       BOOLEAN NOT NULL DEFAULT false,
  last_login_at     TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at        TIMESTAMPTZ
);

-- Partial unique index so a soft-deleted row never blocks reusing its email.
CREATE UNIQUE INDEX users_email_key ON users (email) WHERE deleted_at IS NULL;

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE users;
