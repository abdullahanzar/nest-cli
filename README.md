# nest-cli

`nest-cli` is a local CLI for encrypted `.env` sync.

Default behavior now uses **direct MongoDB mode**:
- developers configure their own MongoDB URI locally,
- ciphertext is written to their own MongoDB,
- no secret storage is sent to your managed backend by default.

The CLI still supports an optional **API compatibility mode** for existing setups.

## Documentation
- Full command and integration reference: `docs/REFERENCE.md`
- Detailed security guide: `docs/SECURITY.md`
- API-to-Mongo migration playbook: `docs/MIGRATION.md`
- Local build and run guide: `docs/BUILD_LOCAL.md`
- Comprehensive testing guide: `docs/TESTING.md`

## Open Source Transparency
`nest-cli` is fully open source.

- Anyone can inspect command behavior and crypto/key policy logic directly in source.
- Teams can audit, fork, self-host, and adapt workflows to their own infrastructure.
- Security claims should always be verified against current code and commit history.

The web companion includes a dedicated transparency page at `/github`.

## License
Licensed under the Apache License 2.0.

- Local project license file: `LICENSE`
- Repository root license file: `../LICENSE`

Apache 2.0 allows source usage, modification, redistribution, and hosting (including commercial use), with required notices preserved.

## Quick Start (Mongo Default)
1. Build:

```bash
go mod tidy
go build -o nest .
```

2. Initialize origin in Mongo mode:

```bash
nest init --origin origin --mongo-uri "mongodb://user:pass@host:27017" --mongo-db nest_cli
```

If you prefer env vars instead of storing URI in config:

```bash
export NEST_MONGO_URI_ORIGIN="mongodb://user:pass@host:27017"
nest init --origin origin --mode mongo --mongo-db nest_cli
```

3. Generate and register key:

```bash
nest keys generate --profile modern
nest keys register --origin origin
```

4. Push and pull encrypted `.env`:

```bash
nest push origin my-app
nest pull origin my-app
```

## API Compatibility Mode (Optional)
For legacy environments using `platanist-nest` API endpoints:

```bash
nest init --origin legacy --mode api --api-url https://api.example.com
nest auth login --origin legacy --email you@example.com --api-key <YOUR_API_KEY>
```

## Security At A Glance
Capabilities:
- client-side encryption before upload,
- strict key/fingerprint checks before decrypt on pull,
- encrypted private key storage (keyring first, file fallback),
- direct developer-owned MongoDB storage by default,
- key revocation support in Mongo mode (`nest keys revoke`).

Keep in mind:
- `.env` is plaintext after pull,
- Mongo credentials are highly sensitive,
- local workstation trust is critical.

Current limitations:
- no built-in key rotation automation,
- no refresh-token flow in API mode,
- TLS pin setting exists but is not enforced yet.
