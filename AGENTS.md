# whisper-server — agent contract

Go 1.27 / Gin API for the Whisper messenger: SQLite+SQLCipher (GORM), Valkey, WebSocket, Firebase push.
Clients: `whisper` (Flutter), `whisper_openapi_client_dart` (generated from `openapi.json`), `whisper_websocket_client_dart`.

## Commands

- build: `go build ./...`
- test: `go test ./...`
- lint: `golangci-lint run` (config in `.golangci.yml`; CI gates new lines only)
- spec: `go run ./cmd/spec` — rewrites `openapi.json`; must be part of every commit that changes routes or models
- locally: `docker compose -f docker-compose.dev.yml up -d` (Valkey) + env from `.env.dev`, then `go run .`

## Hard rules

1. Never commit secrets, keys, `.env*` or Firebase credentials.
2. `openapi.json` is a versioned artifact. Do not generate it with `wget` against a running server — only `go run ./cmd/spec`, so the output stays deterministic.
3. Never edit `whisper_openapi_client_dart/lib` by hand. A contract change means: change the spec here, regenerate the client, bump the version in `whisper/pubspec.yaml`, all in one sequence.
4. Do not weaken auth or encryption, not even "temporarily for debugging" (CORS, `CheckOrigin`, token validation).
5. Do not add a dependency without justification in the PR.
6. Do not change the DB schema without a migration (goose, phase 2) and do not change a Valkey payload without the `v` field.
7. Never disable a test or lint to get CI green. Fix it, or file an issue and escalate.
8. Do not deploy without green CI; production only through a tag.
9. Force-push only to your own feature branch, never to `main`.

## Where it hurts

- **Media endpoints** (`POST /api/media`, `GET /api/media/{id}`) are raw Gin handlers (multipart, streamed response). Fizz cannot describe them, so their OpenAPI operations are hand-written in `routes/media_openapi.go` (see the `shortcut:` comment). When you change a media handler, update that file too, otherwise client and server drift apart.
- **Binary responses**: the client generator maps them to `MultipartFile`, so the download is handled by the hand-written `lib/media_download.dart`, not by the generated `downloadMedia`.
- **The server only ever holds ciphertext.** Never log message content, media or keys.

## Definition of done

- New non-trivial logic has a test (`*_test.go`).
- `golangci-lint run` is clean on the files you touched.
- A route or model change updates `openapi.json`.
- A breaking contract change means a major bump plus the `breaking-change` label on the PR.
- Changelog entry if the repo has a changelog.

## When unsure

Stop and ask. Security decisions are not up for guessing; prefer an issue over a silent compromise.
