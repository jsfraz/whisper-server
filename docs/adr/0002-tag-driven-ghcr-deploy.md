# 0002 - Tag-driven release and pull-based deploy

Status: accepted (2026-11-11)

## Context

Deploys used to run on every push to `main` and rebuilt the image on the VPS.
There was no immutable artifact, no rollback target, and a broken build reached
production before anyone could inspect it.

## Decision

- A `v*` tag builds `ghcr.io/jsfraz/whisper-server:<tag>` (linux/amd64).
- The deploy workflow runs after a successful Release or manually with a tag.
- The VPS pulls the image, runs `docker compose up -d`, waits for the container
  health check and rolls back to the previous tag on failure.
- The image package is public so the VPS needs no registry credentials.

## Consequences

- The deployed artifact is the one that was tagged and scanned.
- Rollback is a tag, not a git revert and rebuild.
- First deploy requires the ghcr package to be made public manually.
