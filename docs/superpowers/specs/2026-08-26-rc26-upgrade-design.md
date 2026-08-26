# v1.0.0-rc.26 Upgrade Design

## Context

The active self-use release is `release/v1.0.0-rc.25` at signed commit
`7352232c5d3fa185f26a153640090066b37b25ff`. The matching development ref and
both remote refs point to the same commit. Upstream tag `v1.0.0-rc.26` resolves
to `8f6961c675932f406260ff0c218bc2aa0603e9b2` and contains four commits after
upstream `v1.0.0-rc.25`.

The rc.26 release removes the 32-bit wallet quota ceiling, rejects legacy
32-bit MySQL and PostgreSQL user quota schemas during startup, temporarily
stops producing 32-bit builds, adds vLLM `thinking_token_budget`, prevents
credential auto-fill in usage-log filters, and upgrades the container Bun
version to 1.4.0.

Production project `GCP-TW_AI-Rel` uses PostgreSQL. The owner confirmed on
2026-08-26 that a production PostgreSQL backup already exists for this upgrade.
The remote schema query did not return through the current Zeabur exec session,
so the four production column types remain an explicit pre-cutover check.

## Goals

1. Integrate upstream `v1.0.0-rc.26` into a new versioned self-use release
   while preserving every customization already released in rc.25.
2. Rebuild the four approved themed backup branches from the clean rc.26 tag.
3. Migrate the production PostgreSQL wallet columns to `bigint` when they are
   still 32-bit, without bypassing the rc.26 startup guard.
4. Verify quota arithmetic, database startup behavior, relaykit independence,
   frontend behavior, signatures, remote refs, and the live deployment.
5. Keep the current rc.25 GLM/OpenRouter topic unchanged and independently
   reachable.

## Non-goals

The active but unreleased branch `feat/rc25/glm-effort-openai-openrouter` is not
part of the rc.26 release candidate. It remains on its existing three commits
and will require a separate decision before it can be ported to rc.26.

The retired classic `zh-TW` customization is not recreated. No upstream pull
request, upstream branch push, custom release tag, database data rewrite, or
32-bit compatibility patch is included.

## Workspace and branch topology

All work occurs in the ignored linked worktree
`.local-tests/worktrees/rc26`. The primary checkout stays on
`feat/rc25/glm-effort-openai-openrouter` with its existing untracked scripts.

The integration order follows project governance:

1. `release/v1.0.0-rc.26` starts from `release/v1.0.0-rc.25`.
2. The signed release branch merges upstream tag `v1.0.0-rc.26` with a signed,
   non-fast-forward merge after resolving customization conflicts.
3. The integrated result is inspected customization by customization.
4. The four rc.26 themed backups are rebuilt from the clean rc.26 tag and
   contain only their own released customization slices.
5. `dev/v1.0.0-rc.26` is created from the completed rc.26 release.
6. Backup refs are pushed and verified before the release ref. Zeabur remains
   on rc.25 until every rc.26 ref, signature, and release gate is verified.

The required backup refs are:

- `feat/v1.0.0-rc.26/reasoning-model-support`
- `feat/v1.0.0-rc.26/chatcompletions-responses-compat`
- `fix/v1.0.0-rc.26/usage-logs-realtime-refresh`
- `fix/v1.0.0-rc.26/channel-affinity-test-isolation`

## Upstream integration and conflict policy

The rc.26 upstream change set overlaps released customizations in at least
`AGENTS.md`, `controller/channel-test.go`, `relay/helper/price_test.go`, and
`relaykit/dto/openai_request.go`. Conflict resolution preserves both sides when
their contracts are independent:

- Upstream quota documentation replaces the obsolete statement that persisted
  wallet columns are 32-bit. Single-request quota conversion stays bounded at
  int32, while wallet mutations use the JavaScript-safe 64-bit boundary.
- The custom channel-test isolation behavior remains while accepting upstream
  channel-test changes.
- Existing pricing regression coverage remains alongside upstream quota test
  adjustments.
- Existing Responses compatibility fields remain alongside upstream vLLM
  `thinking_token_budget` support.

Each themed customization is compared against clean rc.26 code and its existing
regression tests. Full upstream equivalence retires only that customization;
partial equivalence produces a delta-only backup. No customization is retired
from file similarity alone.

## PostgreSQL quota schema transition

rc.26 checks these `public.users` columns before GORM migrations run:

- `quota`
- `used_quota`
- `aff_quota`
- `aff_history`

PostgreSQL must report `bigint` or `int8`. SQLite is exempt because its integer
storage already supports signed 64-bit values. Production must not set
`SKIP_64BIT_QUOTA_SCHEMA_CHECK`; that escape hatch would hide an incompatible
wallet schema.

Before changing the monitored Zeabur branch, query the production schema:

```sql
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'users'
  AND column_name IN ('quota', 'used_quota', 'aff_quota', 'aff_history')
ORDER BY column_name;
```

If all four rows report `bigint`, no schema mutation is needed. If any row
reports `integer`, apply the following transaction against the already-backed-up
production database:

```sql
BEGIN;

ALTER TABLE users
  ALTER COLUMN quota TYPE bigint USING quota::bigint,
  ALTER COLUMN used_quota TYPE bigint USING used_quota::bigint,
  ALTER COLUMN aff_quota TYPE bigint USING aff_quota::bigint,
  ALTER COLUMN aff_history TYPE bigint USING aff_history::bigint;

COMMIT;
```

Run the schema query again and require exactly four `bigint` rows. A DDL error
rolls back the transaction and stops the deployment. Because PostgreSQL takes
an exclusive table lock for these type changes, the migration occurs during
the production cutover window before the rc.26 application starts accepting
traffic.

## Quota and architecture compatibility

The application is built only for 64-bit targets. CI and release artifacts no
longer claim `386` support. This fork does not restore 32-bit builds because Go
`int` is now part of the wallet's 64-bit behavior.

The upstream wallet contract uses `common.MaxWalletQuota` at the JavaScript-safe
integer boundary and `common.WalletQuotaFromDecimalStrict` for wallet and
top-up conversion. Single-request charges continue to use the existing int32
saturation policy so accumulated batches cannot approach int64 wraparound.
Custom billing-expression, price, channel, and relay paths must preserve this
separation.

Changes under `relaykit/` continue to obey independent-module rules. The root
module must not become a relaykit dependency while resolving the rc.26 DTO
overlap.

## Verification design

The integration gate includes:

1. `git diff --check` and focused diffs against both rc.25 and clean rc.26.
2. Focused quota, wallet, top-up, redemption, rate-limit, database-schema, and
   billing-expression tests introduced or changed by rc.26.
3. Existing regression tests for all four custom themes.
4. Root `go test ./...`, `go build ./...`, and `go vet ./...` after frontend
   assets exist.
5. `GOWORK=off go test ./...` and `GOWORK=off go build ./...` inside
   `relaykit/`.
6. Frontend `bun run lint`, `bun run format:check`, and `bun run build:check`,
   plus the repository-required production build that supplies `web/dist`.
7. A local runtime smoke test covering `/api/status` and database startup.
8. Signature verification for the rc.26 release merge, release head, dev head,
   and every backup head.
9. Exact local-versus-origin ref comparison before any Zeabur change.

The initial rc.25 baseline produced two known setup/runtime observations. The
root package cannot compile until `web/dist` is built, and
`TestUpstreamGetBody_HTTP2RetryAfterGracefulGoAway_PassThrough` is intermittent
on Windows. The same unchanged test failed in full rc.25 and rc.26 runs. A
20-run focused reproduction failed twice because the raw test server closes its
first connection immediately after `WriteGoAway`, allowing Windows to deliver
a reset before the client reads the GOAWAY frame. A temporary diagnostic that
kept the first connection open until the retry completed passed 20 out of 20
runs, then was fully reverted. This is a recorded pre-existing broad-suite
exception rather than an rc.26 regression; the focused provider, quota,
relaykit, build, and vet gates remain mandatory.

## Production cutover and rollback

After all rc.26 refs are pushed and verified, present test results, signed SHAs,
backup refs, the confirmed PostgreSQL backup status, and the four-column schema
result. Production cutover waits for the exact confirmation phrase
`確認發布`.

The cutover sequence is database schema verification or conditional DDL,
Zeabur monitored-branch change to `release/v1.0.0-rc.26`, deployment-state
inspection, runtime logs, CPU and memory metrics, and an HTTP `/api/status`
check that must report rc.26. The Anubis relay allowlist and private forwarding
path remain unchanged.

If deployment or startup fails, inspect deployment logs and service metrics
first. Immediate application recovery uses the previous successful Zeabur
snapshot while the branch configuration remains on rc.26. Database rollback is
separate: widening PostgreSQL `integer` columns to `bigint` is backward-readable
by rc.25, so the application can return to the previous snapshot without
narrowing those columns.

## Acceptance criteria

The upgrade is complete only when the release and dev refs are aligned at the
signed rc.26 release head, all four themed backup refs are present and verified,
the production wallet columns are confirmed as `bigint`, all applicable gates
pass, Zeabur reports a healthy rc.26 deployment, `/api/status` exposes rc.26,
and the primary rc.25 GLM topic remains unchanged.
