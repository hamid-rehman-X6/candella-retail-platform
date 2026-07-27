-- +goose Up
CREATE TABLE password_reset_tokens (
  id          TEXT PRIMARY KEY CHECK (id ~ '^prt_[0-9A-Za-z]{12}$'),
  user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- SHA-256 of the opaque URL token; the raw token only travels in the reset link.
  token_hash  TEXT NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);

-- +goose Down
DROP TABLE password_reset_tokens;
