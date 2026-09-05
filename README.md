English | [Português](README.pt.md)

# back-template-go

Backend starter template in Go: Fiber, GORM + Postgres + golang-migrate, JWT auth with rotating/revocable refresh tokens, rate limiting, structured logging, and Docker, all wired together and tested end to end. Part of a backend template family (`back-template-nest`, `back-template-laravel`, `back-template-spring`, `back-template-fastapi`) that shares the same endpoint contract and error shape across different stacks.

## Contents

- [Stack](#stack)
- [Project structure](#project-structure)
- [Getting started (Docker)](#getting-started-docker--recommended)
- [Getting started (without Docker)](#getting-started-without-docker)
- [Environment variables](#environment-variables)
- [Auth](#auth)
- [Roles](#roles)
- [Error shape](#error-shape)
- [Database](#database)
- [Example CRUD resource](#example-crud-resource)
- [Testing](#testing)
- [CI/CD](#cicd)
- [Docker](#docker-1)
- [Scripts](#scripts)
- [Using this as a template](#using-this-as-a-template)
- [Design notes and gotchas](#design-notes-and-gotchas)

## Stack

- [Go](https://go.dev) 1.25
- [Fiber](https://gofiber.io) v2: Express-like router/middleware, the most widely used Go web framework
- [GORM](https://gorm.io) + [pgx](https://github.com/jackc/pgx) (via `gorm.io/driver/postgres`) + [golang-migrate](https://github.com/golang-migrate/migrate): schema versioned in SQL files, embedded in the binary via `go:embed`, applied automatically on boot
- [Postgres](https://www.postgresql.org)
- JWT auth ([golang-jwt/jwt](https://github.com/golang-jwt/jwt)): short-lived access token + longer-lived refresh token, rotated and persisted server-side for revocation (`internal/models/refresh_token.go`)
- `golang.org/x/crypto/bcrypt`: password hashing
- Fiber's built-in [`limiter`](https://docs.gofiber.io/api/middleware/limiter) middleware: rate limiting (5 req/min/IP on login/register), no extra dependency
- Mail via `net/smtp`, with a console fallback in dev (`internal/mail`): no setup required to try the auth flow locally
- `log/slog` (stdlib): structured logging, pretty text in dev, JSON in prod
- Consistent `{"error": {"code", "message", "details"}}` shape across every endpoint (`internal/apierror`)
- [go-playground/validator](https://github.com/go-playground/validator): struct-tag request validation, JSON field names in error details
- Go's `testing` + [testify](https://github.com/stretchr/testify): unit + integration tests against a real Postgres (via `fiber.App.Test`, no mocked DB)
- [golangci-lint](https://golangci-lint.run): meta-linter; native git pre-commit hook (`gofmt` + `golangci-lint`), no cross-language tooling needed
- Docker + docker-compose: multi-stage build, `CGO_ENABLED=0` static binary on a **distroless nonroot** runtime image
- GitHub Actions CI: build/lint/vet/test against a real Postgres service container, Docker image build
- `/health` for the Docker healthcheck (plus a tiny standalone `healthcheck` binary, since distroless has no shell/curl)

## Project structure

```
cmd/
  api/main.go                 # entrypoint: config, migrate, connect, server.New, listen
  healthcheck/main.go          # standalone binary for the Dockerfile HEALTHCHECK
internal/
  config/                        # env-based config, sane defaults
  server/                         # server.New — wires every route; shared by main.go and tests
  db/                              # GORM connection + embedded golang-migrate migrations
    migrations/                     # *.up.sql / *.down.sql — commit these
  models/                           # User, RefreshToken, Note (GORM)
  apierror/                         # ApiError + Fiber ErrorHandler → {"error": {...}} shape, validation binding
  auth/                             # JWT, bcrypt, RequireAuth/RequireAdmin middleware, /auth/* handlers
  account/                          # /account/* handlers
  admin/                            # /admin/users, /admin/notes handlers
  diagnostics/                      # QueryCounter (test-only GORM plugin, N+1 guard)
  notes/                            # /api/notes/* handlers (reference CRUD)
  mail/                             # SMTP send, console fallback in dev
```

## Getting started (Docker — recommended)

```bash
cp .env.example .env
# generate a real secret and drop it into .env as JWT_SECRET
openssl rand -base64 32

docker compose up -d --build
```

App: [http://localhost:8083](http://localhost:8083). Postgres is exposed on host port `5460` by default (not `5432`, to avoid clashing with a local Postgres install). Migrations run automatically on container start.

## Getting started (without Docker)

Requires a Postgres instance and [Go](https://go.dev/doc/install) 1.25+.

```bash
cp .env.example .env   # point DATABASE_URL at your own Postgres
go run ./cmd/api
```

## Environment variables

See `.env.example` for the full, commented list.

| Variable                           | Required    | Purpose                                                          |
| ------------------------------------ | ----------- | ------------------------------------------------------------------ |
| `DATABASE_URL`                       | yes         | Postgres connection string                                          |
| `JWT_SECRET`                         | yes         | ≥32 chars; access/refresh token signing                             |
| `DB_HOST_PORT/NAME/USER/PASSWORD`    | Docker only | `docker-compose.yml` defaults, used to compose `DATABASE_URL`       |
| `ENVIRONMENT`                        | no          | `dev` (pretty logs) or anything else (JSON logs); default `dev`     |
| `LOG_LEVEL`                          | no          | `debug` or anything else (info); default `info`                     |
| `MAIL_HOST`/`MAIL_PORT`/`MAIL_USERNAME`/`MAIL_PASSWORD` | no | Sends real email via SMTP; without `MAIL_HOST`, emails are logged to console instead |

`internal/config/config.go` loads these with sane defaults. Nothing panics on a missing var, but `JWT_SECRET` should always be overridden outside local dev.

## Auth

`internal/auth` (all `/auth/*`, public):

- `POST /auth/register` — 201, empty body. 409 if email taken.
- `GET /auth/verify-email?token=...` — 200 empty. 404 invalid token, 409 expired.
- `POST /auth/login` — 200, `{accessToken, refreshToken}`. 401 invalid credentials or unverified email.
- `POST /auth/refresh` — rotates the refresh token (old one deleted, new one issued). 401 if invalid/expired/revoked.
- `POST /auth/logout` — 200, idempotent.
- `POST /auth/forgot-password` — always 200 (no user-enumeration leak).
- `POST /auth/reset-password` — 200. 404/409 like verify-email.
- `PATCH /account/password`, `DELETE /account` (`internal/account`) — authenticated, `Authorization: Bearer <accessToken>`.
- Rate limited: 5 register/login attempts per 60s per IP (Fiber's `limiter` middleware, see `internal/auth/routes.go`).
- `auth.RequireAuth` (`internal/auth/middleware.go`) is the single middleware every protected route uses: it decodes and validates the bearer token, so there's no per-route duplication. `auth.RequireAdmin` layers a role check on top.

**Refresh tokens are persisted.** Unlike a purely stateless JWT scheme, each refresh token's `jti` is stored in the `refresh_tokens` table so it can be revoked or rotated server-side. Logging out or refreshing deletes the old row, so a stolen refresh token can't be replayed after rotation.

## Roles

`Role` type (`USER` | `ADMIN`, default `USER`) on `User.Role`. Never trust a role from a request body. `GET /admin/users` (`internal/admin`) is the reference admin-only endpoint, guarded by `auth.RequireAdmin`. No self-serve promotion — flip it directly in the DB for local testing: `UPDATE users SET role = 'ADMIN' WHERE email = '...';`.

## Error shape

Every error response — validation, auth, not-found, unhandled — has the same envelope, produced by `apierror.Handler` (Fiber's global `ErrorHandler`):

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "Invalid request body", "details": ["email: failed on the 'email' rule"] } }
```

`code` is one of `VALIDATION_ERROR`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `RATE_LIMITED`, `INTERNAL_ERROR`, deliberately matching the shape used by the rest of the backend family so a front-end template can swap backends with minimal changes to its error-handling code.

## Database

Schema lives in `internal/db/migrations/` (plain SQL, golang-migrate format) and `internal/models/` (GORM structs mirroring it, used for querying rather than generating the schema; this template does **not** rely on `AutoMigrate` in production). After changing the schema:

```bash
migrate create -ext sql -dir internal/db/migrations -seq add_something
# edit the generated *.up.sql / *.down.sql, then update internal/models/ to match
```

(Requires the [`migrate` CLI](https://github.com/golang-migrate/migrate#cli-usage) locally just to scaffold new migration filenames. The app itself applies migrations via the embedded library, so no CLI is needed at runtime.)

Every foreign key to `users` uses `ON DELETE CASCADE` from the first migration, so deleting an account cleans up its notes and refresh tokens automatically (see [Design notes](#design-notes-and-gotchas)).

## Example CRUD resource

`/api/notes` (`internal/notes`) is a full reference implementation: a validator-checked request → a GORM model owned by the authenticated user → a JSON response with camelCase field names matching the rest of the family's convention. Copy this shape for your first real feature, then delete `internal/notes` (and drop the `notes` table via a new migration) once you don't need the reference.

## Testing

- **Unit** (`go test ./internal/auth/...`): password hashing and JWT roundtrip, no DB — `internal/auth/auth_test.go`.
- **Integration** (`go test ./internal/server/...`): `internal/server/server_test.go` drives the full register → verify → login → notes CRUD → refresh → delete-account flow through `fiber.App.Test()` (in-process, no real network listener) against a real Postgres, with no mocked DB. Set `TEST_DATABASE_URL` to point it at a specific database (falls back to `DATABASE_URL`/its default otherwise).
- **N+1 query-count guard** (`go test ./internal/admin/...`): `internal/admin/query_count_test.go` seeds a variable number of notes, registers `diagnostics.QueryCounter` (a GORM plugin counting every SQL statement executed) on the connection, and asserts that `admin.NotesWithOwners` — the function behind `GET /admin/notes` — always runs exactly 1 SQL query, whether there are 4 notes or 8. `NotesWithOwners` uses GORM's `Joins("Owner")` (a single SQL join) instead of loading each note's owner in a separate query; if someone swaps that for a per-note lookup, this test goes red before it reaches production.
- CI spins up a Postgres service container and runs the whole suite (`go test ./...`) against it.

## CI/CD

`.github/workflows/ci.yml` runs two jobs on every push/PR:

1. **build**: `go build`, `gofmt -l` (fails on unformatted files), `golangci-lint run`, `go vet`, `go test ./...`, all against a real Postgres service container
2. **docker**: builds the production Docker image (`docker/build-push-action`, no push) to catch Dockerfile breakage early

Dependabot (`.github/dependabot.yml`) checks Go modules, GitHub Actions, and the Dockerfile weekly.

## Docker

- `Dockerfile`: multi-stage (`build` → `runtime`). The build stage compiles with `CGO_ENABLED=0` for a fully static binary; the runtime stage is `gcr.io/distroless/static:nonroot`, with no shell, no package manager, about 2MB at the base, and a non-root user by default.
- Since distroless has no `curl`/`wget`/shell for a `HEALTHCHECK CMD`, `cmd/healthcheck` is a second tiny Go binary (a plain HTTP GET to `/health`) compiled and copied into the same image.
- `docker-compose.yml`: `db` (Postgres 17, healthchecked via `pg_isready`, host port `5460` by default) and `app` (built from the Dockerfile, healthchecked via the `/healthcheck` binary, waits for `db` to be healthy).

## Scripts

| Command                          | Purpose                          |
| ----------------------------------- | ----------------------------------- |
| `go run ./cmd/api`                   | Start dev server                    |
| `go build ./...`                      | Build everything                    |
| `go test ./...`                        | Run tests                           |
| `gofmt -w .`                            | Format                              |
| `golangci-lint run ./...`                | Lint                                 |
| `git config core.hooksPath githooks`      | One-time: enable the pre-commit hook |
| `docker compose up -d --build`             | Build and start app + Postgres      |
| `docker compose down`                       | Stop                                 |

## Using this as a template

1. Click "Use this template" on GitHub
2. `go mod edit -module github.com/<you>/<repo>` and update every import path (`grep -rl obrenoalvim/back-template-go .` to find them all), and update this README
3. `cp .env.example .env`, set a real `JWT_SECRET`
4. `git config core.hooksPath githooks` (one-time, per clone; this repo's hook isn't installed automatically the way `npm install` triggers Husky)
5. `docker compose up -d --build` (or the no-Docker path above)
6. Delete `internal/notes` once you've copied its pattern for your own first feature

## Design notes and gotchas

- **`ON DELETE CASCADE` on every FK to `users`, from the first migration.** Written this way from the start because `back-template-fastapi` (built earlier in the same session) shipped without it, and `DELETE /account` broke with a live `ForeignKeyViolationError` the moment a user had any notes. Deleting a user needs to cascade at the DB level; app-level "delete children first" is one more thing to forget when a new child table shows up later.
- **`fiber.Ctx.SendStatus` is not "set status, empty body."** It fills the body with the status text (`"Created"`, `"OK"`, ...) whenever nothing else has been written: a literal 7-byte body where the rest of the backend family returns something genuinely empty. `apierror.Empty(c, status)` (`c.Status(status).Send(nil)`) is used everywhere an endpoint should return a truly empty response.
- **A logging middleware that reads the status before the error handler wrote it always logs 200.** Fiber's configured `ErrorHandler` only runs once the full middleware chain (including a logging middleware wrapping everything with `c.Next()`) has already returned control to Fiber's outer dispatcher, so a naive `status := c.Response().StatusCode()` right after `c.Next()` reads the pre-error default status. Every 4xx/5xx request logged as 200 until `requestLogger` (`internal/server/server.go`) was changed to invoke `apierror.Handler` itself when `c.Next()` returns a non-nil error, before reading the status.
- **`go-playground/validator` reports Go field names, not JSON field names, unless you tell it not to.** `fe.Field()` returns `"Email"` (the struct field) by default, not `"email"` (the JSON key), inconsistent with every other backend in the family, which reports the wire-format field name. Fixed via `validate.RegisterTagNameFunc` in `internal/apierror/bind.go`.
- **Migrations are embedded (`go:embed`), not read from disk at runtime.** The distroless runtime image has no filesystem access to a `migrations/` directory shipped separately, so `internal/db/db.go` embeds `internal/db/migrations/*.sql` directly into the compiled binary via `//go:embed all:migrations`. `go build` produces a single self-contained artifact, with no external files to copy into the Docker image beyond the binary itself.
