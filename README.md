# Boardit

Boardit is a full-stack Markdown note and publishing application being evolved into a trustworthy AI writing and private-knowledge portfolio project.

## What works today

- Private notes, folders, search, editing, version conflicts, and immutable revisions.
- Public and unlisted publishing with author permalinks.
- Rotating JWT refresh sessions with replay rejection and logout revocation.
- OpenAPI-generated React Query client, deterministic CI, unit/integration tests, and Playwright smoke tests.
- Versioned PostgreSQL/SQLite migrations plus outbox, job, AI-run, and candidate data boundaries.

The AI runtime, workers, retrieval, eval dashboard, and automatic indexing are foundations only—not implemented product features yet.

## Stack

| Layer | Technologies |
|---|---|
| Frontend | React 19, TypeScript, Vite 8, Tailwind CSS, React Query |
| Editor | MDXEditor, CodeMirror, Markdown |
| Backend | Go 1.24, Gin, GORM |
| Data | PostgreSQL 15; SQLite for fast local tests |
| Contract | OpenAPI, Orval-generated hooks and types |
| Testing | Go test/race, Vitest, MSW, Playwright |

## Quick start

Prerequisites: Go 1.24+, Node.js 22, npm 10, and PostgreSQL 15+ or Docker.

```bash
# Database
docker compose up -d postgres

# Backend
cd backend
cp .env_sample .env
# Set DATABASE_DSN and a JWT_SECRET of at least 32 characters.
go run .

# Frontend, in a second shell
cd frontend
npm ci
npm run dev
```

Open <http://localhost:5173>. To run the frontend without the backend, use `npm run dev:mock`.

API startup applies pending versioned SQL migrations. Migration commands and rollback safety are documented separately; the application does not use AutoMigrate in its startup path.

## Quality gates

```bash
# Backend
cd backend
test -z "$(gofmt -l .)"
go vet ./...
go test -race -p 1 ./...

# Frontend
cd frontend
npm ci
npm run lint
npm test
npm run build
npm run test:e2e
```

See [testing strategy](docs/testing-strategy.md) for change-specific requirements and generated-client checks.

## Documentation

| Need | Document |
|---|---|
| Current system design | [Architecture](docs/architecture.md) |
| Durable technical decisions | [ADRs](docs/adr/README.md) |
| Agent and contributor invariants | [AGENTS.md](AGENTS.md) |
| Tests and CI | [Testing strategy](docs/testing-strategy.md) |
| Database migration and rollback | [Migration runbook](backend/docs/database-migrations.md) |
| Deployment and recovery | [Deployment runbook](docs/deployment.md) |
| AI and async data rules | [AI data boundaries](backend/docs/ai-data-boundaries.md) |
| Shipping an AI feature | [AI feature playbook](docs/ai-feature-playbook.md) |
| Known gaps and decisions | [Known debt](docs/known-debt.md) |
| Product modernization direction | [AI roadmap](docs/modernization-ai-roadmap.md) |
| Historical refactor evidence | [R0-R4 assessment](docs/refactoring-readiness-assessment.md) |

Backend- and frontend-specific setup is intentionally short and lives in [backend/README.md](backend/README.md) and [frontend/README.md](frontend/README.md).

## Deployment

Docker Compose is the supported single-machine deployment model:

```bash
cp .env.docker.example .env
# Set POSTGRES_PASSWORD and JWT_SECRET.
docker compose up -d --build
```

Data persists in the `postgres_data` volume. Back up the database and review pending migrations before production upgrades. See the [deployment runbook](docs/deployment.md) for update and recovery details. Push, deployment, domains, and production credentials are deliberately outside automated maintenance scope.
