DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS user_follows;
ALTER TABLE users DROP COLUMN IF EXISTS follow_approval_required;
