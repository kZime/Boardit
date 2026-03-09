# Blogedit

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
Blogedit/
├── backend/                  # Go backend (Gin + GORM)
│   ├── main.go
│   ├── .env_sample           # Environment variable template
│   ├── docs/api/             # OpenAPI spec (api-contract-v1.yaml)
│   └── internal/
│       ├── handler/          # HTTP handlers
│       ├── middleware/        # JWT middleware
│       ├── model/            # GORM models
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
- Node.js 18+
- PostgreSQL 14+

## Local Development

### 1. Clone the repo

```bash
git clone https://github.com/kZime/Blogedit.git
cd Blogedit
```

### 2. Set up the backend

```bash
cd backend

# Copy and fill in environment variables
cp .env_sample .env
```

Edit `.env`:

```env
DATABASE_DSN="host=localhost user=<user> password=<password> dbname=blogedit port=5432 sslmode=disable TimeZone=UTC"
JWT_SECRET="<your-random-secret>"
```

Create the database, then start the server (GORM auto-migrates tables on first run):

```bash
createdb blogedit          # or use psql
go run main.go             # starts on :8080
```

### 3. Set up the frontend

```bash
cd frontend
npm install
```

**Real API mode** (requires backend running):

```bash
npm run dev                # http://localhost:5173
```

**Mock mode** (no backend needed):

```bash
npm run dev:mock
```

> When switching between modes, clear localStorage in the browser console:
> ```js
> localStorage.removeItem('accessToken');
> localStorage.removeItem('refreshToken');
> location.reload();
> ```

## Running Tests

```bash
cd backend

# Copy test environment variables
cp .env_test_sample .env.test
# Edit .env.test with a separate test database (dbname=testdb)

go test ./... -v
```

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
