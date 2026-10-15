-- +goose Up
-- Candella serves Pakistani shops, so workspaces run on Pakistan Standard Time.
ALTER TABLE tenants ALTER COLUMN timezone SET DEFAULT 'Asia/Karachi';

-- Move existing UTC workspaces to Pakistan time. Every stored timestamp is a
-- timestamptz (an absolute instant), so nothing is rewritten: this only changes
-- how times are displayed and where a sales-history "day" begins and ends.
-- Workspaces deliberately set to another zone are left alone.
UPDATE tenants SET timezone = 'Asia/Karachi' WHERE timezone = 'UTC';

-- +goose Down
-- Restores the old default only. Once applied there is no record of which
-- workspaces were originally UTC, so the conversion above is not reversed.
ALTER TABLE tenants ALTER COLUMN timezone SET DEFAULT 'UTC';
