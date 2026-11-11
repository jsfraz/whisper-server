---
name: Contract change
description: Change the HTTP API contract (routes, models) and propagate it to the generated Dart client and the Flutter app without drift.
---

# Contract change

Use whenever a route, request/response model changes or an endpoint is added.

1. Change the handlers and models in `whisper-server`.
2. Generate the spec: `go run ./cmd/spec`.
3. Review the `openapi.json` diff:
   - new/changed operations must appear in `paths`,
   - for raw Gin handlers (media), update `routes/media_openapi.go` as well,
   - if this is a breaking change (removed field, changed type, new required field), require a major bump and the `breaking-change` label.
4. Verify lint and tests: `golangci-lint run`, `go test ./...`.
5. In `whisper_openapi_client_dart` run `./scripts/regen.sh` and commit the result together with the spec change.
6. If a method signature changed, update the call sites in `whisper/lib` and verify with `flutter analyze --no-fatal-infos`.
7. Bump the client version in `whisper/pubspec.yaml` (the path dependency only changes locally; releases pin it to a tag).

Never edit the generated files in `whisper_openapi_client_dart/lib` by hand. Custom logic belongs in hand-written extensions (`lib/media_download.dart`) and must survive regeneration.
