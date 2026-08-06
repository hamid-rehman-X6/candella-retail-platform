-- +goose Up
-- A tenant is a customer's business/workspace — the top-level isolation boundary.
-- (currency_code is a plain column for now; a currencies lookup table can be added
-- later without changing this shape.)
CREATE TABLE tenants (
  id            TEXT PRIMARY KEY CHECK (id ~ '^tnt_[0-9A-Za-z]{12}$'),
  name          TEXT NOT NULL,
  slug          TEXT NOT NULL,
  industry_type TEXT NOT NULL CHECK (industry_type IN (
                  'pos_general','pharmacy','garments','cosmetics',
                  'grocery','electronics','wholesale','other')),
  status        TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('trialing','active','suspended','cancelled')),
  currency_code CHAR(3) NOT NULL DEFAULT 'USD',
  timezone      TEXT NOT NULL DEFAULT 'UTC',
  created_by    TEXT REFERENCES users(id),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at    TIMESTAMPTZ
);

-- Slug is unique among live tenants (soft-deleted rows don't block reuse).
CREATE UNIQUE INDEX tenants_slug_key ON tenants (slug) WHERE deleted_at IS NULL;

CREATE TRIGGER tenants_set_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE tenants;
