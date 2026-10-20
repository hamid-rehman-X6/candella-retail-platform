-- +goose Up
CREATE INDEX orders_report_store_date_idx ON orders (tenant_id, store_id, created_at);
CREATE INDEX returns_report_store_date_idx ON returns (tenant_id, store_id, created_at);
CREATE INDEX stock_movements_report_store_date_idx ON stock_movements (tenant_id, store_id, created_at);

-- +goose Down
DROP INDEX stock_movements_report_store_date_idx;
DROP INDEX returns_report_store_date_idx;
DROP INDEX orders_report_store_date_idx;
