# Threat model

Scope: the two trust boundaries that can fail silently. The client device and
its keys are trusted; the server and its storage are not trusted with plaintext.

1. **Auth and WebSocket boundary** - login, tokens, WS handshake.
2. **Media pipeline** - upload, storage, download and confirmation.

The tables list the mitigation and the test that proves it. "Missing" means the
mitigation is not implemented yet; those items are tracked in the Engineering
Standard backlog, not silently accepted.

## 1. Auth and WebSocket

| STRIDE | Threat | Mitigation | Test |
| --- | --- | --- | --- |
| Spoofing | Attacker logs in as another user | Login is an RSA-signed JWT verified against the stored 4096-bit public key | `e2e`: register via invite, RSA auth and refresh |
| Spoofing | A captured login JWT is replayed | The JWT must carry a server-issued `nonce` (`GET /api/auth/challenge`), spent with `GETDEL` on use | `e2e`: the login token is single use |
| Spoofing | Stolen WS token is replayed | WS token is single-use: `GETDEL` in Valkey plus short TTL | `e2e`: WS handshake consumes the one-time token |
| Tampering | Forged access token | HS256 signature with a server secret; `TokenValid` rejects other algorithms and requires the matching `typ` claim | `utils`: `token_test.go` |
| Tampering | A client timestamps a message into the far future or onto an existing queue entry | `sentAt` must be within five minutes of the server clock | `e2e`: a message with an implausible timestamp is refused |
| Tampering | Crafted claims crash the server | Type assertions in `TokenValid` are checked and return an error | `utils` unit tests + malformed-token path |
| Repudiation | User denies sending a message | No server-side audit log of message metadata | **Missing** - by design, but worth an explicit decision |
| Information disclosure | Server reads message content | Messages are AES-GCM encrypted on the device; the server stores ciphertext | `e2e`: online and offline message scenarios; app E2E message test |
| Denial of service | Brute force login, invite or media endpoints | Per client IP token bucket on register, auth, challenge and upload (`RateLimit`); only the reverse proxy may set `X-Forwarded-For` | `middlewares`: `rate_limit_test.go` |
| Denial of service | One account floods another user's queue | Per receiver queue counter with a hard cap (`MESSAGE_MAX_QUEUED`) | `e2e`: an offline queue has a per user cap |
| Denial of service | One account floods the WebSocket | Per connection message rate limiter, the connection is closed on abuse | `utils`: `message_rate_test.go` |
| Denial of service | One account fills the disk with media | Per account pending media quota (`MEDIA_MAX_PER_USER`) | `e2e`: a user has a media quota |
| Elevation of privilege | Non-admin creates invites | Admin flag checked server-side in invite handlers | not covered by E2E yet |
| Elevation of privilege | Refresh token is not revocable | Refresh token carries an id recorded in Valkey; account deletion revokes the session set | `e2e`: a deleted user loses API access (refresh is rejected) |
| Spoofing | A WS token is read from a proxy access log | The client sends it in the `Authorization` header; the query parameter is kept only for older builds | code review |
| Elevation of privilege | A stolen refresh token is reused | A refresh token is single use: it is consumed on refresh and replaced by a rotated one | `e2e`: a refresh token is single use and rotates |
| Elevation of privilege | A deleted account keeps messaging on an open socket | Every message re-checks the sender; the socket is closed when the account is deleted | `e2e` + `hub` sender check |
| Elevation of privilege | A revoked session keeps working | **Partial.** Revoking the session set also breaks the RSA login from that point, because the login needs a fresh challenge — but a device holding the private key can request a new challenge and log in again. A hard lockout needs a device registry (revoke a device, not a session) or account deletion | `e2e`: a revoked refresh session is refused; hard lockout not implemented |

## 2. Media pipeline

| STRIDE | Threat | Mitigation | Test |
| --- | --- | --- | --- |
| Spoofing | Someone other than the receiver downloads media | Metadata binds sender and receiver; download checks `receiverId` | `e2e`: media upload and download keep only ciphertext |
| Spoofing | Path traversal via media id | `MediaFilePath` uses `filepath.Base`; ids are UUIDs | server E2E + code review |
| Tampering | Server alters media bytes | AES-GCM MAC is verified by the client after download | app E2E: media upload sends only ciphertext and receiver decrypts it |
| Information disclosure | Server reads or infers media | Server stores the encrypted blob and no MIME/plaintext; type, key and caption travel inside the encrypted message | `e2e`: stored and downloaded bytes equal the uploaded ciphertext |
| Information disclosure | Media outlives its message | Valkey TTL removes metadata; a subscriber deletes the file, and a periodic sweep removes any file whose metadata key is gone | `e2e`: sweeper removes media whose expiry notification was missed |
| Denial of service | Oversized upload | The request body is capped with `http.MaxBytesReader` and the copy with `io.LimitReader`, so nothing beyond `MEDIA_MAX_UPLOAD_SIZE` reaches the disk | `e2e`: an oversized upload is rejected before it is stored |
| Denial of service | Unbounded uploads | Per client IP token bucket on upload; no per-user quota yet | **Missing** - the same address behind a NAT shares one bucket |

## Out of scope

- Compromised client device or stolen unlocked phone.
- Firebase account compromise.
- Physical access to the VPS.
- Supply-chain attacks beyond Dependabot, CodeQL, Snyk and gitleaks.
