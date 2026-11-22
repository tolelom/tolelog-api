# CLAUDE.md

Guidance for Claude Code in this repository. Package map and the rules every change must follow are in `ARCHITECTURE.md`. Commands, the Swagger contract rules and deploy are in `CONTRIBUTING.md`.

## Verify

Before finishing, run `go test ./...` and `golangci-lint run`. If a handler annotation or DTO changed, run `swag init -g cmd/server/main.go --requiredByDefault` and commit `docs/` with it; CI fails on a stale `docs/`.

## Code rules

- Never edit `docs/` by hand. tolelog-web generates its TypeScript types from `docs/swagger.json`, so annotations must match what handlers actually return.
- A `//nolint` directive must name the specific linter and give a reason (`nolintlint` with `require-specific` and `require-explanation`).
- Validate request fields with the shared validator in `internal/validate`. Use `max_bytes` for byte-length limits such as bcrypt's 72 bytes, and `alphanum_underscore` for usernames.

## Deploy

The workflow ends at the GHCR push. Do not add an SSH deploy step back: port 22 is closed to the outside and deploy is GitOps (see `846c0b4`).
