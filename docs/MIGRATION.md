# Migration Guide: API Mode -> Mongo Mode

This guide helps existing `nest-cli` users migrate from API compatibility mode to direct Mongo mode.

## Why Migrate
Mongo mode makes ciphertext storage fully developer/team-owned:
- CLI writes encrypted envelopes directly to your MongoDB.
- No default dependency on `platanist-nest` storage endpoints.

## Preconditions
- You have a reachable MongoDB instance.
- You have backup/snapshot strategy for target DB.
- You understand that access/audit controls move closer to your own Mongo operations.

## Phase 1: Prepare Mongo Access

1. Create a Mongo user with least privilege for target DB.
2. Decide URI handling:
- store in config (`mongo_uri`) if preferred,
- or environment variable for reduced credential-at-rest exposure.
3. Validate connectivity with:

```bash
nest doctor --origin <new-origin>
```

## Phase 2: Configure New Mongo Origin

Option A: URI in config

```bash
nest init --origin mongo-prod --mode mongo --mongo-uri "mongodb://user:pass@host:27017" --mongo-db nest_cli
```

Option B: URI in env var

```bash
export NEST_MONGO_URI_MONGO_PROD="mongodb://user:pass@host:27017"
nest init --origin mongo-prod --mode mongo --mongo-db nest_cli
```

## Phase 3: Register Keys in Mongo Mode

1. Verify local key inventory:

```bash
nest keys list
```

2. Register active key on new origin:

```bash
nest keys register --origin mongo-prod
```

3. Confirm remote key record exists:

```bash
nest keys remote-list --origin mongo-prod
```

## Phase 4: Data Migration (Ciphertext)

For each application:
1. Pull from old API-mode origin.

```bash
nest pull legacy-origin app-name --out .env.migrate
```

2. Push to Mongo-mode origin.

```bash
nest push mongo-prod app-name --file .env.migrate
```

3. Remove temporary plaintext file.

```bash
rm -f .env.migrate
```

4. Validate pull from Mongo mode:

```bash
nest pull mongo-prod app-name --out .env.verify
```

## Phase 5: Cutover

1. Update default origin:

```bash
nest config set-origin mongo-prod --mode mongo --mongo-db nest_cli
```

2. Continue standard operations:

```bash
nest push mongo-prod app-name
nest pull mongo-prod app-name
```

## Rollback Plan

If migration has issues:
1. Keep legacy API origin config untouched during migration window.
2. Switch commands back to old origin alias.
3. Re-run push/pull from legacy mode while issues are fixed.

Example rollback usage:

```bash
nest pull legacy-origin app-name
nest push legacy-origin app-name
```

## Post-Migration Hardening

1. Restrict Mongo network access and user roles.
2. Rotate any temporary migration credentials.
3. Enable DB auditing/logging at Mongo layer.
4. Revoke compromised or old keys when needed:

```bash
nest keys revoke <key-id> --origin mongo-prod
```

## Known Differences After Migration

- `nest auth login` is not required for Mongo-mode origins.
- Pull will fail if requested key is revoked.
- Token-based API controls no longer gate direct Mongo operations.

## Verification Checklist

1. `nest doctor --origin mongo-prod` succeeds.
2. `nest keys remote-list --origin mongo-prod` returns expected keys.
3. `nest push mongo-prod app-name` creates versioned ciphertext.
4. `nest pull mongo-prod app-name` decrypts successfully.
5. Revoked key test fails as expected on pull.
