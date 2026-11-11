# 0001 - goose migrations instead of GORM AutoMigrate

Status: accepted (2026-11-11)

## Context
AutoMigrate silently adds columns, cannot roll back or migrate data, and the
production database has no schema version.

## Decision
Use `pressly/goose` with embedded SQL in `database/migrations/`, run `goose.Up`
on startup. The first migration uses `CREATE TABLE IF NOT EXISTS` so it adopts
databases created by AutoMigrate.

## Consequences
Schema changes are explicit files, production is forward-only, and a new table
needs a numbered migration instead of a model change alone.
