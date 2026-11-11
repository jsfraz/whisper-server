-- +goose Up
-- IF NOT EXISTS adopts databases created by the old AutoMigrate baseline.
CREATE TABLE IF NOT EXISTS users (
    id integer PRIMARY KEY AUTOINCREMENT,
    username text,
    public_key text,
    admin numeric
);

-- +goose Down
DROP TABLE IF EXISTS users;
