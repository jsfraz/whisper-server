# 0003 - Go E2E against real dependencies

Status: accepted (2026-11-11)

## Context

Unit tests cannot prove the auth, WebSocket, offline queue and media flows.
Mocks would test the mocks, especially for the one-time WS token and the
Valkey TTL behaviour.

## Decision

Run E2E tests in `e2e/` against the real Gin router and a temporary SQLite
database, with Valkey started by testcontainers. Cover five flows: invite
registration with RSA auth, WS one-time handshake, online delivery, offline
queue and reconnect, media ciphertext upload/download.

## Consequences

- CI needs Docker; tests skip only when Docker is unavailable.
- The suite runs on every PR and asserts observable behaviour, not internals.
- It is not a load test and does not cover Firebase push delivery.
