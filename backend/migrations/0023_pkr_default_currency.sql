-- +goose Up
-- Candella serves Pakistani retailers, so new workspaces price in PKR.
ALTER TABLE tenants ALTER COLUMN currency_code SET DEFAULT 'PKR';

-- Relabel existing USD workspaces as PKR. This is a relabel, NOT a currency
-- conversion: price_minor / cost_minor values keep their numbers, and both
-- currencies use two minor digits, so 1250 means $12.50 before and Rs 12.50
-- after. Safe only because every USD workspace so far is pre-launch test data;
-- operators should re-check their prices afterwards.
--
-- orders.currency_code and payments.currency_code are deliberately left alone:
-- they are snapshots of what was charged at the time, and rewriting them would
-- falsify past receipts.
UPDATE tenants SET currency_code = 'PKR' WHERE currency_code = 'USD';

-- +goose Down
-- Restores the old default only. The relabel above is not reversed, because
-- once applied there is no record of which workspaces were originally USD.
ALTER TABLE tenants ALTER COLUMN currency_code SET DEFAULT 'USD';
