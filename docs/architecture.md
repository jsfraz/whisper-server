# Architecture

Whisper is four repositories with one runtime system. The app and both Dart
clients live on the device; the Go server is the only component that talks to
SQLite, Valkey and Firebase.

## System context

```mermaid
C4Context
  title Whisper - system context

  Person(user, "User", "Runs the Whisper app on a phone or Linux desktop")
  System(app, "Whisper app", "Flutter client; owns keys and plaintext")
  System(server, "Whisper server", "Go/Gin; routes, auth, WebSocket, media")
  SystemDb(sqlite, "SQLite + SQLCipher", "Users and admin flags")
  System(valkey, "Valkey", "Invites, offline messages, media metadata, WS tokens, push tokens")
  System_Ext(fcm, "Firebase Cloud Messaging", "Push notifications")

  Rel(user, app, "Uses")
  Rel(app, server, "HTTPS and WSS", "ciphertext only")
  Rel(server, sqlite, "Reads and writes users")
  Rel(server, valkey, "Queues and TTL data")
  Rel(server, fcm, "Sends push notifications")
```

## Containers

```mermaid
C4Container
  title Whisper - containers

  Person(user, "User", "")

  Container(app, "Whisper app", "Flutter 3.47 / Dart 3.13", "UI, RSA auth, AES-GCM encryption, Hive cache")
  Container(openapi, "OpenAPI client", "Generated Dart", "HTTP contract generated from the server spec")
  Container(wsclient, "WebSocket client", "Dart", "Typed WS messages and one-time token handshake")

  Container(api, "API", "Go 1.27 / Gin / Fizz", "Auth, users, invites, media, WebSocket hub")
  ContainerDb(db, "SQLite", "SQLCipher", "users table (goose migrations)")
  Container(valkey, "Valkey", "Valkey", "invite:*, msg:*, media:*, ws:*, fcm:*, del:users")
  System_Ext(fcm, "Firebase Cloud Messaging", "")

  Rel(user, app, "Uses")
  Rel(app, openapi, "HTTP")
  Rel(app, wsclient, "WebSocket")
  Rel(openapi, api, "HTTPS", "JSON")
  Rel(wsclient, api, "WSS", "binary JSON")
  Rel(api, db, "users")
  Rel(api, valkey, "state and queues")
  Rel(api, fcm, "push")
```

The server never sees plaintext messages or media. It stores ciphertext in
transport, metadata in Valkey with a TTL, and the encrypted media blob on disk.

## Dependency rules

- `routes` only wires routes and middleware to handlers; no business logic.
- `handlers` may call `database`, `models` and existing `utils` helpers. They
  must not implement crypto or token logic themselves; that belongs in `utils`.
- `database` owns SQLite and Valkey access. It must not import `handlers`,
  `routes` or Gin.
- `utils` holds shared primitives (crypto, tokens, hub, logger). It must not
  import `handlers`, `routes` or `database`.
- Clients talk HTTP/WS only. They never know about SQLite, Valkey or the DB
  schema; the OpenAPI spec is the contract.
- The app never edits generated client code; contract changes go through
  `whisper-server` and `scripts/regen.sh`.

These rules are documented, not enforced by a linter: the handler rule is about
behaviour (not importing a package) and a `depguard` gate would not catch it.
Review and the `contract-change` skill are the enforcement.

## Deliberate non-goals

- No monorepo and no platform repository.
- No staging environment; production deploys are tag-driven.
- No component-level C4 diagrams; they would drift faster than they help.
