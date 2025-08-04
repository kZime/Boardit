# Blogedit Backend

## APIs

| Method | Endpoint             | Description                                                                   |

| ------ | -------------------- | ----------------------------------------------------------------------------- |
| POST   | `/api/auth/register` | `{ username, email, password }`, return new user info (without password) |
| POST   | `/api/auth/login`    | `{ email, password }`, return `{ access_token, refresh_token }`    |
| POST   | `/api/auth/refresh`  | `{ refresh_token }`, return new `{ access_token, refresh_token }` |
| GET    | `/api/user`          | get current user info (need to carry `Authorization: Bearer <access_token>` in Header) |

## Unit Tests
To run the unit tests, please create a database named testdb with username `test` and password `test`
and the sample `.env_test_sample` file is provided in the backend directory.

For Testing the auth handler, set up the router with Gin's test mode and then simulate requests to the endpoints. The payloads and expected reponses should be defined based on requirements.

## Develop Log
**2025-08-04:** Add github.com/stretchr/testify/assert for testing.
