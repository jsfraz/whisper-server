# 0007 - PostgreSQL instead of SQLite, with the ciphertext archive in the same database

Status: accepted (2026-11-17)

## Context
SQLite was only ever meant to hold accounts; messages lived in Valkey with a
one-month TTL. History, delta sync and per-user retention make the archive a
first-class dataset: it needs pagination, a cursor, and deletes by expiry. Valkey
is memory-bound, has no query model and its TTL is a delivery timer, not a
retention policy. The app has no production data yet, so there is nothing to
migrate.

## Decision
One PostgreSQL database holds users, devices and the ciphertext archive, behind
goose migrations. Valkey keeps only short-lived state (invites, sessions, WS
tokens, media metadata, push targets). A message row is the history, the offline
buffer and the delta source at once, ordered by a monotonic `seq`. `expires_at`
comes from the shorter retention of both participants; an hourly sweeper deletes
expired batches.

## Consequences
SQLCipher is gone: the database volume is now the thing to protect, and backups
are `pg_dump`. One more service in Compose and one secret (`DATABASE_URL`).
Moving a message to another backend means one package (`database/messages.go`)
plus its migrations; the API never exposes the storage shape.
