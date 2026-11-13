# Architecture Decision Records

Short records of decisions that shape more than one file. Each ADR answers
what was decided, why, and what it costs. Cross-cutting decisions that have no
single home repository live in the Notion decision log.

| ADR | Decision |
| --- | --- |
| [0001](0001-goose-migrations.md) | Use goose migrations instead of GORM AutoMigrate |
| [0002](0002-tag-driven-ghcr-deploy.md) | Release images by tag and deploy by pulling them |
| [0003](0003-e2e-testcontainers.md) | Go E2E against a real server, SQLite and Valkey |
| [0004](0004-repository-structure.md) | Four repositories, no staging environment |
| [0005](0005-structured-logging.md) | Structured logs with a request id |
| [0006](0006-user-ids-uuid.md) | User ids are UUIDs (clean break, clients re-register) |
