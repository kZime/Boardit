# Boardit backend

The Go API is a modular monolith for authentication, notes, folders, publishing, revisions, and durable change events.

## Run

```bash
cp .env_sample .env
# Set DATABASE_DSN and JWT_SECRET.
go run .
```

Startup validates configuration, opens PostgreSQL or SQLite, applies pending embedded SQL migrations, and then starts the HTTP server on the configured address.

## Dependency direction

```text
router -> HTTP handler -> noteapp.Service -> Repository -> GORM/database
```

- `internal/noteapp` owns permissions, concurrency, transaction, revision, and publishing rules.
- `internal/handler` owns HTTP parsing and response mapping.
- `internal/model` is persistence-only.
- Future AI writes must call the note use-case and must not write GORM models directly.

See the root [architecture](../docs/architecture.md) and [agent guide](../AGENTS.md) for complete invariants.

## Test

```bash
test -z "$(gofmt -l .)"
go vet ./...
go test -race -p 1 ./...
```

Tests default to SQLite with `DATABASE_DSN=:memory:`. CI runs against PostgreSQL 15. Test databases are truncated; never use a database containing valuable data.

## Database operations

```bash
# Apply pending migrations explicitly
go run ./cmd/migrate -direction up

# Roll back exactly one migration after reviewing its down SQL
go run ./cmd/migrate -direction down
```

Read [database-migrations.md](docs/database-migrations.md) before schema changes or rollback. Applied migrations are immutable and every new version requires PostgreSQL and SQLite up/down files.

## API contract

`docs/api/api-contract-v1.yaml` is authoritative. After changing it, regenerate the frontend client from `frontend/` with `npm run orval`. Do not hand-edit generated TypeScript.

Useful references:

- [AI and asynchronous data boundaries](docs/ai-data-boundaries.md)
- [Testing strategy](../docs/testing-strategy.md)
- [Architecture decisions](../docs/adr/README.md)
