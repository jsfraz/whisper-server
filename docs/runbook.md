# Runbook

Operational procedures for the production VPS. The deployment is a Docker
Compose project in `$PROJECT_PATH/whisper-server`; the server listens on
`127.0.0.1:8081` and stores state in the `whisper_data` and `valkey_data`
volumes.

Replace `$APP_DIR` with the absolute path to the compose project.

## Health check

```bash
curl -fsS http://127.0.0.1:8081/health   # {"status":"ok"}
docker compose -f "$APP_DIR/docker-compose.prod.yml" --env-file "$APP_DIR/.env.prod" ps
docker inspect -f '{{.State.Health.Status}}' whisper-server
```

A deploy counts as successful only when the container reports `healthy`.

## Restart

```bash
cd "$APP_DIR"
docker compose -f docker-compose.prod.yml --env-file .env.prod restart whisper-server
```

A restart keeps SQLite, Valkey and media on disk. If the process keeps
crash-looping, check `docker compose logs --tail=200 whisper-server`; most
startup failures are a missing `data/firebase.json` or a bad `.env.prod`.

## Rollback

Automatic: the deploy workflow writes the active tag to `.whisper-image-tag`.
If the new container does not become healthy, it redeploys the previous tag and
the workflow still fails (so an issue is opened and the failure is visible).

Manual, to a known good tag:

1. GitHub -> Actions -> **Deploy to VPS** -> Run workflow, enter the old tag
   (for example `v1.2.2`).
2. Or on the VPS:

   ```bash
   cd "$APP_DIR"
   WHISPER_IMAGE_TAG=v1.2.2 docker compose -f docker-compose.prod.yml \
     --env-file .env.prod up -d
   ```

Then verify `/health`. Do not roll back the database schema: goose migrations
are forward-only in practice. A rollback to an older image must still work with
the current schema; if it does not, restore the database backup instead.

## Logs

```bash
docker compose -f "$APP_DIR/docker-compose.prod.yml" --env-file "$APP_DIR/.env.prod" \
  logs -f --tail=200 whisper-server
```

Logs are JSON in `GIN_MODE=release`. Every request line has `request_id`
(the `X-Request-ID` header is reused when the client sends it), `method`,
`path`, `status`, `duration_ms` and `client_ip`.

## Secret rotation

Rotate one secret at a time, update the GitHub secret, redeploy, and verify
`/health`. Effects below are what users will notice.

| Secret | Effect |
| --- | --- |
| `ACCESS_TOKEN_SECRET` | All access tokens become invalid. Clients refresh or re-authenticate; no data loss. |
| `REFRESH_TOKEN_SECRET` | All refresh tokens become invalid. Clients have to authenticate again with their RSA key; message history is unaffected. |
| `WS_ACCESS_TOKEN_SECRET` | Open WS connections are rejected on the next handshake; clients reconnect and fetch a fresh one-time token. |
| `VALKEY_PASSWORD` | Update `.env.prod` and restart both services. Data survives; no re-encryption. |
| `SMTP_PASSWORD` | Invite e-mails fail until fixed; no impact on existing users. |
| `FIREBASE_JSON_BASE64` | Push notifications fail until fixed; messages, media and WS delivery keep working. |
| `SQLITE_PASSWORD` | **Do not rotate by editing `.env.prod`.** The database is encrypted with this key. Rekeying requires an offline backup and `PRAGMA rekey`, then a new secret. |

## Database backup

Stop the server before copying the SQLite file so the backup is consistent:

```bash
cd "$APP_DIR"
docker compose -f docker-compose.prod.yml --env-file .env.prod stop whisper-server
docker run --rm -v whisper_whisper_data:/data -v "$PWD":/backup alpine \
  tar czf /backup/whisper-data.tgz -C /data .
docker compose -f docker-compose.prod.yml --env-file .env.prod start whisper-server
```

Confirm the volume name with `docker volume ls`; Compose prefixes project
volumes with the project name from `docker-compose.prod.yml` (`whisper`).

## Deploy failed

1. Open the `[deploy] Production deploy failed` issue; it links the failed run.
2. Check what the workflow did: image pull, health check, or rollback.
3. If the rollback also failed, deploy the last known good tag manually.
4. Do not re-run the same tag until the cause is understood; the deploy is
   idempotent but the health check is what fails.

## First deploy of a new image

The `ghcr.io/jsfraz/whisper-server` package must be public, otherwise the VPS
cannot pull it without a registry login. After the first Release, verify the
package visibility in GitHub -> Packages. If it has to stay private, add a
registry login to the deploy script and a `GHCR_TOKEN` secret first.
