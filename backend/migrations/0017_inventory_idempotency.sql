-- +goose Up
ALTER TABLE stock_movements ADD COLUMN idempotency_key TEXT;
ALTER TABLE stock_movements ADD COLUMN request_hash TEXT;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_idempotency_fields_check
  CHECK ((idempotency_key IS NULL AND request_hash IS NULL) OR
         (idempotency_key IS NOT NULL AND request_hash IS NOT NULL AND
          length(idempotency_key) BETWEEN 8 AND 100 AND length(request_hash) = 64));
CREATE UNIQUE INDEX stock_movements_tenant_idempotency_idx
  ON stock_movements (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

-- +goose Down
DROP INDEX stock_movements_tenant_idempotency_idx;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_idempotency_fields_check;
ALTER TABLE stock_movements DROP COLUMN request_hash;
ALTER TABLE stock_movements DROP COLUMN idempotency_key;
