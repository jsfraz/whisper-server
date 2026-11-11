# 0003 - Go E2E against real dependencies

Status: accepted (2026-11-11)

## Context
Unit tests cannot prove auth, WebSocket, offline-queue or media flows, and
mocks would mostly test the mocks.

## Decision
Run `e2e/` against the real Gin router and temporary SQLite, with Valkey from
testcontainers: invite registration with RSA auth, one-time WS handshake,
online delivery, offline queue and reconnect, media ciphertext upload/download.

## Consequences
CI needs Docker and the suite runs on every PR; tests skip only when Docker is
unavailable. It is not a load test and does not cover Firebase push delivery.
