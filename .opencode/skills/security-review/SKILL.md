---
name: Security review
description: Review an auth, crypto, media or session change in whisper-server for security regressions before merge.
---

# Security review

Use before merging a change in `handlers/`, `middlewares/`, `utils/token.go`, `utils/crypto.go`, `database/ws_auth.go` or `handlers/media.go`.

Walk the diff and answer every point. A finding without a countermeasure means request changes.

1. **Auth**: does the middleware verify the signature, the expiry and the existence of the user? Does the change introduce a bypass (`CheckOrigin`, custom header, debug bypass)?
2. **Tokens**: the refresh token is still not revocable (known debt) — does the change make it worse? Is the WS token single-use (GETDEL) and short-lived?
3. **Input**: does every request field have validation and a limit? Do path/query parameters avoid reaching the filesystem without `filepath.Clean` plus a prefix check?
4. **Files**: does upload enforce a size limit, does download verify the requester is the receiver? Is there any path where the server sees plaintext?
5. **Leaks**: are tokens, keys, message content or e-mails logged? Do error responses avoid leaking internal state ("user exists", SQL errors)?
6. **DoS**: does a new path have unbounded per-request work (large allocations, N+1 Valkey calls, blocking the hub goroutine)?
7. **Dependencies**: does a new library have a justification, and is it not a crypto library added without need?

Report as a table: point → OK / finding → proposed fix. Never approve weakening auth or encryption, even when it is labelled temporary.
