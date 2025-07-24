# Blogedit Backend

## APIs

| Method | Endpoint        | Description | Request Example    | Todo |
| ------ | --------------- | ----------- | ------------------ | ---- |
| POST   | `/api/register` | register    | `{ "username": "tom", "email": "t@a.com", "password": "123456" }` |  ❌ |
| POST   | `/api/login`    | login       | `{ "email": "t@a.com", "password": "123456" }` | ❌ |
| POST   | `/api/refresh`  | refresh     | `{ "refresh_token": "..." }` | ❌ |
| GET    | `/api/user`     | get user    | `Authorization: Bearer <access_token>` | ❌ |
