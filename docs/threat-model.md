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
| Spoofing | Stolen WS token is replayed | WS token is single-use: `GETDEL` in Valkey plus short TTL | `e2e`: WS handshake consumes the one-time token |
| Tampering | Forged access token | HS256 signature with a server secret; `TokenValid` rejects other algorithms | covered indirectly by auth middleware and E2E auth |
| Tampering | Crafted claims crash the server | Type assertions in `TokenValid` are unchecked | **Missing** - panic on crafted claims, see backlog |
| Repudiation | User denies sending a message | No server-side audit log of message metadata | **Missing** - by design, but worth an explicit decision |
| Information disclosure | Server reads message content | Messages are AES-GCM encrypted on the device; the server stores ciphertext | `e2e`: online and offline message scenarios; app E2E message test |
| Denial of service | Brute force login, invite or media endpoints | No rate limiting and `AllowAllOrigins = true` | **Missing** - see backlog |
| Elevation of privilege | Non-admin creates invites | Admin flag checked server-side in invite handlers | not covered by E2E yet |
| Elevation of privilege | Refresh token never expires or is not revocable | Refresh token is a long-lived JWT; no rotation or revocation | **Missing** - see backlog |

## 2. Media pipeline

| STRIDE | Threat | Mitigation | Test |
| --- | --- | --- | --- |
| Spoofing | Someone other than the receiver downloads media | Metadata binds sender and receiver; download checks `receiverId` | `e2e`: media upload and download keep only ciphertext |
| Spoofing | Path traversal via media id | `MediaFilePath` uses `filepath.Base`; ids are UUIDs | server E2E + code review |
| Tampering | Server alters media bytes | AES-GCM MAC is verified by the client after download | app E2E: media upload sends only ciphertext and receiver decrypts it |
| Information disclosure | Server reads or infers media | Server stores the encrypted blob and no MIME/plaintext; type, key and caption travel inside the encrypted message | `e2e`: stored and downloaded bytes equal the uploaded ciphertext |
| Information disclosure | Media outlives its message | Valkey TTL removes metadata; a subscriber deletes the file | **Partial** - deletion is not atomic if the subscriber is down; see backlog |
| Denial of service | Oversized upload | `MEDIA_MAX_UPLOAD_SIZE` is enforced before writing | no dedicated test |
| Denial of service | Unbounded uploads/downloads | No per-user rate limiting | **Missing** - see backlog |

## Out of scope

- Compromised client device or stolen unlocked phone.
- Firebase account compromise.
- Physical access to the VPS.
- Supply-chain attacks beyond Dependabot, CodeQL, Snyk and gitleaks.
