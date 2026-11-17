# Architecture

Whisper is four repositories with one runtime system. The app and both Dart
clients live on the device; the Go server is the only component that talks to
PostgreSQL, Valkey and Firebase.

## System context

```mermaid
C4Context
  title Whisper - system context

  Person(user, "User", "Runs the Whisper app on a phone or Linux desktop")
  System(app, "Whisper app", "Flutter client; owns keys and plaintext")
  System(server, "Whisper server", "Go/Gin; routes, auth, WebSocket, media")
  SystemDb(postgres, "PostgreSQL", "Accounts, devices, ciphertext archive")
  System(valkey, "Valkey", "Invites, sessions, media metadata, WS tokens, push tokens")
  System_Ext(fcm, "Firebase Cloud Messaging", "Push notifications")

  Rel(user, app, "Uses")
  Rel(app, server, "HTTPS and WSS", "ciphertext only")
  Rel(server, postgres, "Reads and writes accounts and ciphertext")
  Rel(server, valkey, "Short-lived state")
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
  ContainerDb(db, "PostgreSQL", "PostgreSQL 17", "users, devices, messages (goose migrations)")
  Container(valkey, "Valkey", "Valkey", "invite:*, media:*, media:user:*, ws:*, fcm:*, sessions:*, del:users")
  System_Ext(fcm, "Firebase Cloud Messaging", "")

  Rel(user, app, "Uses")
  Rel(app, openapi, "HTTP")
  Rel(app, wsclient, "WebSocket")
  Rel(openapi, api, "HTTPS", "JSON")
  Rel(wsclient, api, "WSS", "binary JSON")
  Rel(api, db, "accounts and ciphertext")
  Rel(api, valkey, "state and queues")
  Rel(api, fcm, "push")
```

The server never sees plaintext messages or media. It stores ciphertext in
transport, metadata in Valkey with a TTL, and the encrypted media blob on disk.

One ciphertext row per message is the archive, the history and the offline
delivery buffer at once: `messages` has a monotonic `seq`, a conversation key
(the sorted participant pair) and an `expires_at` from the shorter retention of
both accounts. A client that reconnects sends a `sync` frame with the last `seq`
it stored and receives only the delta; a resent message collides on
`(sender_id, receiver_id, sent_at)` and keeps its original row. A device that is
revoked loses its `sessions:<userId>:<deviceId>` set and its live socket, and
every token carries the device id, so the lockout starts on the next request.

## Dependency rules

- `routes` only wires routes and middleware to handlers; no business logic.
- `handlers` may call `database`, `models` and existing `utils` helpers. They
  must not implement crypto or token logic themselves; that belongs in `utils`.
- `database` owns PostgreSQL and Valkey access. It must not import `handlers`,
  `routes` or Gin.
- `utils` holds shared primitives (crypto, tokens, the connection registry,
  logger). It must not import `handlers`, `routes` or `database`.
- `push` is the only place that talks to Firebase Messaging; handlers call it.
- Clients talk HTTP/WS only. They never know about PostgreSQL, Valkey or the DB
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
