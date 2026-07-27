-- +goose Up
CREATE TABLE mfa_backup_codes (
  id         TEXT PRIMARY KEY CHECK (id ~ '^bkp_[0-9A-Za-z]{12}$'),
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- SHA-256 of a single-use backup code.
  code_hash  TEXT NOT NULL,
  used_at    TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX mfa_backup_codes_user_id_idx ON mfa_backup_codes (user_id);

-- +goose Down
DROP TABLE mfa_backup_codes;
