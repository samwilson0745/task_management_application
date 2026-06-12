# Task Management Application

A full-stack task management application with a Go REST API, PostgreSQL database, and a
Next.js frontend. Users can sign up, log in, and manage their own tasks with filtering,
search, sorting, and pagination.

In addition to the core requirements, this app includes the following bonus features:

- **Role-based access** — admins can view (read-only) all users' tasks.
- **Real-time updates** — task changes are pushed live to the UI via Server-Sent Events (SSE).
- **Optimistic UI** — toggling, creating, editing, and deleting tasks update the UI
  immediately, with automatic rollback and a toast notification if the request fails.
- **Task attachments** — attach files (max 10MB) when creating a task or from the task
  detail page; attachments are previewable inline (images/PDFs/text) or downloadable
  directly from the task card, and can be deleted.
- **Activity log** — each task records a history of create/update events with the
  acting user and timestamp.
- **Dark mode** — a toggle in the navbar with the preference persisted in `localStorage`.
- **Dockerized** — `docker compose up` runs the database, API, and frontend together.
- **CI** — GitHub Actions runs backend Go tests (against a real Postgres service) and
  frontend type-check/lint/tests/build on every push and PR.

## Tech Stack

- **Frontend:** Next.js (App Router), TypeScript, Tailwind CSS
- **Backend:** Go, [chi](https://github.com/go-chi/chi) router, [pgx](https://github.com/jackc/pgx) (PostgreSQL driver)
- **Database:** PostgreSQL
- **Auth:** JWT (signed HS256), bcrypt password hashing

## Project Structure

```
.
├── backend/    # Go REST API
├── frontend/   # Next.js app
└── docker-compose.yml
```

## Prerequisites

- Go 1.23+
- Node.js 20+
- PostgreSQL 14+ (or Docker)

## Quick Start with Docker

The fastest way to run the whole stack locally:

```bash
docker compose up --build
```

This starts:

- PostgreSQL on `localhost:5432`
- Backend API on `http://localhost:8080`
- Frontend on `http://localhost:3000`

The backend automatically runs database migrations on startup.

## Running Locally (without Docker)

### 1. Database

Create a PostgreSQL database, e.g.:

```bash
createdb taskmanager
```

### 2. Backend

```bash
cd backend
cp .env.example .env
# edit .env if needed (DATABASE_URL, JWT_SECRET, etc.)
go run ./cmd/api
```

The API will be available at `http://localhost:8080`. Migrations are applied automatically
on startup.

### 3. Frontend

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

The app will be available at `http://localhost:3000`.

## Deploying (Free Tier)

### Backend + DB on Render

A `render.yaml` blueprint is included. In the Render dashboard, choose
**New > Blueprint**, point it at this repo, and it will provision:

- A free PostgreSQL database
- The API as a Docker web service (built from `backend/Dockerfile`)
- A generated `JWT_SECRET`

Note: Render's free tier doesn't support persistent disks, so uploaded
attachments are stored on the container's ephemeral filesystem and will be
lost on redeploy/restart. For persistent attachments, upgrade to a paid plan
with a disk, or switch attachment storage to an object store like S3.

After creation, update the `ALLOWED_ORIGIN` env var on the API service to
your deployed frontend URL (e.g. `https://your-app.vercel.app`).

### Frontend on Vercel

Import this repo into Vercel, set the project root to `frontend/`, and add
the environment variable:

```
NEXT_PUBLIC_API_URL=https://your-api.onrender.com
```

Vercel will build and deploy the Next.js app automatically on every push.

## Environment Variables

### Backend (`backend/.env.example`)

| Variable        | Description                                  | Default                                                                   |
| ---------------- | --------------------------------------------- | ---------------------------------------------------------------------------- |
| `PORT`           | HTTP server port                               | `8080`                                                                        |
| `DATABASE_URL`   | PostgreSQL connection string                   | `postgres://postgres:postgres@localhost:5432/taskmanager?sslmode=disable`    |
| `JWT_SECRET`     | Secret used to sign JWTs                       | _(required)_                                                                  |
| `ALLOWED_ORIGIN` | Origin allowed for CORS (the frontend's URL)   | `http://localhost:3000`                                                       |
| `STORAGE_DIR`    | Directory where uploaded task attachments are stored | `./data/attachments`                                                   |

### Frontend (`frontend/.env.example`)

| Variable             | Description              | Default                  |
| --------------------- | -------------------------- | -------------------------- |
| `NEXT_PUBLIC_API_URL`  | Base URL of the backend     | `http://localhost:8080`   |

## API Reference

All `/tasks` routes require an `Authorization: Bearer <token>` header.

### Auth

| Method | Path           | Description                       |
| ------ | -------------- | ----------------------------------- |
| POST   | `/auth/signup` | Create an account, returns a JWT    |
| POST   | `/auth/login`  | Log in, returns a JWT               |
| GET    | `/auth/me`     | Get the current authenticated user  |

### Tasks

| Method | Path         | Description                |
| ------ | ------------ | ----------------------------- |
| POST   | `/tasks`     | Create a task                 |
| GET    | `/tasks`     | List the current user's tasks |
| GET    | `/tasks/:id` | Fetch a single task           |
| PATCH  | `/tasks/:id` | Update a task                 |
| DELETE | `/tasks/:id` | Delete a task                 |
| GET    | `/tasks/stream` | SSE stream of live task changes (`task.created`, `task.updated`, `task.deleted`) |
| GET    | `/tasks/:id/activity` | List the change history for a task |
| GET    | `/tasks/:id/attachments` | List attachments for a task |
| POST   | `/tasks/:id/attachments` | Upload a file attachment (multipart `file` field, max 10MB) |
| GET    | `/tasks/:id/attachments/:attachmentID` | View/download an attachment (served inline, so browsers preview images/PDFs/text) |
| DELETE | `/tasks/:id/attachments/:attachmentID` | Delete an attachment |

#### `GET /tasks` query parameters

- `status` — filter by `todo`, `in_progress`, or `done`
- `search` — case-insensitive search by task title
- `sort_by` — `due_date`, `priority`, or `created_at` (default: `created_at`)
- `sort_dir` — `asc` or `desc` (default: `desc`)
- `page` — page number, 1-indexed (default: `1`)
- `page_size` — items per page, max 100 (default: `20`)
- `scope=all` — **admin only**. Returns tasks for all users (each task includes the
  owner's `user_email`). Returns `403` for non-admin users.

Filtering, search, and sorting can be combined in a single request.

#### Authenticating the SSE stream

Browsers' `EventSource` API cannot set custom headers, so `GET /tasks/stream` and the
attachment download endpoint also accept the JWT as a `?token=` query parameter in
addition to the standard `Authorization: Bearer` header.

#### Task fields

```json
{
  "title": "string, required, max 200 chars",
  "description": "string, optional",
  "status": "todo | in_progress | done (default: todo)",
  "priority": "low | medium | high (default: medium)",
  "due_date": "RFC3339 timestamp or null"
}
```

### Error responses

Errors are returned as JSON in a consistent shape:

```json
{ "error": "human-readable message" }
```

Validation errors (HTTP 422) additionally include a `details` map of field -> message:

```json
{
  "error": "validation failed",
  "details": { "title": "is required" }
}
```

## Authentication & Authorization

- Passwords are hashed with bcrypt before being stored.
- On successful signup/login, a JWT (7-day expiry) is returned and stored in the browser's
  `localStorage`. The frontend sends it as a `Bearer` token on every request to `/tasks` and
  `/auth/me`.
- A page refresh re-validates the stored token via `GET /auth/me` and restores the session.
- All `/tasks` routes are protected; users can only view, update, or delete tasks they own.
  Attempting to access another user's task returns `404 Not Found` (to avoid leaking the
  existence of other users' tasks).
- **Admin role**: the `users.role` column defaults to `user`. An admin can view (read-only)
  every user's tasks via `GET /tasks?scope=all` and `GET /tasks/:id`, but creating, updating,
  and deleting are still restricted to the task's owner. There is no API to grant admin
  access (to avoid a privilege-escalation surface) — promote a user manually:

  ```sql
  UPDATE users SET role = 'admin' WHERE email = 'someone@example.com';
  ```

  After promotion, the user sees a "View all users' tasks" toggle on the Tasks page.

## Testing

### Backend

```bash
cd backend
go test ./...
```

Unit tests cover password hashing, JWT generation/parsing, and input validation helpers.

Integration tests for the HTTP handlers (signup/login, task CRUD, ownership checks,
filter/search/sort/pagination, and the activity log) require a running PostgreSQL instance.
Run them with:

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/taskmanager_test?sslmode=disable" go test ./...
```

### Frontend

```bash
cd frontend
npm test
```

Tests cover the API client (success, error, and 204 responses) and the task form
(client-side validation and submission).

### CI

`.github/workflows/ci.yml` runs on every push and pull request:

- **Backend job**: spins up a Postgres service container, then runs `go build`, `go vet`,
  and `go test ./...` (including the integration tests) against it.
- **Frontend job**: runs `tsc --noEmit`, `eslint`, `vitest`, and `next build`.

## Assumptions & Trade-offs

- **Status & priority values** are fixed enums (`todo`/`in_progress`/`done` and
  `low`/`medium`/`high`) rather than free-form strings, enforced at the database and API
  level.
- **"Mark as complete"** in the UI toggles a task between `done` and `todo` via the existing
  `PATCH /tasks/:id` endpoint rather than a dedicated endpoint.
- **Auth uses JWT in `localStorage`** rather than HTTP-only cookies for simplicity. In a
  production deployment with stricter XSS requirements, an HTTP-only cookie + CSRF token
  approach would be preferable.
- **Authorization model**: a `role` column exists on the `users` table (defaulting to
  `user`) to support a future admin role, but admin-only endpoints are not part of the core
  requirements implemented here.
- **Search** uses a case-insensitive `ILIKE` match on the task title. For larger datasets, a
  full-text search index would be more scalable.
- **Pagination** is offset-based (`page`/`page_size`), capped at 100 items per page.
- Migrations run automatically on backend startup rather than via a separate CLI step, to
  keep local setup to a single command.
- **Attachments** are stored on the local filesystem (under `STORAGE_DIR`, persisted via a
  Docker volume in `docker-compose.yml`) rather than an object store like S3, to keep the
  setup self-contained. Files are capped at 10MB and stored under a randomly generated
  filename per task to avoid path traversal/collisions.
- **Real-time updates** use Server-Sent Events (one-way server -> client) rather than
  WebSockets, since the client never needs to push data over this channel — SSE is simpler
  and works over plain HTTP with automatic reconnection built into `EventSource`.
- **Activity log** currently records `created` and `updated` actions (task deletion removes
  the task and its activity rows via `ON DELETE CASCADE`, so there is no "deleted" entry to
  show).
