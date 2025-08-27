# Blogedit API Contract – v1

**Status:** Stable (v1)
**Scope:** Auth & User (Sprint 1) + Notes/Folder Tree/Autosave/XSS (Sprint 1.5)
**Base URL:** `/api/v1` (Except Auth + User, which still uses `/api/auth/*` and `/api/user`)

## Conventions

* **Auth**：JWT（`Authorization: Bearer <access_token>`）. Except for `/api/auth/*`, all endpoints require authentication.
* **Content-Type**：`application/json; charset=utf-8`
* **Time Format**：ISO 8601 UTC（Example: `2025-08-13T21:31:12Z`）
* **IDs**：Integer（`number`）. `sort_order` is an integer, default is `0`.
* **Visibility**：`private | unlisted | public`
* **Error Object (Uniform Format)**

  ```json
  {
    "error": "ERROR_CODE",
    "message": "human readable message",
    "details": { "field?": "desc" }
  }
  ```

  Common `ERROR_CODE`：`UNAUTHORIZED`(401), `FORBIDDEN`(403), `NOT_FOUND`(404),
  `VALIDATION_ERROR`(400), `VERSION_CONFLICT`(409), `RATE_LIMITED`(429), `INTERNAL`(500)

---

## 1) Auth & User (Legacy)

> Keep legacy paths compatible: `/api/auth/*` and `/api/user`

### 1.1 Register

* **POST** `/api/auth/register`
* **Body**

  ```json
  { "username": "alice", "email": "a@ex.com", "password": "secret123" }
  ```

* **200**

  ```json
  { "id": 1, "username": "alice", "email": "a@ex.com", "created_at": "2025-08-04T10:00:00Z" }
  ```

* **409** (Email or username already exists)

  ```json
  { "error": "VALIDATION_ERROR", "message": "email already registered" }
  ```

### 1.2 Login

* **POST** `/api/auth/login`
* **Body**

  ```json
  { "email": "a@ex.com", "password": "secret123" }
  ```

* **200**

  ```json
  { "access_token": "<jwt>", "refresh_token": "<jwt>", "expires_in": 3600 }
  ```

* **401**

  ```json
  { "error": "UNAUTHORIZED", "message": "invalid credentials" }
  ```

### 1.3 Refresh Token

* **POST** `/api/auth/refresh`
* **Body**

  ```json
  { "refresh_token": "<jwt>" }
  ```

* **200**

  ```json
  { "access_token": "<jwt>", "refresh_token": "<jwt>", "expires_in": 3600 }
  ```

* **401**

  ```json
  { "error": "UNAUTHORIZED", "message": "invalid refresh token" }
  ```

### 1.4 Get Current User

* **GET** `/api/user`
* **Headers**：`Authorization: Bearer <access_token>`
* **200**

  ```json
  { "id": 1, "username": "alice", "email": "a@ex.com", "created_at": "2025-08-04T10:00:00Z" }
  ```

---

## 2) DTO Schemas (Core Return Structures)

### 2.1 NoteDTO

```json
{
  "id": 12,
  "user_id": 1,
  "folder_id": 3,
  "title": "My First Note",
  "slug": "my-first-note",
  "content_md": "# Hello",
  "content_html": "<h1>Hello</h1>",
  "is_published": false,
  "visibility": "private",
  "sort_order": 0,
  "created_at": "2025-08-13T21:30:00Z",
  "updated_at": "2025-08-13T21:31:12Z"
}
```

### 2.2 FolderDTO

```json
{
  "id": 3,
  "user_id": 1,
  "name": "Drafts",
  "parent_id": null,
  "sort_order": 0,
  "created_at": "2025-08-13T21:00:00Z",
  "updated_at": "2025-08-13T21:00:00Z"
}
```

### 2.3 NoteRevisionDTO (TODO)

```json
{
  "id": 99,
  "note_id": 12,
  "content_md": "# Hello (old)",
  "diff": null,
  "created_at": "2025-08-13T21:31:12Z"
}
```

---

## 3) Notes APIs

### 3.1 Create Note

* **POST** `/api/v1/notes`
* **Body** (All fields are optional)

  ```json
  { "title": "Untitled", "folder_id": 3, "content_md": "# New note" }
  ```

* **201**

  Return `NoteDTO`
* **409** (Same user, same `slug` conflict)

  ```json
  { "error": "VALIDATION_ERROR", "message": "slug already exists" }
  ```

### 3.2 Get Note

* **GET** `/api/v1/notes/{id}`
* **200**：Return `NoteDTO`
* **404**：`{ "error":"NOT_FOUND","message":"Note not found" }`
* **403**：Access to other user's resource

### 3.3 Update Note (Autosave & Optimistic Concurrency)

* **PATCH** `/api/v1/notes/{id}`
* **Body** (Some fields are optional; it's recommended to include `updated_at` for optimistic concurrency)

  ```json
  {
    "title": "New title?",
    "folder_id": 5,
    "content_md": "# Edited ...",
    "is_published": false,
    "visibility": "private",
    "updated_at": "2025-08-13T21:31:12Z"
  }
  ```

* **Behavior**

  * If `content_md` changes: Server renders to `content_html` and performs **XSS cleaning** (strict whitelist).
  * Record a `note_revisions` (diff can be empty for now).
  * **Autosave Idempotent**：If content hasn't changed, return 200 directly, `updated_at` remains unchanged.
  * **Concurrency Conflict**：Client-provided `updated_at` doesn't match server → `409 VERSION_CONFLICT`.
* **200**：Return updated `NoteDTO`
* **409** (Version Conflict)

  ```json
  {
    "error": "VERSION_CONFLICT",
    "message": "note has been modified by another client",
    "server_updated_at": "2025-08-13T21:35:40Z",
    "server_snapshot": { /* NoteDTO */ }
  }
  ```

### 3.4 Delete Note

* **DELETE** `/api/v1/notes/{id}`
* **204** No content
* **404/403** Same as above

### 3.5 List Notes

* **GET** `/api/v1/notes`
* **Query**

  * `folder_id` (Optional)
  * `q` (Fuzzy search title / content)
  * `status` (Optional: `published|draft|all`; default `all`)
  * `limit` (Default 50, max 200)
  * `offset` (Default 0)
* **200**

  ```json
  {
    "items": [ /* NoteDTO[] */ ],
    "total": 123,
    "limit": 50,
    "offset": 0
  }
  ```

---

## 4) Folders APIs

### 4.1 Create Folder

* **POST** `/api/v1/folders`
* **Body**

  ```json
  { "name": "Drafts", "parent_id": null }
  ```

* **201**：Return `FolderDTO`

### 4.2 Update Folder

* **PATCH** `/api/v1/folders/{id}`
* **Body**

  ```json
  { "name": "New Name", "parent_id": null }
  ```

* **200**：Return `FolderDTO`

### 4.3 Delete Folder

* **DELETE** `/api/v1/folders/{id}`
* **Behavior** (Minimal implementation for this sprint): If there are subfolders or notes, return 400 and prompt that it cannot be deleted (cascade strategy later)
* **204**：If empty folder
* **400** (Non-empty)

  ```json
  { "error": "VALIDATION_ERROR", "message": "folder is not empty" }
  ```

---

## 5) Tree Reorder (Drag-and-drop sorting / Batch persistence)

### 5.1 Reorder

* **POST** `/api/v1/tree/reorder`
* **Body** (Fill one of the two arrays)

  ```json
  {
    "folders": [
      { "id": 10, "sort_order": 0 },
      { "id": 11, "sort_order": 1 }
    ],
    "notes": [
      { "id": 12, "sort_order": 0, "folder_id": 10 },
      { "id": 13, "sort_order": 1, "folder_id": 10 }
    ]
  }
  ```

  * Show in ascending order by `sort_order` within the same level
  * `notes` can carry a new `folder_id` to implement cross-folder movement
* **200**

  ```json
  { "ok": true }
  ```

* **400**

  ```json
  { "error": "VALIDATION_ERROR", "message": "invalid payload" }
  ```

---

## 6) Security & Rendering

* **XSS Protection**：Server renders `content_md` to HTML using `goldmark`, then cleans it with a strict whitelist (e.g., `bluemonday.UGCPolicy()`) and saves `content_html`. Frontend preview should prioritize **server-cleaned `content_html`**.
* **Permissions**：All resource access is default limited to `user_id = current_user`. Accessing other user's resources returns `403 FORBIDDEN`.
* **Slug Uniqueness**：Constrained to `(user_id, slug)` uniqueness (allowing `slug = null`).

---

## 7) Concurrency & Autosave (Frontend Must Read)

* When calling `PATCH /notes/{id}`, please include the current note's `updated_at` field for **optimistic concurrency control**.
* **Autosave Suggestions**：Frontend should use **throttling** (~1200ms) for input saving and **debouncing** (~500ms) after drag-and-drop sorting.
* **Conflict Handling**：When receiving `409 VERSION_CONFLICT`, UI should prompt "Server has updated", and allow showing server snapshot and allowing overwrite/merge.

---

## 8) Error Code & Status Code Summary

| HTTP | error             | Description                      |
| ---: | ----------------- | ----------------------- |
|  200 | —                 | Success                      |
|  201 | —                 | Created                     |
|  204 | —                 | Deleted successfully, no content                |
|  400 | VALIDATION\_ERROR | Parameter error/business validation failed             |
|  401 | UNAUTHORIZED      | Unauthorized or invalid token           |
|  403 | FORBIDDEN         | Illegal access to other user's resources                |
|  404 | NOT\_FOUND        | Resource not found                   |
|  409 | VERSION\_CONFLICT | Optimistic lock conflict (`updated_at` mismatch) |
|  429 | RATE\_LIMITED     | Rate limit (if enabled)               |
|  500 | INTERNAL          | Server internal error                 |

---

## 9) Extensibility & Compatibility (Future Planning)

* **Tags**：`/api/v1/tags`、`/api/v1/note-tags` (Next phase)
* **Revisions**：`GET /api/v1/notes/{id}/revisions` (Next phase)
* **Search**：Current is simple `q` fuzzy query, future can upgrade to full-text search and add endpoints, without breaking compatibility.

---

## 10) Changelog

* **2025-08-13**：Add v1 Notes/Folder/Tree Reorder/Autosave/XSS contract; unify error format; define DTO and query parameters; clarify visibility and slug constraints.
* **2025-08-04**：Auth & User (Register, Login, Refresh, Get Current User). Add `stretchr/testify/assert` for testing (see project development log).

---

## 11) Appendix: Quick Checklist for Frontend

* All protected requests: Include `Authorization: Bearer <access_token>`
* `PATCH /notes/{id}`: Include `updated_at`; handle 409 popup
* Preview area: Use returned `content_html` (cleaned)
* Refresh/relogin recovery: Record `lastNoteId` (local or backend preference storage)
* Drag-and-drop sorting: Aggregate and call `/tree/reorder` once

---
