-- +goose Up
-- The API used to query as the database login itself, which on local Docker is
-- a superuser that bypasses RLS: tenant isolation held only where a repository
-- remembered `SET LOCAL ROLE candella_catalog`. Each pool now switches into a
-- restricted role as soon as it connects (see pkg/database), so RLS applies to
-- every query by default, and a query made without a tenant sees no tenant rows.
--
--   candella_app      — the tenant-facing API. No RLS bypass. Ledgers stay
--                       append-only: it inherits candella_catalog's grants, which
--                       have no UPDATE/DELETE on stock movements, orders or
--                       payments, and it may only INSERT into audit_logs.
--   candella_platform — the platform console. Reads across tenants through a
--                       SELECT-only policy, writes nothing but its own
--                       last-access stamp, and cannot read password hashes, MFA
--                       secrets or sign-in tokens.
--
-- Roles are cluster-wide, so they are created only if missing; several
-- databases on one server (e.g. disposable test databases) can share them.

-- +goose StatementBegin
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'candella_app') THEN
    CREATE ROLE candella_app NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'candella_platform') THEN
    CREATE ROLE candella_platform NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
  END IF;
  -- The migration login can switch into both, as with candella_catalog. A
  -- separate production login needs the same grant (see the PR's rollout notes).
  EXECUTE format('GRANT candella_app, candella_platform TO %I', current_user);
END $$;
-- +goose StatementEnd

-- ----- candella_app ---------------------------------------------------------

GRANT USAGE ON SCHEMA public TO candella_app;
-- Tenant tables: exactly candella_catalog's grants, still filtered by RLS.
GRANT candella_catalog TO candella_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON
  users, sessions, oauth_accounts, verification_codes, password_reset_tokens,
  user_mfa, mfa_backup_codes
  TO candella_app;
-- Workspaces, memberships and invitations are soft-deleted, never removed.
GRANT SELECT, INSERT, UPDATE ON tenants, tenant_memberships, staff_invitations TO candella_app;
-- The audit trail is append-only for the API.
GRANT INSERT ON audit_logs TO candella_app;

-- ----- candella_platform ----------------------------------------------------

GRANT USAGE ON SCHEMA public TO candella_platform;
GRANT SELECT ON tenants, tenant_memberships, audit_logs, platform_admins TO candella_platform;
GRANT UPDATE (last_access_at) ON platform_admins TO candella_platform;
GRANT SELECT (id, email, full_name, status, email_verified_at, mfa_enabled,
              last_login_at, created_at, updated_at, deleted_at)
  ON users TO candella_platform;
GRANT SELECT (id, user_id, ip_address, user_agent, expires_at, revoked_at, created_at)
  ON sessions TO candella_platform;

-- Tenant tables: read-only, across tenants. This is a policy scoped to the
-- role, not a setting the tenant policies check, so no tenant request can
-- switch it on.
GRANT SELECT ON
  stores, store_access, products, product_variants, stock_levels, stock_movements,
  document_sequences, orders, order_items, payments, suppliers, supplier_variants,
  catalog_import_batches
  TO candella_platform;
CREATE POLICY stores_platform_read ON stores FOR SELECT TO candella_platform USING (true);
CREATE POLICY store_access_platform_read ON store_access FOR SELECT TO candella_platform USING (true);
CREATE POLICY products_platform_read ON products FOR SELECT TO candella_platform USING (true);
CREATE POLICY product_variants_platform_read ON product_variants FOR SELECT TO candella_platform USING (true);
CREATE POLICY stock_levels_platform_read ON stock_levels FOR SELECT TO candella_platform USING (true);
CREATE POLICY stock_movements_platform_read ON stock_movements FOR SELECT TO candella_platform USING (true);
CREATE POLICY document_sequences_platform_read ON document_sequences FOR SELECT TO candella_platform USING (true);
CREATE POLICY orders_platform_read ON orders FOR SELECT TO candella_platform USING (true);
CREATE POLICY order_items_platform_read ON order_items FOR SELECT TO candella_platform USING (true);
CREATE POLICY payments_platform_read ON payments FOR SELECT TO candella_platform USING (true);
CREATE POLICY suppliers_platform_read ON suppliers FOR SELECT TO candella_platform USING (true);
CREATE POLICY supplier_variants_platform_read ON supplier_variants FOR SELECT TO candella_platform USING (true);
CREATE POLICY catalog_import_batches_platform_read ON catalog_import_batches FOR SELECT TO candella_platform USING (true);

-- +goose Down
DROP POLICY catalog_import_batches_platform_read ON catalog_import_batches;
DROP POLICY supplier_variants_platform_read ON supplier_variants;
DROP POLICY suppliers_platform_read ON suppliers;
DROP POLICY payments_platform_read ON payments;
DROP POLICY order_items_platform_read ON order_items;
DROP POLICY orders_platform_read ON orders;
DROP POLICY document_sequences_platform_read ON document_sequences;
DROP POLICY stock_movements_platform_read ON stock_movements;
DROP POLICY stock_levels_platform_read ON stock_levels;
DROP POLICY product_variants_platform_read ON product_variants;
DROP POLICY products_platform_read ON products;
DROP POLICY store_access_platform_read ON store_access;
DROP POLICY stores_platform_read ON stores;

-- Privileges in this database only; the roles themselves are left in place,
-- because other databases on the same server may still use them.
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM candella_platform;
REVOKE USAGE ON SCHEMA public FROM candella_platform;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM candella_app;
REVOKE candella_catalog FROM candella_app;
REVOKE USAGE ON SCHEMA public FROM candella_app;
