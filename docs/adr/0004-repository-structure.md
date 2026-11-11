# 0004 - Four repositories, no staging

Status: accepted (2026-11-11)

## Context

The app, server and two clients have different release cycles and toolchains.
There is one developer, a public free-tier GitHub setup and no team that would
use a staging environment or review approvals.

## Decision

Keep four repositories. There is no platform repository and no staging
deployment. Contract changes are made in the server, the OpenAPI client is
regenerated and the app bumps its dependency in one sequence.

## Consequences

- CI configuration is duplicated on purpose; a shared platform repo would be
  tighter coupling than the duplication costs.
- Production is the first environment that sees a release. Tags, health checks
  and rollback replace staging.
- Cross-repository decisions live in the Notion Engineering Standard.
