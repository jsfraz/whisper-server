-- +goose Up
-- Registration used to check the username and then insert, which let two
-- concurrent requests (with two different invites) create the same name. The
-- database now enforces uniqueness. Rows that already collide are renamed once
-- (their owner keeps a working account; the id suffix keeps it unique).
UPDATE users
SET username = username || '-' || substr(id, 1, 8)
WHERE id NOT IN (SELECT MIN(id) FROM users GROUP BY username);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);

-- +goose Down
DROP INDEX IF EXISTS idx_users_username;
