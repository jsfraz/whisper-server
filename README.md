# whisper-server

Secure private self-hosted messaging server using end-to-end encryption.

For Whisper app, see [whisper](https://github.com/jsfraz/whisper) repository.

Also see [Wiki](https://github.com/jsfraz/whisper-server/wiki)!

## Deployment

```bash
sudo docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
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
