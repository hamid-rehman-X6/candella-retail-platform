-- +goose Up
CREATE TABLE sessions (
  id         TEXT PRIMARY KEY CHECK (id ~ '^sess_[0-9A-Za-z]{12}$'),
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- SHA-256 of the opaque session token; the raw token lives only in the cookie.
  token_hash TEXT NOT NULL UNIQUE,
  ip_address INET,
  user_agent TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- +goose Down
DROP TABLE sessions;
