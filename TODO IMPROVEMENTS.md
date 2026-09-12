# TODO Improvements

Queued items from an autonomous review pass (2026-09-12). Each is either a design
tradeoff or a schema/behavior change judged too sensitive to apply without review.

### Restrict JWT signing methods on parse
- **Category:** Bug (defense-in-depth)
- **What:** `ParseAccessToken`/`ParseRefreshToken` (`internal/auth/jwt.go`) call
  `jwt.ParseWithClaims` without `jwt.WithValidMethods([]string{"HS256"})`, so the
  parser accepts any algorithm the token header names (HS384/HS512, or in older
  JWT libraries "none") as long as the keyfunc's secret happens to validate it.
  golang-jwt v5 already blocks the classic "none" bypass by default, but pinning
  the allowed method is a one-line, zero-risk hardening that removes the
  algorithm-confusion class entirely rather than relying on library defaults.
- **Where:** `internal/auth/jwt.go:60-86`
- **Why:** Deliberately not bundled into this pass's safe fixes since it touches
  auth-critical code path; wanted a second pair of eyes even though the change
  itself is small.
- **Risk:** Low — additive validation, no behavior change for valid tokens.
- **Effort:** Low

### Index verification_token / reset_token columns
- **Category:** Refactor (performance)
- **What:** `internal/auth/handlers.go` looks up users by
  `WHERE verification_token = ?` and `WHERE reset_token = ?`
  (`VerifyEmail`/`ResetPassword`), but neither column has an index in
  `internal/db/migrations/000001_init.up.sql` — both force a sequential scan on
  `users`, fine at template scale but a real cost once the table grows.
- **Where:** `internal/db/migrations/000001_init.up.sql` (`users` table),
  `internal/auth/handlers.go` (`VerifyEmail`, `ResetPassword`)
- **Why:** Requires a new migration (schema change), which is out of scope for a
  direct fix in this pass.
- **Risk:** Low — additive index, no query behavior change.
- **Effort:** Low

### Refresh tokens are never purged after expiry
- **Category:** Feature
- **What:** `refresh_tokens` rows are only deleted on explicit logout/rotation
  (`internal/auth/handlers.go`); an expired-but-never-used token's row lives in
  the table forever. Harmless functionally (expiry is still checked on refresh)
  but the table grows unbounded for active users who never explicitly log out.
- **Where:** `internal/models/refresh_token.go`, `internal/auth/handlers.go`
- **Why:** No cleanup job exists; adding one (cron, `DELETE ... WHERE expires_at
  < now()` on a scheduler, or a Postgres extension) is a new capability, not a
  bug fix.
- **Risk:** Low.
- **Effort:** Medium (needs a scheduler/job runner decision).
