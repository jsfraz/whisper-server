# 0002 - Tag-driven release and pull-based deploy

Status: accepted (2026-11-11)

## Context
Deploys ran on every push to `main` and rebuilt on the VPS: no immutable
artifact, no rollback target, and a broken build could reach production.

## Decision
A `v*` tag builds `ghcr.io/jsfraz/whisper-server:<tag>` (linux/amd64). Deploy
runs after a successful Release or manually with a tag, pulls the image, waits
for the health check, and rolls back to the previous tag on failure. The image
package is public, so the VPS needs no registry credentials.

## Consequences
The deployed artifact is the tagged and scanned one; rollback is a tag, not a
rebuild; the first deploy requires making the package public manually.
