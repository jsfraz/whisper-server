# 0004 - Four repositories, no staging

Status: accepted (2026-11-11)

## Context
The app, server and two clients have different release cycles and toolchains.
There is one developer, a public free-tier setup, and no team for approvals or
a staging environment.

## Decision
Keep four repositories, no platform repository and no staging environment.
Contract changes happen in the server, the OpenAPI client is regenerated and
the app bumps its dependency in one sequence. Cross-repository decisions live
in the Notion Engineering Standard.

## Consequences
CI config is duplicated on purpose; production is the first environment to see
a release; tags, health checks and rollback replace staging.
