-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

-- Shared trigger that stamps updated_at on every UPDATE, so application code
-- never has to remember to set it (schema design §1.4).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS set_updated_at();
-- +goose StatementEnd
DROP EXTENSION IF EXISTS citext;
