# Boardit

A full-stack blog/note editor with Markdown support, folder organization, and JWT authentication.

- **Home** (`/`): Public post list — all published posts with `visibility: public`.
- **Post detail** (`/post/:username/:slug`): Read-only view; unlisted posts are accessible by direct link.
- **Editor** (`/editor`): Create and edit notes (login required); set visibility and publish to show on the home page.

## Tech Stack

| Layer | Technologies |
|-------|-------------|
| Backend | Go 1.24 · Gin · GORM · PostgreSQL |
| Frontend | React 19 · TypeScript · Vite · Tailwind CSS v4 |
| Editor | MDXEditor · CodeMirror |
| Auth | JWT (access token + refresh token) |
| API codegen | Orval (OpenAPI → React Query hooks) |
| Mock | MSW (Mock Service Worker) |

## Project Structure

```
Boardit/
├── backend/                  # Go backend (Gin + GORM)
│   ├── main.go
│   ├── .env_sample           # Environment variable template
│   ├── docs/api/             # OpenAPI spec (api-contract-v1.yaml)
│   └── internal/
│       ├── handler/          # HTTP handlers
│       ├── config/           # Validated runtime configuration
│       ├── middleware/       # JWT middleware
│       ├── model/            # Persistence-only GORM models
│       ├── noteapp/          # Note/folder use cases, DTOs, repository boundary
│       └── router/           # Route definitions
└── frontend/                 # React frontend (Vite)
    ├── src/
    │   ├── api/gen/          # Auto-generated API client (do not edit)
    │   ├── contexts/         # Auth context (JWT storage)
    │   ├── mocks/            # MSW mock handlers
    │   └── pages/            # PostList · PostDetail · Login · Register · Editor
    └── orval.config.cjs      # API codegen config
```

## Prerequisites

- Go 1.24+
- Node.js 22 (see `frontend/.nvmrc`)
- PostgreSQL 15+

## Local Development

### 1. Clone the repo

```bash
git clone https://github.com/kZime/Boardit.git
cd Boardit
```

### 2. Start PostgreSQL (Docker only)

For daily development, use Docker to run only the database — no need to build backend or frontend images:

```bash
docker compose up -d postgres
```

### 3. Set up the backend

```bash
cd backend
cp .env_sample .env
```

Edit `.env`:

```env
DATABASE_DSN="host=localhost user=<user> password=<password> dbname=boardit port=5432 sslmode=disable TimeZone=UTC"
JWT_SECRET="<your-random-secret>"
```

First run auto-migrates the database tables, then starts the server:

```bash
go run main.go             # starts on :8080
```

### 4. Set up the frontend

```bash
cd frontend
npm ci
npm run dev                # http://localhost:5173 (hot-reload)
```

> **Mock mode** (no backend needed): `npm run dev:mock`
>
> When switching between modes, clear localStorage:
> ```js
> localStorage.removeItem('accessToken');
> localStorage.removeItem('refreshToken');
> location.reload();
> ```

### When to use Docker Compose

| Scenario | Command |
|----------|---------|
| Daily dev | `docker compose up -d postgres` + `go run` + `npm run dev` |
| Verify full build before committing | `docker compose up -d --build` |
| Deploy to server | `docker compose up -d --build` |

## Deployment with Docker Compose

Docker Compose is well-suited for single-machine deployment (VPS, self-hosted). One command starts the backend, frontend, and PostgreSQL without installing Go, Node, or PostgreSQL on the host. The stack is set up for **linux/amd64** (typical Ubuntu x86_64 servers).

### Ubuntu server (recommended: build on the server)

On a fresh Ubuntu 22.04/24.04 (x86_64), install Docker and Compose, then run the app:

```bash
# Install Docker Engine and Compose (Ubuntu)
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker $USER
# Log out and back in (or newgrp docker), then:
```

Then in the project directory (clone first if needed):

```bash
git clone https://github.com/kZime/Boardit.git
cd Boardit
cp .env.docker.example .env
# Edit .env: set POSTGRES_PASSWORD and JWT_SECRET (min 32 chars)
docker compose up -d --build
```

The app will be available on port 80. To allow it through the firewall: `sudo ufw allow 80/tcp && sudo ufw enable` (if using UFW).

**ARM servers (e.g. AWS Graviton):** In `docker-compose.yml`, change `platform: linux/amd64` to `platform: linux/arm64` for all three services (or remove the `platform` lines to use the host architecture).

### Steps (generic)

1. **Clone and enter the repo**
   ```bash
   git clone https://github.com/kZime/Boardit.git
   cd Boardit
   ```

2. **Configure environment variables**
   ```bash
   cp .env.docker.example .env
   ```
   Edit `.env` and set at least:
   - `POSTGRES_PASSWORD` — database password
   - `JWT_SECRET` — random string, at least 32 characters

3. **Build and start**
   ```bash
   docker compose up -d --build
   ```
   The first run builds images and starts all three services; later you can use `docker compose up -d` only.

4. **Access**
   - Frontend (with API reverse proxy): <http://localhost:80> (or the port set by `FRONTEND_PORT`)

### Notes

- **postgres**: Data is stored in Docker volume `postgres_data`; it persists across restarts.
- **backend**: Go service; it waits for PostgreSQL to be healthy before starting. GORM runs migrations on first run.
- **frontend**: Multi-stage build (Node build + Nginx serving static files) and reverse-proxies `/api` to the backend, so the app is same-origin and CORS does not need extra configuration.

### Updating after you push

On the server, pull the latest code and rebuild/restart the stack:

```bash
cd /path/to/Boardit   # or wherever you cloned the repo
git pull
docker compose up -d --build
```

- `git pull` gets the latest commit.
- `docker compose up -d --build` rebuilds images that changed (backend/frontend from source), leaves `postgres` and its volume as-is, and restarts only the services that need it.

Optional: clean old images and build cache to free disk space:

```bash
docker image prune -f
# or more aggressively: docker system prune -f
```

### Other deployment options

| Scenario | Option |
|----------|--------|
| Single machine / small team | **Docker Compose** (recommended, as above) |
| No server management | PaaS: Backend + DB on [Railway](https://railway.app) / [Render](https://render.com), frontend as static hosting on Vercel/Netlify |
| Multi-instance / high availability | Kubernetes or cloud container services (e.g. ECS, Cloud Run) |

## Running Tests

**Backend** (default: no setup required):

```bash
cd backend
go test ./... -v
```

Tests use SQLite in-memory when `DATABASE_DSN=:memory:` in `backend/.env.test` (the default in the sample). No PostgreSQL or test user is needed. To test against a real PostgreSQL instance (e.g. to catch dialect-specific issues), copy `backend/.env_test_sample` to `backend/.env.test`, set `DATABASE_DSN` to a test database, and run the same command. CI runs tests against PostgreSQL.

## Updating the API Contract

After editing `backend/docs/api/api-contract-v1.yaml`, regenerate the frontend client:

```bash
cd frontend
npm run orval
```

## Available Scripts

### Backend

| Command | Description |
|---------|-------------|
| `go run main.go` | Start dev server on `:8080` |
| `go test ./... -v` | Run all tests |
| `go test -v -cover ./...` | Run tests with coverage |

### Frontend

| Command | Description |
|---------|-------------|
| `npm run dev` | Dev server with real API |
| `npm run dev:mock` | Dev server with MSW mock API |
| `npm run build` | Production build |
| `npm run orval` | Regenerate API client from OpenAPI spec |
| `npm run lint` | Run ESLint |

## API Overview

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | — | Register |
| POST | `/api/auth/login` | — | Login → tokens |
| POST | `/api/auth/refresh` | — | Refresh tokens |
| GET | `/api/user` | JWT | Current user |
| GET | `/api/v1/public/notes` | — | List public notes (no auth) |
| GET | `/api/v1/public/notes/:username/:slug` | — | Get public note by username and slug (no auth) |
| GET | `/api/v1/notes` | JWT | List notes |
| POST | `/api/v1/notes` | JWT | Create note |
| GET | `/api/v1/notes/:id` | JWT | Get note |
| PATCH | `/api/v1/notes/:id` | JWT | Update note |
| DELETE | `/api/v1/notes/:id` | JWT | Delete note |
| POST | `/api/v1/folders` | JWT | Create folder |
| PATCH | `/api/v1/folders/:id` | JWT | Update folder |
| DELETE | `/api/v1/folders/:id` | JWT | Delete folder |
| POST | `/api/v1/tree/reorder` | JWT | Reorder tree |
