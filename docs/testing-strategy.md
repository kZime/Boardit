# Testing strategy

Tests are selected by responsibility. A higher-level test does not replace a cheaper test at the layer that owns the rule.

## Test matrix

| Change | Required protection |
|---|---|
| Note/folder/auth business rule | Go use-case or handler regression test |
| Permission or user scoping | Cross-user negative test |
| Transaction or revision behavior | Success plus forced-failure rollback test |
| Database schema | Fresh migration, legacy upgrade, idempotency, and down migration |
| API request/response | OpenAPI update, generated client, and HTTP characterization test |
| Frontend pure logic | Vitest unit test |
| Query/pagination/save coordinator | Hook or feature test with mocked requester |
| Login/editor/publishing route | Playwright smoke test |
| AI feature | Unit/integration tests plus an explicit eval dataset and threshold |

## Complete local gates

### Backend

```bash
cd backend
test -z "$(gofmt -l .)"
go vet ./...
go test -race -p 1 ./...
```

`-p 1` serializes packages because PostgreSQL CI tests share one ephemeral database. The race detector still runs inside every package test binary.

### Frontend

```bash
cd frontend
npm ci
npm audit --omit=dev --audit-level=critical
npm run lint
npm test
npm run build
npm run test:e2e
```

### Generated API drift

After editing `backend/docs/api/api-contract-v1.yaml`:

```bash
cd frontend
npm run orval
git diff --exit-code -- src/api/gen
```

When the contract change is intentional, stage the expected generated files before using the final drift command.

## Database coverage

- SQLite tests provide fast migration and use-case feedback.
- CI provisions PostgreSQL 15 and runs the serialized backend suite.
- Every migration needs PostgreSQL and SQLite variants with matching `up` and `down` files.
- Never point automated tests at a database containing valuable data; suites truncate application tables.

## Test design conventions

- Assert public behavior and stable invariants, not private implementation details.
- Force one dependency failure when testing transaction atomicity.
- For concurrency, pass the previous integer version and assert a stale write conflicts.
- For private data, create two users and prove the second cannot list, search, read, revise, or mutate the first user's resource.
- Mock responses must conform to generated API types, including newly required fields.

## Definition of done

A change is complete when relevant regression tests and all complete gates pass, Orval has no unintended drift, `git diff --check` is clean, and non-blocking warnings are recorded rather than hidden.
