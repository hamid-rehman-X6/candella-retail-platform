-- +goose Up
ALTER TABLE orders ADD COLUMN cashier_name_snapshot TEXT NOT NULL DEFAULT '';
UPDATE orders o SET cashier_name_snapshot = u.full_name
FROM users u WHERE o.created_by = u.id;
CREATE INDEX orders_tenant_cashier_created_idx
  ON orders (tenant_id, created_by, created_at DESC, id DESC);

-- +goose Down
DROP INDEX orders_tenant_cashier_created_idx;
ALTER TABLE orders DROP COLUMN cashier_name_snapshot;
