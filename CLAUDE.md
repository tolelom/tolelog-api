# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Fiber REST API server (`tolelom_api`) with JWT authentication, GORM/MySQL persistence, and Swagger docs. Go 1.25+.

## Common Commands

```bash
# Build
go build -o server ./cmd/server

# Run
go run cmd/server/main.go

# Test
go test -v ./...
go test -v -cover ./...          # with coverage (used in CI)

# Lint (golangci-lint: errcheck, govet, staticcheck, ineffassign, unused, forbidigo — gorm DB.Save 금지)
golangci-lint run

# Dependencies
go mod download
go mod tidy

# Regenerate Swagger docs (after changing handler annotations or DTOs; CI fails if docs/ is stale)
swag init -g cmd/server/main.go --requiredByDefault

# Dev with hot reload (requires air)
air
```

## Architecture

**Domain-based layered architecture** with interface DI: Handler → Service Interface → GORM/DB

- `cmd/server/main.go` — Entry point. Loads config, inits DB, sets up router, starts Fiber with graceful shutdown (SIGINT/SIGTERM).
- `internal/config/` — Config struct with all env vars. Initializes MySQL via GORM with auto-migration for User, Post, Tag, Comment. Includes legacy tag data migration (comma-separated → normalized tables).
- `internal/router/` — Route registration, DI wiring, middleware stack setup. Creates services, injects into handlers. Rate limiting on auth endpoints (10/min).
- `internal/user/` — User/auth domain. `handler.go` (Register/Login/RefreshToken/GetProfile/UploadAvatar), `service.go` (AuthService interface + implementation).
- `internal/post/` — Post domain. `handler.go` (CRUD + search + pagination + tag filtering), `service.go` (Service interface with ownership checks).
- `internal/comment/` — Comment domain. `handler.go` (GetComments/CreateComment/DeleteComment), `service.go` (Service interface with access control).
- `internal/image/` — Image upload handler. Accepts multipart/form-data, validates file type, stores to upload dir.
- `internal/dto/` — Request/response DTOs with validation rules and model-to-DTO conversion functions.
- `internal/model/` — GORM entities: `User`, `Post`, `Comment`, `Tag`. Post↔Tag is M:N via `post_tags` junction table. Post and Comment use soft deletes.
- `internal/validate/` — Shared `go-playground/validator` instance with custom `alphanum_underscore` and `max_bytes` (byte-length limit, e.g. bcrypt's 72 bytes) rules.
- `internal/middleware/` — `auth.go` (required + optional auth), `security.go` (X-Content-Type-Options, X-Frame-Options, CSP, etc.), `logger.go` (request logging), `error.go` (global error handler).
- `internal/upload/` — File type detection and MIME validation utilities. UUID-based filename generation (5MB max).
- `internal/auth/` — JWT token pair generation/validation (HMAC-SHA256, 15min access + 7-day refresh, 5s clock skew) and bcrypt password helpers.
- `internal/markdown/` — Markdown stripping and plain-text excerpts (list excerpts, RSS descriptions).
- `internal/permalink/` — Public site URL, title slugs, canonical post paths (`/post/{slug}-{id}`) for feed and sitemap.
- `internal/observability/` — Sentry init/flush and `CaptureException` (falls back to slog when `SENTRY_DSN` is unset).
- `docs/` — Auto-generated Swagger files. Do not edit manually. `docs/swagger.json` is the API contract tolelog-web generates its TS types from, so annotations must match what handlers return:
  - Success bodies: `dto.SuccessResponse{data=dto.X}` (never bare `dto.X` or untyped `fiber.Map`).
  - `--requiredByDefault` marks every field required unless its json tag has `omitempty`; give optional request fields `omitempty`.
  - Fields that serialize as `null` (nil pointers/slices without `omitempty`) need `extensions:"x-nullable"`.
  - Routes outside `/api/v1` (`/health`, `/feed`, `/sitemap.xml`) are not annotated.

## Route Groups

All under `/api/v1`:
- `/auth` — Register, Login, RefreshToken (rate-limited)
- `/posts` — CRUD, search, public listing with pagination + tag filtering
- `/posts/:id/comments` — Comment CRUD
- `/users/:user_id` — Profile, user posts (optional auth for private post visibility)
- `/users/avatar` — Avatar upload (auth required)
- `/upload` — General image upload (auth required)

## Configuration

Environment variables loaded from `.env` (see `.env_example`):

| Variable | Default | Notes |
|----------|---------|-------|
| `PORT` | 8080 | Server port |
| `DB_HOST` | localhost | MySQL host |
| `DB_PORT` | 3306 | MySQL port |
| `DB_USER` | root | |
| `DB_PASSWORD` | root | |
| `DB_NAME` | blog | |
| `JWT_SECRET` | (auto-generated in dev) | **Required** in production |
| `ENVIRONMENT` | development | `development` or `production` |
| `UPLOAD_DIR` | ./uploads/images | File storage path |
| `TRUSTED_PROXIES` | 172.16.0.0/12 | Comma-separated proxies whose `CF-Connecting-IP` header is trusted for the client IP (rate limits, logs) |

## CI/CD

GitHub Actions (`.github/workflows/deploy_backend.yml`): on push to main, runs lint → test → build (linux/arm64) → Docker image push to GHCR as `:latest`, `:main`, `:<sha>`.

**The workflow ends at the GHCR push — it does not deploy.** Deployment is GitOps: Renovate watches the `:main` tag and auto-pins the new digest into the `server-config` repo's compose file, and `git-sync-deploy` applies it on the server. SSH 22 is not exposed externally per security policy, so do not add an SSH deploy step back (see `846c0b4`).
