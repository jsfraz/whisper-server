# 0006 - User ids are UUIDs

Status: accepted (2026-11-13)

## Context
User ids were sequential SQLite `AUTOINCREMENT` integers. They are handed to
clients and used in URLs, WebSocket payloads, JWT `sub` claims, media metadata
and Valkey keys (`msg:<id>_*`, `fcm:<id>`), so a numeric id lets anyone
enumerate accounts and guess ids of other users.

## Decision
A user id is a random UUID (`uuid.NewString()`), created in
`models.NewUser`. The column becomes `text PRIMARY KEY` in migration
`00002_user_ids_to_uuid.sql`, the JWT `sub` claim carries the UUID string, and
every Valkey key that embeds a user id is formatted with `%s`.

The migration is a **clean break**: the old numeric ids cannot be mapped to
UUIDs on the server (the mapping would have to live on the clients), so the
migration drops the existing rows and the first start issues a new admin
invite. Queued Valkey state keyed by numeric ids (`msg:*`, `fcm:*`,
`del:users`) is left to expire, except the delete-list set, whose non-UUID
members are ignored.

Client consequences: `whisper_openapi_client_dart` is regenerated (ids are
`String`), `whisper_websocket_client_dart` models change type, and the Flutter
app's Hive models store the id as a string. Local app data written before this
change cannot be read by the new adapters, so existing installs must log out
(delete local cache) and register again with a fresh invite.

## Consequences
Ids are unpredictable, at the cost of a one-time re-registration for every
account and of re-creating the whole test account setup in the dev environment.
New accounts need no extra work: the id is assigned on insert.
