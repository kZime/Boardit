# Architecture decision records

ADRs capture durable decisions and their trade-offs. They explain why the architecture looks this way; `docs/architecture.md` describes the resulting current state.

| ADR | Decision | Status |
|---|---|---|
| [0001](0001-modular-monolith.md) | Keep a modular monolith | Accepted |
| [0002](0002-openapi-generated-client.md) | OpenAPI-first generated frontend client | Accepted |
| [0003](0003-note-version-revision-outbox.md) | Integer note versions plus revision/outbox transaction | Accepted |
| [0004](0004-versioned-sql-migrations.md) | Embedded versioned SQL migrations | Accepted |
| [0005](0005-postgres-outbox-jobs.md) | PostgreSQL outbox/jobs before an external broker | Accepted |
| [0006](0006-refresh-session-rotation.md) | Server-side rotating refresh sessions | Accepted |

Create a new numbered record when changing a durable decision. Do not rewrite an accepted ADR to hide a superseded choice; add a new ADR and mark the old one superseded.
