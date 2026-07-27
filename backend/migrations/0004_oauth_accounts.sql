-- +goose Up
CREATE TABLE oauth_accounts (
  id                  TEXT PRIMARY KEY CHECK (id ~ '^oauth_[0-9A-Za-z]{12}$'),
  user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider            TEXT NOT NULL CHECK (provider IN ('google')),
  provider_account_id TEXT NOT NULL,
  email               CITEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- A given external account maps to exactly one Candella user.
  UNIQUE (provider, provider_account_id)
);

CREATE INDEX oauth_accounts_user_id_idx ON oauth_accounts (user_id);

-- +goose Down
DROP TABLE oauth_accounts;
