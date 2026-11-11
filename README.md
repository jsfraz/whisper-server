# whisper-server

Secure private self-hosted messaging server using end-to-end encryption.

For Whisper app, see [whisper](https://github.com/jsfraz/whisper) repository.

Also see [Wiki](https://github.com/jsfraz/whisper-server/wiki)!

## Development

```bash
sudo docker compose -f docker-compose.dev.yml --env-file .env.dev up -d
```

### Environment variables

| Variable | Description | Required | Default value |
| --- | --- | --- | --- |
| GIN_MODE | Gin framework and entire application mode (debug or release) | No | debug |
| SERVER_URL | Server URL | Yes | - |
| SQLITE_PASSWORD | Password for the SQLite database | Yes | - |
| VALKEY_HOST | Valkey host | Yes | - |
| VALKEY_PORT | Valkey port | No | 6379 |
| VALKEY_PASSWORD | Password for Valkey | Yes | - |
| ADMIN_MAIL | Administrator email address | Yes | - |
| ADMIN_INVITE_TTL | Administrator invitation validity period (in seconds) | No | 600 (10 minutes) |
| INVITE_TTL | Invitation validity period (in seconds) | No | 900 (15 minutes) |
| SMTP_HOST | SMTP host for sending emails | Yes | - |
| SMTP_PORT | SMTP port | No | 465 |
| SMTP_USER | SMTP username | Yes | - |
| SMTP_PASSWORD | SMTP password | Yes | - |
| ACCESS_TOKEN_SECRET | Secret key for access token | Yes | - |
| ACCESS_TOKEN_LIFESPAN | Access token lifespan (in seconds) | No | 900 (15 minutes) |
| REFRESH_TOKEN_SECRET | Secret key for refresh token | Yes | - |
| REFRESH_TOKEN_LIFESPAN | Refresh token lifespan (in seconds) | No | 604800 (7 days) |
| WS_ACCESS_TOKEN_SECRET | Secret key for short-lived WebSocket access token | Yes | - |
| WS_ACCESS_TOKEN_LIFESPAN | WebSocket access token lifetime (in seconds) | No | 10 (10 seconds) |
| MESSAGE_TTL | Message retention time (in seconds) | No | 2592000 (30 days) |

### Firebase

Visit [docs](https://firebase.google.com/docs/admin/setup) to set up Firebase Admin SDK and export Firebase credentials in `.json` file that you will need later. (mentioned in `docker-compose.prod.yml`)

## CI/CD Deployment

Three GitHub Actions workflows run in this repository:

- `ci.yml` — on pull requests and on pushes to `main`: build, tests, lint (new code only), OpenAPI spec drift, `oasdiff` breaking changes, secret scan, dependency scan.
- `codeql.yml` — code scanning on `main` and weekly.
- `deploy-to-vps.yml` — **after `ci.yml` succeeds on `main`**, deploys the current `main` to the VPS. It does not run on a bare push.

Required repository secrets (Settings > Secrets and variables > Actions):

| Secret | Used by | Description |
| --- | --- | --- |
| `VPS_HOST` | deploy | IP address or domain name of the VPS server |
| `VPS_USERNAME` | deploy | Username for SSH access |
| `VPS_SSH_KEY` | deploy | Private SSH key for server access |
| `VPS_PORT` | deploy | SSH port (usually 22) |
| `PROJECT_PATH` | deploy | Absolute path to the directory to clone the project to on the VPS server |
| `FIREBASE_JSON_BASE64` | deploy | Contents of the `firebase.json` credentials file encoded in base64 |
| `SNYK_TOKEN` | ci | Snyk API token for the dependency scan |
| `GITHUB_TOKEN` | ci, codeql | Provided automatically by GitHub, do not create it |

The deploy workflow also writes `.env.prod` on the VPS from repository secrets. These are the runtime variables listed in [Environment variables](#environment-variables) and must exist as secrets too:

| Secret | Description |
| --- | --- |
| `SERVER_URL` | Public server URL |
| `SQLITE_PASSWORD` | SQLCipher database password |
| `VALKEY_PASSWORD` | Valkey password |
| `ADMIN_MAIL` | Administrator email address |
| `SMTP_HOST`, `SMTP_USER`, `SMTP_PASSWORD` | Outgoing mail credentials |
| `ACCESS_TOKEN_SECRET`, `REFRESH_TOKEN_SECRET`, `WS_ACCESS_TOKEN_SECRET` | JWT signing secrets |

Variables with defaults (`ADMIN_INVITE_TTL`, `INVITE_TTL`, token lifespans, `MESSAGE_TTL`, `MEDIA_TTL`, `MEDIA_MAX_UPLOAD_SIZE`, `MEDIA_DIR`, `SMTP_PORT`, `VALKEY_PORT`, `GIN_MODE`) are not passed by the workflow; set them in `docker-compose.prod.yml` if you need to override a default.


To get `FIREBASE_JSON_BASE64`, run:

- **Linux/macOS**: `base64 -w 0 < firebase.json` (copy the output)
- **Windows**: `[Convert]::ToBase64String([IO.File]::ReadAllBytes("firebase.json"))` (in PowerShell)

### Setting up an SSH key

1. Generate a new SSH key pair without a password:

```bash
   ssh-keygen -t ed25519 -C “github-actions” -f ~/.ssh/github_actions_key
```

2. Add the public key to the server in the `~/.ssh/authorized_keys` file:

```bash
   ssh-copy-id -i ~/.ssh/github_actions_key.pub -p your_ssh_port user@your_server
```

3. Copy the contents of the private key (`~/.ssh/github_actions_key`) and save it as a GitHub Secret named `VPS_SSH_KEY`.

### Reverse proxy

For deploying behind a reverse proxy see [nginx configuration](whisper.conf).
