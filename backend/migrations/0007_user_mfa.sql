-- +goose Up
CREATE TABLE user_mfa (
  user_id               TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  -- AES-256-GCM ciphertext of the TOTP shared secret (nonce||ciphertext).
  totp_secret_encrypted BYTEA NOT NULL,
  -- NULL until the user confirms a code and 2FA is actually turned on.
  enabled_at            TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER user_mfa_set_updated_at
BEFORE UPDATE ON user_mfa
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE user_mfa;
