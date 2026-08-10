# Boardit current architecture

**Status**: current after R0-R4
**Style**: modular monolith
**Contract**: `backend/docs/api/api-contract-v1.yaml`

This document describes the system as it exists now. The readiness assessment is retained as historical analysis and should not be used as the current component map.

## System context

Boardit has three runtime components:

- React web application: public reading, authentication, note tree, editor, metadata, and publishing UI.
- Go API: authentication plus note, folder, publishing, revision, and tree use-cases.
- PostgreSQL: current state, immutable revisions, refresh sessions, and asynchronous/AI foundation tables.

MSW provides a frontend-only mock API for deterministic development and Playwright smoke tests. SQLite supports fast backend tests; PostgreSQL 15 remains the production dialect and CI integration target.

## Backend modules

```text
main/config
    -> router and middleware
        -> HTTP adapters
            -> noteapp.Service
                -> Repository interface
                    -> GORM/PostgreSQL
```

| Module | Responsibility |
|---|---|
| `internal/config` | Validate runtime configuration before startup |
| `internal/router` | Assemble routes, middleware, CORS, and trusted proxies |
| `internal/handler` | Parse HTTP requests and map use-case errors to the API contract |
| `internal/noteapp` | Authorization, validation, publishing, concurrency, transactions, and DTO mapping |
| `internal/model` | Persistence-only GORM models |
| `internal/database` | Connection, embedded versioned migrations, rollback, and test cleanup |

Authentication handlers still use the shared database package directly; extracting an auth application module is tracked as debt.

## Frontend modules

| Module | Responsibility |
|---|---|
| `src/api/gen` | Orval-generated API types and React Query hooks |
| `src/api/orval-axios.ts` | Generated-client requester adapter |
| `src/api/axios.ts` | Shared HTTP client and single-flight token refresh |
| `src/auth` | Token storage and shared authentication form rules |
| `src/features/editor` | Note pagination/tree, metadata dialogs, save coordination, and editor types |
| `src/pages` | Route-level composition only |
| `src/mocks` | Contract-shaped MSW development API |

The Editor route is lazy-loaded so MDXEditor and CodeMirror do not enter the public-page initial bundle.

## Accepted note save

```text
Editor state + base version
    -> generated updateNote request
    -> HTTP adapter
    -> noteapp optimistic version check
    -> one transaction:
         conditional note update
         immutable note_revision insert
         outbox_event insert
    -> updated note + next version
```

If the base version is stale, the transaction returns `VERSION_CONFLICT` with the current server snapshot. A revision failure or outbox failure rolls back the note update.

Tree moves also advance the note version and record a revision/event. Deletion leaves a `note.deleted` outbox event for future index cleanup.

## Authentication lifecycle

- Login creates a short-lived access token and a persisted refresh session.
- Refresh tokens have an explicit token type and random `jti`.
- Refresh atomically revokes the old session and creates a replacement in the same session family. Reuse of a rotated token revokes every active session in that family and returns 401.
- Logout revokes the supplied refresh session and immediately clears local browser tokens.
- Access-token refresh requests share one frontend promise so concurrent 401 responses settle together.

Refresh tokens are currently stored in localStorage. Migration to an HttpOnly cookie is an open security decision.

## Asynchronous and AI foundation

- `outbox_events` stores transactionally durable pending records from note transactions; it does not provide delivery until a worker exists.
- `background_jobs` defines provider-neutral claiming, retry, and deduplication state.
- `ai_runs` records model/prompt/status/cost metadata boundaries.
- `ai_candidates` stores proposed Markdown against a base note version.

No worker, provider SDK, prompt runtime, vector index, or automatic candidate acceptance is implemented yet. See `backend/docs/ai-data-boundaries.md`.

## Deployment model

Docker Compose remains the intended single-machine deployment model. Microservices, a separate message cluster, Kubernetes, and multi-model routing are intentionally deferred until measured scale or reliability requirements justify them.
