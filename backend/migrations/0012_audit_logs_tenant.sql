-- +goose Up
-- Completes audit_logs now that tenants exist (0009 deferred this deliberately).
--
-- One table records every actor: tenant users, Candella staff acting from the
-- platform console, and the system itself. actor_type is what tells them apart,
-- while user_id stays the FK for all human actors — platform admins are users too.
ALTER TABLE audit_logs
  ADD COLUMN tenant_id  TEXT REFERENCES tenants(id) ON DELETE SET NULL,
  ADD COLUMN actor_type TEXT NOT NULL DEFAULT 'user'
               CHECK (actor_type IN ('user', 'platform_admin', 'system')),
  ADD COLUMN user_agent TEXT;

-- Drives the per-tenant activity timeline (newest first).
CREATE INDEX audit_logs_tenant_id_idx ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX audit_logs_action_idx ON audit_logs (action);
-- Platform-plane actions: the "what has staff been doing" stream.
CREATE INDEX audit_logs_platform_idx ON audit_logs (created_at DESC)
  WHERE actor_type = 'platform_admin';

-- +goose Down
DROP INDEX IF EXISTS audit_logs_platform_idx;
DROP INDEX IF EXISTS audit_logs_action_idx;
DROP INDEX IF EXISTS audit_logs_tenant_id_idx;
ALTER TABLE audit_logs
  DROP COLUMN user_agent,
  DROP COLUMN actor_type,
  DROP COLUMN tenant_id;
