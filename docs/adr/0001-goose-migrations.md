# 0001 - goose migrations instead of GORM AutoMigrate

Status: accepted (2026-11-11)

## Context

The schema was created by `AutoMigrate(&models.User{})`. It silently adds
columns, cannot express a rollback or a data migration, and the production
database has no record of which schema version it is on.

## Decision

Use `pressly/goose` with embedded SQL files in `database/migrations/`. Run
`goose.Up` on startup. The first migration uses `CREATE TABLE IF NOT EXISTS` so
it adopts databases created by AutoMigrate.

## Consequences

- Schema changes are explicit files and show up in review.
- Rollbacks are possible in development; production stays forward-only.
- A new table means a new numbered migration, never a model change alone.
