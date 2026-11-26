# 0008 - Ephemeral metadata: relay only, never stored

Status: accepted (2026-11-21)

## Context
F4 adds typing, presence, live location, disappearing messages and view-once.
The exit criterion is that all of them can be switched off, and with them off the
server learns nothing beyond delivery. Typing and presence therefore may not
become rows, cursors or push; a disappearing message must still expire though,
and only the server can delete its ciphertext.

## Decision
Typing, presence and live location travel as `typing`/`presence`/`liveLocation`
WebSocket frames. The server relays each to the live devices of a peer, checks
that the two accounts have a conversation, and keeps nothing: no row, no
timestamp, no archive, no push. Last seen is stamped by the server into the
relay frame. A message may carry `ttlSeconds` (1..604800); it is folded into the
existing `expires_at`, so a disappearing message keeps one storage mechanism and
one sweeper. Per-chat agreement and the view-once flag stay inside the
ciphertext: the server never parses `content`.

## Consequences
No migration and no new table. Presence dies with the process and is not shared
across instances; a client that never sends the frames leaves no trace. The
server accepts the lifetime of a message as visible metadata, which is the price
of being able to delete it.
