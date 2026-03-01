# Testing nest-cli

This guide documents the complete test strategy for `nest-cli`, from fast local checks to full MongoDB-backed integration coverage.

## Test Layers

`nest-cli` tests are intentionally split into layers so contributors can get fast feedback first and still validate production behavior.

1. Command and flag behavior tests (`cmd/`)
- Validates argument counts, required flags, defaults, env fallback behavior, origin resolution, and key help text snippets.
- Focuses on CLI behavior contracts.

2. Internal unit tests (`internal/`)
- Config load/save/defaults and Mongo URI precedence.
- Crypto envelope/private blob round-trips and tamper/wrong-passphrase failures.
- Key manager generate/list/find/location/load-private behavior.
- Storage factory and API backend behavior.

3. Integration tests (`internal/storage` with `integration` tag)
- Spins up real MongoDB with Testcontainers.
- Validates register/list/push/pull/revoke flows and version increments.

## Prerequisites

Fast suite:
- Go 1.22+

Integration suite:
- Go 1.22+
- Docker running locally

Install/update dependencies once:

```bash
go mod tidy
```

## Local Test Commands

Run all fast tests (recommended default):

```bash
go test ./...
```

Run only command/flag tests:

```bash
go test ./cmd -count=1
```

Run only storage package unit tests:

```bash
go test ./internal/storage -count=1
```

Run Mongo integration tests:

```bash
go test -tags=integration ./internal/storage -run TestMongoBackendIntegrationFlow -count=1
```

Run integration tests in short mode (skips container-backed tests):

```bash
go test -tags=integration -short ./internal/storage
```

## Recommended Developer Workflow

1. During feature work:

```bash
go test ./cmd ./internal/...
```

2. Before opening a PR:

```bash
go test ./...
go test -tags=integration ./internal/storage -run TestMongoBackendIntegrationFlow -count=1
```

3. Before release:

```bash
gofmt -w ./cmd ./internal
go test ./...
go test -tags=integration ./internal/storage -run TestMongoBackendIntegrationFlow -count=1
```

## CI Execution

GitHub Actions workflow: `.github/workflows/nest-cli-tests.yml`

Jobs:
1. `fast-tests`
- Runs on Ubuntu.
- Executes `go test ./...` in `nest-cli`.

2. `mongo-integration`
- Runs on Ubuntu with Docker available.
- Executes `go test -tags=integration ./internal/storage -run TestMongoBackendIntegrationFlow -count=1`.

Both jobs run on pushes/PRs affecting `nest-cli/**` and workflow files.

## What Is Covered

Command and flag coverage includes:
- Root help and key command visibility.
- `init` mode validation and Mongo URI env fallback.
- `config set-origin` validation and env fallback.
- `auth login` required flags and Mongo-mode rejection.
- Key command validations (`generate`, `location`, `register`, `revoke`).
- `push` and `pull` arg validation and active-key requirements.
- `resolveOrigin` failure modes.

Internal coverage includes:
- Config default/load/save/load-or-create behavior.
- Mongo URI precedence: explicit config > origin env > global env.
- Envelope encryption/decryption for both `modern` and `nist` profiles.
- Tamper detection and AAD mismatch handling.
- Encrypted private key blob round-trip and wrong passphrase handling.
- Key manager persistence and key loading behavior.
- API backend health and unsupported API revoke behavior.

Integration coverage includes:
- Mongo key registration and listing.
- Secret push with version auto-increment.
- Secret pull of latest version.
- Key revocation and pull rejection after revoke.
- Mongo health check.

## Troubleshooting

`no required module provides package github.com/testcontainers/testcontainers-go`
- Run `go mod tidy` in `nest-cli`.

Integration test fails with Docker connection errors
- Ensure Docker daemon is running.
- Verify your user can access Docker.

`context deadline exceeded` while starting Mongo container
- Network/image pull may be slow.
- Retry once and verify Docker can pull `mongo:7`.

`origin is in mongo mode but has empty mongo_uri`
- Set `origins.<name>.mongo_uri` or export `NEST_MONGO_URI_<ORIGIN>` / `NEST_MONGO_URI`.

## Notes for Contributors

- Keep command tests behavior-focused. Avoid full help snapshots because they are brittle.
- Keep integration tests isolated and tag-gated with `integration`.
- Prefer table-driven tests for new flag/validation additions.
