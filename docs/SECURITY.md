# nest-cli Security Guide

This document explains:
- what the system is capable of,
- what operators must keep in mind,
- current security limitations and residual risk.

`nest-cli` now defaults to **direct MongoDB mode**.

## Mode Comparison

| Area | Mongo Mode (default) | API Mode (compatibility) |
|---|---|---|
| Storage owner | Developer/team MongoDB | Server-managed MongoDB |
| Data path | CLI -> MongoDB | CLI -> API -> MongoDB |
| AuthN | Mongo credentials | Bearer token from `/api/cli/auth/login` |
| Policy enforcement | Mostly client-side | Mostly server-side |
| Built-in audit | Optional/limited | Stronger server-side audit potential |
| Operational complexity | Lower infra coupling | Requires API service |

## Security Capabilities

Current capabilities:
- client-side envelope encryption before upload,
- strong asymmetric key profiles (`modern` and `nist`),
- private key kept local (keyring first, encrypted fallback file),
- strict key/fingerprint checks before pull decrypt,
- key revocation and revoked-key denial in Mongo mode,
- atomic local write on pull (`.tmp` then rename),
- direct Mongo ownership by developer/team in default mode.

## Cryptographic Posture

Profiles:
- `modern`:
  - X25519 key agreement
  - XChaCha20-Poly1305 for payload and wrapping AEAD
  - HKDF-based wrap key derivation
- `nist`:
  - RSA-4096 key wrapping (OAEP-SHA256)
  - AES-256-GCM payload encryption

Private key fallback blob protection:
- KDF: Argon2id (`time=3`, `memory=64MB`, `parallelism=4`, `keyLen=32`)
- Cipher: AES-256-GCM
- File permission: `0600`

AAD binding includes:
- origin
- application
- key ID

## What To Keep In Mind (Critical)

1. Local workstation trust is the root of trust.
- If endpoint is compromised, secrets and keys are at risk.

2. `.env` is plaintext after pull.
- Pull safely and avoid writing to insecure/shared paths.

3. Mongo URI is sensitive credential material.
- Treat it like a password.
- Avoid terminal history leaks and plaintext sharing.
- Use env vars (`NEST_MONGO_URI_<ORIGIN>`, `NEST_MONGO_URI`) when you do not want URI in config.

4. In Mongo mode, control plane protections move to operator responsibility.
- Enforce Mongo RBAC, network controls, and backup policies yourself.

5. Key mismatch errors are intentional protection.
- Do not bypass them; they prevent decryption with wrong key material.

## Operator Security Checklist

1. Use strong key passphrases and rotate them periodically.
2. Restrict MongoDB users to least privilege for `nest_cli_*` collections.
3. Use network allowlists/private networking for MongoDB access.
4. Enable MongoDB encryption at rest where available.
5. Use TLS to MongoDB and verify cert chains.
6. Keep `~/.nest-cli` protected and excluded from backups if not encrypted.
7. Never commit pulled `.env` files.
8. Run pull only on trusted machines.
9. Periodically review key metadata and remove stale keys.
10. Maintain independent audit logging if using Mongo mode in teams.
11. Revoke compromised keys immediately with `nest keys revoke`.

## Current Limitations and Residual Risk

1. No built-in automated key rotation/re-encryption workflow.
- Impact: teams may keep long-lived keys.
- Mitigation: schedule manual rotation and controlled re-push.

2. No enforced TLS pinning in current CLI transport.
- Impact: trust depends on default TLS behavior and endpoint hygiene.
- Mitigation: use trusted certificates and private networks.

3. No universal memory zeroization guarantees.
- Impact: advanced memory scraping remains possible on compromised hosts.
- Mitigation: host hardening and least privilege.

4. Mongo mode does not inherently provide centralized audit semantics.
- Impact: weaker accountability in shared credential/team environments.
- Mitigation: create separate DB users per operator and maintain append-only audit trails.

5. No built-in RBAC-by-application policy in CLI logic.
- Impact: access model primarily follows Mongo credential scope.
- Mitigation: split DB users/collections by environment or app boundaries.

## Recommended Next Security Milestones

1. Implement `nest keys rotate` with guided re-encryption.
2. Add optional local signed audit trail output.
3. Enforce TLS pinning where configured.
4. Add policy scoping per application/environment.
5. Add formal secret manager integration for Mongo URIs.

## Incident Response (Mongo Default)

If compromise is suspected:
1. stop all pull operations,
2. rotate Mongo credentials,
3. generate and activate new key,
4. re-register key metadata,
5. re-encrypt/re-push sensitive applications,
6. retire old key IDs from operational use,
7. review Mongo activity logs for suspicious reads/writes.

## What the System Is Capable Of Today

- Encrypting `.env` locally before any upload.
- Storing only ciphertext envelopes in developer-owned MongoDB by default.
- Enforcing key/fingerprint checks in pull flow before local decrypt.
- Supporting both direct Mongo mode and API compatibility mode per origin.

For command-level operational details, see `docs/REFERENCE.md`.
