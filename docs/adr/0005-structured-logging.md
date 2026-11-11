# 0005 - Structured logs with a request id

Status: accepted (2026-11-11)

## Context
Gin's text access log cannot be filtered in an aggregator and has no
correlation id, so a report like "my message did not arrive" meant guessing
which lines belonged together.

## Decision
Use `log/slog`: JSON in `release`, text in `debug`. A middleware accepts
`X-Request-ID` or generates a UUID, echoes it, and logs method, path, status,
duration and client IP. Operational errors use `slog`; startup fatal errors
keep `log.Fatal`.

## Consequences
Every request is traceable by one id; message content, media and keys are never
logged; ingestion gets JSON without a logging library.
