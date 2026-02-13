# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Fiber REST API server (`tolelom_api`) with JWT authentication, GORM/MySQL persistence, and Swagger docs. Go 1.25+.

## Common Commands

```bash
# Build
go build -v ./...
go build -o fiber_api_server ./cmd/server

# Run
go run cmd/server/main.go

# Test
go test -v ./...

# Dependencies
go mod download
go mod tidy

# Regenerate Swagger docs (after changing handler annotations)
swag init -g cmd/server/main.go

# Dev with hot reload (requires air)
air
```

## Architecture

**Domain-based layered architecture** with interface DI: Handler → Service Interface → GORM/DB

- `cmd/server/main.go` — Entry point. Loads env, inits config, sets up router, starts Fiber.
- `internal/config/` — Config struct holding DB connection and JWT secret. Initializes MySQL via GORM with auto-migration.
- `internal/router/` — Route registration and DI wiring. Creates services, injects into handlers. Groups: `/api/v1/auth`, `/api/v1/posts`, `/api/v1/users`. JWT middleware applied to protected routes.
- `internal/user/` — User domain. `handler.go` (Register/Login handlers), `service.go` (AuthService interface + implementation, domain errors).
- `internal/post/` — Post domain. `handler.go` (CRUD handlers), `service.go` (Service interface + implementation with ownership checks, domain errors).
- `internal/dto/` — Request/response DTOs (`LoginRequest`, `RegisterRequest`, `CreatePostRequest`, `PostResponse`, `ErrorResponse`, etc.) and model-to-DTO conversion functions.
- `internal/model/` — GORM entities only (`User`, `Post`). Post has soft delete support.
- `internal/validate/` — Shared `go-playground/validator` instance. Use `validate.Struct(&req)`.
- `internal/middleware/` — JWT auth middleware (extracts Bearer token, sets `user_id`/`username` in Fiber locals), global error handler with request logging.
- `internal/utils/` — JWT generation/validation (HMAC-SHA256, 24h expiry, 5s leeway) and bcrypt password helpers.
- `docs/` — Auto-generated Swagger files. Do not edit manually.

## Configuration

Environment variables loaded from `.env` (see `.env_example`): `PORT`, `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `JWT_SECRET`. Defaults exist in `config.go` (localhost MySQL on 3306, port 8080).

## CI/CD

GitHub Actions (`.github/workflows/ci-cd.yml`): on push to main, runs build+test, then deploys a Darwin ARM64 binary via SCP/SSH.
