-- +goose Up
-- User ids become unpredictable UUIDs. This is a clean break: the old sequential
-- ids cannot be mapped to UUIDs without the clients, so existing rows are dropped
-- and users register again with a fresh invite. Valkey state (msg:*, fcm:*,
-- del:users) keyed by the old numeric ids is left to expire on its own.
CREATE TABLE users_v2 (
    id text PRIMARY KEY,
    username text,
    public_key text,
    admin numeric
);
DROP TABLE users;
ALTER TABLE users_v2 RENAME TO users;

-- +goose Down
-- Not reversible: the previous ids and rows are gone.
CREATE TABLE users_v1 (
    id integer PRIMARY KEY AUTOINCREMENT,
    username text,
    public_key text,
    admin numeric
);
DROP TABLE users;
ALTER TABLE users_v1 RENAME TO users;
