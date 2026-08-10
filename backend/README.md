# Boardit Backend

## APIs

| Method | Endpoint             | Description                                                                   |
| ------ | -------------------- | ----------------------------------------------------------------------------- |
| POST   | `/api/auth/register` | `{ username, email, password }`, return new user info (without password) |
| POST   | `/api/auth/login`    | `{ email, password }`, return `{ access_token, refresh_token, expires_in }`    |
| POST   | `/api/auth/refresh`  | `{ refresh_token }`, return new `{ access_token, refresh_token, expires_in }` |
| POST   | `/api/auth/logout`   | revoke the supplied refresh session |
| GET    | `/api/user`          | get current user info (need to carry `Authorization: Bearer <access_token>` in Header) |
| GET    | `/api/v1/notes/:id/revisions` | list immutable revisions owned by the current user |

## Architecture

The note domain is a modular monolith with one-way dependencies:

```text
Gin router -> HTTP adapters -> noteapp.Service -> Repository -> GORM/PostgreSQL
```

- `internal/handler/*_http.go` parses HTTP input and maps application errors to the v1 contract.
- `internal/noteapp/service.go` owns note, folder, publishing, authorization and transaction rules.
- `internal/noteapp/repository.go` is the persistence boundary; GORM models never appear in public use-case inputs or outputs.
- `internal/config` validates runtime settings before database or router initialization.
- Future AI commands must call `noteapp.Service`; they must not write GORM models directly.

## Database evolution

- Application startup runs embedded, versioned SQL migrations instead of GORM `AutoMigrate`.
- Note create/update writes the current note, immutable revision and outbox event atomically.
- Migration commands and rollback safety are documented in [`docs/database-migrations.md`](docs/database-migrations.md).
- Provider-neutral jobs, AI runs and candidate boundaries are documented in [`docs/ai-data-boundaries.md`](docs/ai-data-boundaries.md).

## Testing

### Test Structure

The backend uses a comprehensive testing strategy with test suites for different components:

```
backend/
├── internal/
│   ├── handler/
│   │   ├── handler_test.go      # Auth handler tests (test suite)
│   │   ├── note_test.go         # HTTP contract/characterization tests
│   │   └── ...
│   ├── noteapp/
│   │   ├── service_test.go      # Use-case tests without Gin
│   │   └── ...
│   ├── middleware/
│   │   ├── jwt_test.go          # JWT middleware tests (test suite)
│   │   └── ...
│   ├── router/
│   │   ├── router_test.go       # Router configuration tests (test suite)
│   │   └── ...
│   ├── database/
│   │   ├── database_test.go     # Database connection tests
│   │   └── ...
│   └── testutils/
│       └── testutils.go         # Test utilities and helpers
```

### Test Types

#### 1. Handler Tests (`handler_test.go`)
- **Purpose**: Test business logic and API endpoints
- **Structure**: Uses `AuthTestSuite` with shared state
- **Features**:
  - Tests complete authentication flow (register → login → refresh → get user)
  - Shared test data between tests
  - Database cleanup between tests
  - Real HTTP requests simulation

#### 2. Middleware Tests (`jwt_test.go`)
- **Purpose**: Test JWT authentication middleware
- **Structure**: Uses `JWTMiddlewareTestSuite`
- **Test Cases**:
  - Valid JWT tokens
  - Invalid/expired tokens
  - Missing Authorization headers
  - Invalid header formats
  - Wrong signing methods
  - Missing/invalid claims

#### 3. Router Tests (`router_test.go`)
- **Purpose**: Test route configuration and middleware setup
- **Structure**: Uses `RouterTestSuite`
- **Test Cases**:
  - Route existence verification
  - CORS configuration
  - Authentication middleware application
  - Error handling (404, 405)
  - Content-Type handling

#### 4. Database Tests (`database_test.go`)
- **Purpose**: Test database connection and migration
- **Features**: Database initialization and cleanup

### Running Tests

#### Default (no setup)
With `backend/.env.test` containing `DATABASE_DSN=:memory:` (or no `.env.test` and env not set), tests use **SQLite in-memory** — no PostgreSQL or test user required. Just run:

```bash
cd backend
go test -v ./...
```

#### Optional: test against PostgreSQL
To run tests against a real PostgreSQL instance (e.g. to catch dialect-specific issues): copy `.env_test_sample` to `.env.test`, set `DATABASE_DSN` to your test database, and run the same command. CI uses PostgreSQL.

#### Test Commands

```bash
# Run all tests
go test -v ./...

# Run tests for specific package
go test -v ./internal/handler
go test -v ./internal/noteapp
go test -v ./internal/middleware
go test -v ./internal/router
go test -v ./internal/database

# Run specific test suite
go test -v ./internal/handler -run TestAuthSuite
go test -v ./internal/middleware -run TestJWTMiddlewareSuite
go test -v ./internal/router -run TestRouterSuite

# Run specific test
go test -v ./internal/handler -run TestAuthSuite/TestLoginSuccess
go test -v ./internal/middleware -run TestJWTMiddlewareSuite/TestValidToken

# Run tests with coverage
go test -v -cover ./...
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Data Management

- **Setup**: Each test suite initializes its own test data
- **Cleanup**: Database is cleaned between tests to ensure isolation
- **Isolation**: Tests are independent and can run in any order

### Test Coverage

Current test coverage includes:
- ✅ Authentication flow (register, login, refresh, get user)
- ✅ JWT middleware validation
- ✅ Route configuration and CORS
- ✅ Database connection and migration
- ✅ Error handling and edge cases

### Adding New Tests

When adding new functionality, follow these patterns:

#### 1. Handler Tests
```go
func (suite *YourTestSuite) TestNewFeature() {
    // Setup test data
    suite.setupTestData()
    
    // Make request
    req, err := http.NewRequest("POST", "/api/endpoint", bytes.NewBuffer(body))
    suite.NoError(err)
    
    // Assert response
    suite.Equal(http.StatusOK, w.Code)
}
```

#### 2. Middleware Tests
```go
func (suite *MiddlewareTestSuite) TestNewMiddleware() {
    // Test middleware behavior
    req, err := http.NewRequest("GET", "/test", nil)
    suite.NoError(err)
    
    // Assert middleware response
    suite.Equal(expectedStatus, w.Code)
}
```

#### 3. Router Tests
```go
func (suite *RouterTestSuite) TestNewRoute() {
    // Test route existence and behavior
    req, err := http.NewRequest("GET", "/api/new-route", nil)
    suite.NoError(err)
    
    // Assert route response
    suite.NotEqual(http.StatusNotFound, w.Code)
}
```

## API Contract

The API contract is defined in `docs/api/api-contract-v1.md`.

Every time the API contract is updated, please update the `docs/api/api-contract-v1.yaml` file.

## Development Log

**2025-08-20:** 
- Added comprehensive test suite structure
- Implemented JWT middleware tests
- Added router configuration tests
- Improved test isolation and cleanup

**2025-08-04:** 
- Add github.com/stretchr/testify/assert for testing
- Initial test setup for auth handlers
