-- A username identifies a user's public page (/u/<username>), so it has to be
-- unique. Two things are handled together here:
--
--   1. Accounts created from a user-created event have no username until the
--      person picks one. That unset state becomes NULL rather than '' -- a
--      unique index treats NULLs as distinct, so any number of not-yet-named
--      accounts coexist, whereas '' would all collide on one value.
--   2. Names are unique without regard to case: /u/Weeb and /u/weeb must not be
--      two different people, so the index is on lower(username).
--
-- If this fails with a unique_violation, two existing rows already share a
-- name (case-insensitively); resolve those before re-running.
ALTER TABLE users ALTER COLUMN username DROP NOT NULL;
UPDATE users SET username = NULL WHERE username = '';
CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_key ON users (lower(username));
