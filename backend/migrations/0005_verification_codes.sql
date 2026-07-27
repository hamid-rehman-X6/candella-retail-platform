-- +goose Up
CREATE TABLE verification_codes (
  id          TEXT PRIMARY KEY CHECK (id ~ '^vcode_[0-9A-Za-z]{12}$'),
  user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose     TEXT NOT NULL CHECK (purpose IN ('email_verify')),
  -- SHA-256 of the 6-digit code; the code itself is only ever emailed.
  code_hash   TEXT NOT NULL,
  attempts    INT NOT NULL DEFAULT 0,
  expires_at  TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX verification_codes_user_purpose_idx ON verification_codes (user_id, purpose);

-- +goose Down
DROP TABLE verification_codes;
