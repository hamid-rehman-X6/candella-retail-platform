-- +goose Up
-- A register is one cash drawer (a counter) in a store. A register session is
-- one shift on it: opened with a float, closed with a counted amount. Every
-- cash sale is taken into the open session of the register it was rung up on.
CREATE TABLE registers (
  id TEXT PRIMARY KEY CHECK (id ~ '^reg_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  store_id TEXT NOT NULL,
  name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, store_id, id),
  FOREIGN KEY (tenant_id, store_id) REFERENCES stores(tenant_id, id)
);
CREATE UNIQUE INDEX registers_store_name_key ON registers (tenant_id, store_id, lower(name));

-- Every existing store gets the counter it has implicitly been selling from.
INSERT INTO registers (id, tenant_id, store_id, name)
SELECT 'reg_' || substr(md5(random()::text || s.id), 1, 12), s.tenant_id, s.id, 'Counter 1'
FROM stores s;

CREATE TABLE register_sessions (
  id TEXT PRIMARY KEY CHECK (id ~ '^shift_[0-9A-Za-z]{12}$'),
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  store_id TEXT NOT NULL,
  register_id TEXT NOT NULL,
  register_name_snapshot TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
  opened_by TEXT NOT NULL REFERENCES users(id),
  opener_name_snapshot TEXT NOT NULL,
  opening_cash_minor BIGINT NOT NULL CHECK (opening_cash_minor BETWEEN 0 AND 1000000000000),
  opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_by TEXT REFERENCES users(id),
  closer_name_snapshot TEXT,
  closed_at TIMESTAMPTZ,
  sales_count BIGINT,
  cash_sales_minor BIGINT,
  expected_cash_minor BIGINT,
  counted_cash_minor BIGINT CHECK (counted_cash_minor BETWEEN 0 AND 1000000000000),
  variance_minor BIGINT,
  variance_reason TEXT CHECK (length(variance_reason) <= 500),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, store_id, id),
  FOREIGN KEY (tenant_id, store_id, register_id) REFERENCES registers(tenant_id, store_id, id),
  -- An open shift carries no closing figures; a closed one carries all of them,
  -- and they must add up.
  CHECK (
    (status = 'open' AND closed_by IS NULL AND closer_name_snapshot IS NULL AND closed_at IS NULL
      AND sales_count IS NULL AND cash_sales_minor IS NULL AND expected_cash_minor IS NULL
      AND counted_cash_minor IS NULL AND variance_minor IS NULL AND variance_reason IS NULL)
    OR
    (status = 'closed' AND closed_by IS NOT NULL AND closer_name_snapshot IS NOT NULL AND closed_at IS NOT NULL
      AND sales_count >= 0 AND cash_sales_minor >= 0
      AND expected_cash_minor = opening_cash_minor + cash_sales_minor
      AND counted_cash_minor IS NOT NULL
      AND variance_minor = counted_cash_minor - expected_cash_minor
      AND (variance_minor = 0 OR length(btrim(variance_reason)) >= 3))
  )
);
-- One open shift per register: a second open attempt, however concurrent, fails.
CREATE UNIQUE INDEX register_sessions_one_open ON register_sessions (tenant_id, register_id) WHERE status = 'open';
CREATE INDEX register_sessions_store_idx ON register_sessions (tenant_id, store_id, opened_at DESC);

-- A sale's shift must be in the same store as the sale. Older sales have none.
ALTER TABLE orders ADD COLUMN register_session_id TEXT;
ALTER TABLE orders ADD FOREIGN KEY (tenant_id, store_id, register_session_id)
  REFERENCES register_sessions(tenant_id, store_id, id);
CREATE INDEX orders_register_session_idx ON orders (tenant_id, register_session_id)
  WHERE register_session_id IS NOT NULL;

-- Tenant access. Opening figures can never be edited, and a closed shift
-- cannot be changed at all (the restrictive policy below), so history is fixed.
GRANT SELECT, INSERT ON registers TO candella_catalog;
GRANT UPDATE (name) ON registers TO candella_catalog;
GRANT SELECT, INSERT ON register_sessions TO candella_catalog;
GRANT UPDATE (status, closed_by, closer_name_snapshot, closed_at, sales_count, cash_sales_minor,
              expected_cash_minor, counted_cash_minor, variance_minor, variance_reason)
  ON register_sessions TO candella_catalog;

ALTER TABLE registers ENABLE ROW LEVEL SECURITY;
ALTER TABLE registers FORCE ROW LEVEL SECURITY;
CREATE POLICY registers_tenant ON registers
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));

ALTER TABLE register_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE register_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY register_sessions_tenant ON register_sessions
  USING (tenant_id = current_setting('app.current_tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true));
-- WITH CHECK (true): without it the USING clause would also apply to the new
-- row, and closing (status -> 'closed') would be refused.
CREATE POLICY register_sessions_closed_final ON register_sessions AS RESTRICTIVE
  FOR UPDATE USING (status = 'open') WITH CHECK (true);

-- Platform console: read-only, across tenants (see 0025).
GRANT SELECT ON registers, register_sessions TO candella_platform;
CREATE POLICY registers_platform_read ON registers FOR SELECT TO candella_platform USING (true);
CREATE POLICY register_sessions_platform_read ON register_sessions FOR SELECT TO candella_platform USING (true);

-- +goose Down
DROP INDEX orders_register_session_idx;
ALTER TABLE orders DROP COLUMN register_session_id;
DROP TABLE register_sessions;
DROP TABLE registers;
