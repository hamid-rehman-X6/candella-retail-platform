-- +goose Up
-- Append-only audit trail for security-relevant auth events. (tenant_id from the
-- full schema is added later, once the tenants table exists.)
CREATE TABLE audit_logs (
  id          TEXT PRIMARY KEY CHECK (id ~ '^alog_[0-9A-Za-z]{12}$'),
  user_id     TEXT REFERENCES users(id) ON DELETE SET NULL,
  action      TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id   TEXT NOT NULL,
  metadata    JSONB NOT NULL DEFAULT '{}',
  ip_address  INET,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_user_id_idx ON audit_logs (user_id);
CREATE INDEX audit_logs_created_at_idx ON audit_logs (created_at);

-- +goose Down
DROP TABLE audit_logs;
