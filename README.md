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

The application uses GitHub Actions to automatically build and deploy when pushing to the main branch. For this process to work correctly, the following secret keys must be set in the repository settings (Settings > Secrets and variables > Actions):

| Secret | Description |
| --- | --- |
| VPS_HOST | IP address or domain name of the VPS server |
| VPS_USERNAME | Username for SSH access |
| VPS_SSH_KEY | Private SSH key for server access |
| VPS_PORT | SSH port (usually 22) |
| PROJECT_PATH | Absolute path to the project on the VPS server |
| FIREBASE_JSON_BASE64 | Contents of the firebase.json file encoded in base64 |

And other variables listed in [Environment variables](#environment-variables).

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
