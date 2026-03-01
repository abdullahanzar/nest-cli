# nest-cli Reference

This document is the complete command and integration reference for `nest-cli`.

## Overview
`nest-cli` provides encrypted `.env` synchronization with:
- `nest push <origin> <application-name>`
- `nest pull <origin> <application-name>`

Encryption happens on the client before upload.

For complete local build instructions, see `BUILD_LOCAL.md`.

## Storage Modes
Each origin uses one mode:
- `mongo` (default): direct writes to developer-owned MongoDB.
- `api` (compatibility): write/read via `platanist-nest` HTTP endpoints.

Important:
- In `mongo` mode, `origin` points to local config containing Mongo URI settings.
- In `api` mode, `origin` points to API URL settings.

## Requirements
Common:
- Go `1.22+`

Mongo mode (default):
- Access to your own MongoDB URI
- Optional DB name override via config (`mongo_database`)

Mongo URI sources (in precedence order):
1. `origins.<name>.mongo_uri` in config
2. `NEST_MONGO_URI_<ORIGIN_NAME>` environment variable
3. `NEST_MONGO_URI` environment variable

Examples:
- `NEST_MONGO_URI_ORIGIN=mongodb://...`
- `NEST_MONGO_URI_PROD_EU=mongodb://...`

API mode (compatibility):
- Running `platanist-nest` with CLI endpoints
- Valid `email` + `apiKey`
- `SESSION_SECRET` configured server-side

## Build
```bash
go mod tidy
go build -o nest .
```

Optional local install:
```bash
go install .
```

For cross-platform build targets and troubleshooting, see `BUILD_LOCAL.md`.

## License
`nest-cli` is licensed under Apache License 2.0.
See `LICENSE` for full terms and redistribution conditions.

## Command Reference

### Global flags
- `--config string`: custom config file path.

### `nest init`
Initialize local origin settings.

Mongo default example:
```bash
nest init --origin origin --mongo-uri "mongodb://user:pass@host:27017" --mongo-db nest_cli
```

API mode example:
```bash
nest init --origin legacy --mode api --api-url https://api.example.com
```

Flags:
- `--origin`: alias name, default `origin`
- `--mode`: `mongo|api` (defaults to `mongo` unless inferred from provided flags)
- `--mongo-uri`: MongoDB URI for direct mode
- `--mongo-db`: optional database name override
- `--api-url`: API base URL for compatibility mode

### `nest doctor`
Runs mode-aware diagnostics for an origin.

```bash
nest doctor --origin origin
```

Behavior:
- `mongo` mode: Mongo ping
- `api` mode: `GET /api/health`

### `nest auth login`
Authenticate API-mode origins and store bearer token in config.

```bash
nest auth login --origin legacy --email you@example.com --api-key <YOUR_API_KEY>
```

Notes:
- This command is for `api` mode only.
- `mongo` mode does not require CLI token login.

### `nest config show`
Show config summary and origin modes.

```bash
nest config show
```

### `nest config set-origin`
Set or update an origin in either mode.

Mongo mode:
```bash
nest config set-origin origin --mode mongo --mongo-uri "mongodb://user:pass@host:27017" --mongo-db nest_cli
```

API mode:
```bash
nest config set-origin legacy --mode api --api-url https://api.example.com
```

### `nest config set-profile`
Set default profile for key generation.

```bash
nest config set-profile modern
nest config set-profile nist
```

### `nest config set-token`
Manual bearer token override for API mode.

```bash
nest config set-token <TOKEN>
```

### `nest keys generate`
Generate local keypair and encrypted private key blob.

```bash
nest keys generate --profile modern
nest keys generate --profile nist
```

Prompts for passphrase.

### `nest keys list`
List local key metadata.

```bash
nest keys list
```

`*` marks active key.

### `nest keys use`
Set active key ID.

```bash
nest keys use <key-id>
```

### `nest keys location`
Show private key storage backend/path.

```bash
nest keys location
nest keys location <key-id>
```

### `nest keys register`
Register local key metadata to active origin backend.

```bash
nest keys register --origin origin
nest keys register <key-id> --origin origin
```

### `nest keys revoke`
Revoke a registered key for an origin.

```bash
nest keys revoke --origin origin
nest keys revoke <key-id> --origin origin
```

Notes:
- Currently implemented for Mongo mode.
- Pull operations deny revoked keys.

### `nest keys remote-list`
List registered remote keys from active origin backend.

```bash
nest keys remote-list --origin origin
```

### `nest push <origin> <application-name>`
Encrypt local `.env` and upload envelope.

```bash
nest push origin my-app
nest push origin my-app --file .env.production
```

Flags:
- `--file`: input path (default `.env`)

### `nest pull <origin> <application-name>`
Fetch encrypted envelope, decrypt locally, write output atomically.

```bash
nest pull origin my-app
nest pull origin my-app --out .env.recovered
```

Flags:
- `--out`: output path (default `.env`)

## Config Reference
Default path:
- `~/.nest-cli/config.yaml`

Schema:

```yaml
default_origin: origin
crypto_profile: modern
active_key_id: "<key-id>"
auth_token: "<jwt-token-for-api-mode-only>"
origins:
  origin:
    mode: "mongo"
    mongo_uri: "mongodb://user:pass@host:27017" # optional if env var is set
    mongo_database: "nest_cli"
    api_base_url: ""
    tls_pin_sha256: ""
  legacy:
    mode: "api"
    api_base_url: "https://api.example.com"
```

Notes:
- If `mode` is omitted and `api_base_url` exists, mode resolves to `api` (backward compatibility).
- Otherwise mode resolves to `mongo`.
- You may keep Mongo URI in config (preferred by some users) or use env vars.

## Crypto Reference

### Profiles
- `modern`
  - asymmetric: X25519
  - payload AEAD: XChaCha20-Poly1305
  - wrapping: HKDF(shared secret) + XChaCha20-Poly1305
- `nist`
  - asymmetric: RSA-4096
  - payload AEAD: AES-256-GCM
  - wrapping: RSA-OAEP(SHA-256)

### Envelope AAD
AAD is bound to:
- origin
- application
- key ID

## Storage and Collections
Local:
- `~/.nest-cli/config.yaml`
- `~/.nest-cli/keys/index.json`
- `~/.nest-cli/keys/<key-id>.enc` fallback blobs

Mongo mode remote collections (developer-owned DB):
- `nest_cli_keys`
- `nest_cli_secrets`
- `nest_cli_versions`

API mode remote collections (server-managed):
- depends on `platanist-nest` implementation

## API Contract (Compatibility Mode)
Used only when origin mode is `api`.

Required routes:
- `POST /api/cli/auth/login`
- `POST /api/cli/keys/register`
- `GET /api/cli/keys/list`
- `POST /api/cli/secrets/push`
- `POST /api/cli/secrets/pull`
- `GET /api/health`

## Troubleshooting

### `origin "..." not configured`
```bash
nest config set-origin origin --mode mongo --mongo-uri "mongodb://user:pass@host:27017"
```

### `origin "..." is in mongo mode but has empty mongo_uri`
```bash
nest config set-origin origin --mode mongo --mongo-uri "mongodb://user:pass@host:27017"
```

Or set env variable:

```bash
export NEST_MONGO_URI_ORIGIN="mongodb://user:pass@host:27017"
```

### `origin "..." is in api mode but has empty api_base_url`
```bash
nest config set-origin legacy --mode api --api-url https://api.example.com
```

### `origin "..." is in mongo mode and does not require auth login`
Use `nest auth login` only for API-mode origins.

### Pull key mismatch
```bash
nest keys list
nest keys use <expected-key-id>
nest pull origin my-app
```

### Pull blocked due to revoked key
Cause: key has been revoked in remote key registry for this origin.

Fix:

```bash
nest keys list
nest keys use <active-non-revoked-key>
nest pull origin my-app
```

### Doctor fails
- Mongo mode: verify Mongo URI/network/access rights.
- API mode: verify backend URL and `GET /api/health`.

## Optional Completion
```bash
nest completion bash
nest completion zsh
nest completion fish
nest completion powershell
```
