-- Reverse cleanly so the migration chain can rebuild: drop the index, refill
-- the blanks that were nulled, and restore the column's original NOT NULL shape.
DROP INDEX IF EXISTS users_username_lower_key;
UPDATE users SET username = '' WHERE username IS NULL;
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
