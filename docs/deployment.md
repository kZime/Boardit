# Deployment runbook

Boardit currently targets one host with Docker Compose. This is appropriate for the portfolio/demo workload and preserves a simple PostgreSQL transaction boundary.

## Required configuration

Copy the deployment template and set secrets locally on the host:

```bash
cp .env.docker.example .env
```

Required values:

- `POSTGRES_PASSWORD`: database password.
- `JWT_SECRET`: random value of at least 32 characters.

Optional values include the database/user names, frontend port, CORS origins, trusted proxies, and frontend API base URL. Do not commit the populated `.env` file.

## Start or update

```bash
docker compose up -d --build
docker compose ps
```

The Compose health check starts the backend after PostgreSQL is ready. API startup applies pending forward migrations before listening. The frontend serves static files through Nginx and proxies `/api` to the backend.

For an update:

1. Review release notes and pending SQL migrations.
2. Back up PostgreSQL and verify that the backup is readable.
3. Pull the intended commit or release tag.
4. Run `docker compose up -d --build`.
5. Check service health, login, note editing, and a public permalink.

Do not use an unreviewed down migration as the first recovery action. Prefer a forward fix; follow `backend/docs/database-migrations.md` when rollback is necessary.

## Persistence and recovery

PostgreSQL data lives in the `postgres_data` volume. Rebuilding application images does not delete the volume. Removing the volume is destructive and is not part of normal deployment or cleanup.

Backups must cover the database before schema changes. Application images and source can be rebuilt from Git; the database cannot.

## Architecture and platform

- The current Compose file targets `linux/amd64`.
- On an ARM host, review or remove the explicit `platform` entries before deployment.
- Expose only the frontend port unless direct API access is intentionally required.
- Terminate TLS at a trusted reverse proxy or hosting layer for any public deployment.
- Set CORS and trusted-proxy values explicitly for the deployed origin.

## Scaling boundary

Do not introduce Kubernetes or a separate message broker solely for presentation. Revisit deployment architecture when measured traffic, availability, worker throughput, or team ownership exceeds a single-host design.
