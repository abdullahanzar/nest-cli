# Build and Run Locally

This guide explains how to build `nest-cli` locally for development, testing, and release-style binaries.

## Prerequisites
- Go `1.22+`
- Git
- Optional for Mongo mode validation: local or remote MongoDB instance

Check versions:

```bash
go version
git --version
```

## Clone and Enter Project

```bash
git clone <your-repo-url>
cd nest/nest-cli
```

If you already have the repository, just enter:

```bash
cd /path/to/nest/nest-cli
```

## Install Dependencies

```bash
go mod tidy
```

## Build

Create a local binary in the current directory:

```bash
go build -o nest .
```

Run help:

```bash
./nest --help
```

## Run Without Building (Dev Loop)

```bash
go run . --help
go run . config show
```

## Install Globally (Optional)

Install to your Go bin path:

```bash
go install .
```

Then run from anywhere (if `$GOBIN` or `$GOPATH/bin` is on your `PATH`):

```bash
nest --help
```

## Format and Test

```bash
gofmt -w ./cmd ./internal
go test ./...
```

For full test matrix details (command/flag tests, integration tests with Mongo + Docker, CI parity), see `docs/TESTING.md`.

Run Mongo integration tests locally:

```bash
go test -tags=integration ./internal/storage -run TestMongoBackendIntegrationFlow -count=1
```

## Build for Multiple Platforms

Linux amd64:

```bash
GOOS=linux GOARCH=amd64 go build -o dist/nest-linux-amd64 .
```

Linux arm64:

```bash
GOOS=linux GOARCH=arm64 go build -o dist/nest-linux-arm64 .
```

macOS arm64:

```bash
GOOS=darwin GOARCH=arm64 go build -o dist/nest-darwin-arm64 .
```

Windows amd64:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/nest-windows-amd64.exe .
```

## Automated GitHub Releases (GoReleaser)

`nest-cli` has repository-local release automation:

- Config: `.goreleaser.yaml`
- Workflow: `.github/workflows/nest-cli-release.yml`

The workflow runs on tag pushes matching `v*` and publishes release artifacts to GitHub Releases.

To backfill artifacts for an existing tag, run the `nest-cli release` workflow manually from GitHub Actions and provide the tag name in the `tag` input.

Release flow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Generated artifacts include:
- Linux/macOS/Windows binaries
- `.tar.gz` archives (`.zip` on Windows)
- `checksums.txt`

## Quick Functional Smoke Test

Use an isolated config file so your main config is untouched.

```bash
./nest --config /tmp/nest-cli-test.yaml init --origin origin --mode mongo --mongo-db nest_cli
./nest --config /tmp/nest-cli-test.yaml keys generate --profile modern
./nest --config /tmp/nest-cli-test.yaml keys list
./nest --config /tmp/nest-cli-test.yaml doctor --origin origin
```

If using env-based Mongo URI:

```bash
export NEST_MONGO_URI_ORIGIN="mongodb://user:pass@host:27017"
./nest --config /tmp/nest-cli-test.yaml config set-origin origin --mode mongo --mongo-db nest_cli
```

## Troubleshooting

`go: command not found`:
- Install Go and ensure it is on `PATH`.

`permission denied` when running `./nest`:
- Run `chmod +x ./nest`.

`origin is in mongo mode but has empty mongo_uri`:
- Set either `origins.<name>.mongo_uri` or env var `NEST_MONGO_URI_<ORIGIN>` (or `NEST_MONGO_URI`).

`doctor` fails in mongo mode:
- Verify Mongo URI, credentials, network access, and DB reachability.

## Licensing

`nest-cli` is licensed under Apache License 2.0.
See `LICENSE` in the `nest-cli` directory for full terms.
