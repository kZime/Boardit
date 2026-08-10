# Database migrations

Boardit applies embedded, versioned SQL migrations during API startup. PostgreSQL and SQLite have separate SQL files under `internal/database/migrations/<dialect>`; `AutoMigrate` is not used in the application path.

## Invariants

- Each migration has matching `up` and `down` files.
- Applied versions, names, SHA-256 checksums and timestamps are recorded in `schema_migrations`.
- An applied migration must never be edited. Add a new version instead; startup rejects checksum drift.
- Each migration and its version record run in one database transaction.
- PostgreSQL takes an exclusive migration-table lock so concurrent application starts serialize safely.
- Production startup only moves forward. Rollback is an explicit operator action.

## Commands

Set `DATABASE_DSN`, then run from `backend/`:

```bash
# Apply every pending migration.
go run ./cmd/migrate -direction up

# Roll back exactly the latest applied version.
go run ./cmd/migrate -direction down
```

The API also applies pending migrations before opening the HTTP listener.

## Release and rollback procedure

1. Back up the database and verify restore access.
2. Run migrations against a staging copy of production data.
3. Deploy application code only after the forward migration succeeds.
4. Prefer a new forward migration for corrections.
5. Use `-direction down` only when the matching application version is ready and the down file has been reviewed.

Rolling back `000003` removes refresh-session family tracking and its replay containment. Rolling back `000002` removes refresh sessions, async/AI foundation tables and revision metadata columns. Rolling back `000001` drops all application tables and is destructive. Never run these rollbacks against production without a verified backup and an approved recovery plan.

## Tests

`internal/database/database_test.go` verifies fresh migration, idempotency, checksums and one-step rollback on SQLite. CI runs the full backend suite against PostgreSQL 15 to cover dialect-specific SQL.
