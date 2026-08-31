# rc.30 Upgrade and Zeta Changelog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce signed rc.30 release, development, and five reusable backup refs; verify the rc.26-to-rc.30 database path; add a complete ZETA changelog; validate Dev deployment; and keep production on its verified rc.26 deployment until the owner supplies `確認發布`.

**Architecture:** Extend the integrated rc.29 release with a signed upstream rc.30 merge, preserving the ZETA usage-log and PostgreSQL migrator behavior at the two overlapping files. Rebuild each customization theme from the clean rc.30 tag in its own worktree, then generate `CHANGELOG-ZETA.md` from the final ref graph and run application, frontend, relaykit, container, and database verification before any remote or Zeabur transition.

**Tech Stack:** Git signed commits and worktrees, Go 1.22+, GORM v2, PostgreSQL 9.6 and 16, MySQL 5.7.8+, SQLite, Bun, React 19, Vitest, Docker, and Zeabur CLI.

**Spec:** `docs/superpowers/specs/2026-09-01-rc30-upgrade-zeta-changelog-design.md`

## Global Constraints

- Use `release/v1.0.0-rc.29` at `dccf03595850addfd0901523e5ace279ecb9da83` as the rc.30 release ancestor.
- Merge exact upstream tag `v1.0.0-rc.30` at `27ff6a8767e728f879d52770c273d4f73214a430` with a signed non-fast-forward commit.
- Preserve all five themes: reasoning, Chat Completions/Responses compatibility, usage-log realtime refresh, channel-affinity test isolation, and PostgreSQL AutoMigrate compatibility.
- Rebuild every rc.30 backup from the clean rc.30 tag; never merge a complete rc.29 backup branch into rc.30.
- Keep `relaykit/` independently buildable with `GOWORK=off`.
- Support SQLite, MySQL 5.7.8+, PostgreSQL 9.6, and PostgreSQL 16.
- Use `common.*` JSON wrappers in root-module business code.
- Preserve explicit zero values in optional relay request fields.
- Every new commit must be signed and contain exactly `Co-authored-by: Codex <noreply@openai.com>`.
- Push only to `origin`; never push or prepare work for `upstream`.
- Production currently runs `release/v1.0.0-rc.26` at `c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7`.
- Keep production on rc.26 until the owner supplies the exact phrase `確認發布` after database, backup, restore, signature, and deployment evidence is presented.
- Never print, persist, or pass live secret values through command output or committed files.
- Use `apply_patch` for every file edit.

---

### Task 1: Integrate the upstream rc.30 tag

**Files:**
- Modify: `model/log.go`
- Modify: `model/main.go`
- Add from upstream: `model/token_migration.go`
- Add from upstream: `model/token_migration_test.go`
- Preserve: `model/postgres_migrator.go`
- Preserve: `model/postgres_migration_test.go`

**Interfaces:**
- Consumes: signed specification revision
  `43b18f09db5e42b7e6a01677b6498e41f567c165`, the signed plan commit that
  follows it, and upstream tag `27ff6a8767e728f879d52770c273d4f73214a430`.
- Produces: signed merge commit whose first parent is the plan-bearing rc.30
  release tip descended from `43b18f09d` and whose second parent is the exact
  rc.30 tag.

- [ ] **Step 1: Reconfirm the immutable inputs and clean index**

Run:

```text
git status --short --branch
git rev-parse HEAD
git rev-parse v1.0.0-rc.30
git merge-base release/v1.0.0-rc.29 v1.0.0-rc.30
git merge-base --is-ancestor 43b18f09db5e42b7e6a01677b6498e41f567c165 HEAD
git verify-commit HEAD
```

Expected: clean `release/v1.0.0-rc.30`; current HEAD contains this plan and is
a signed descendant of `43b18f09d...`; tag `27ff6a876...`; merge-base
`2b6f1dfef...`; current HEAD signature valid.

- [ ] **Step 2: Start the signed-history merge without committing**

Run:

```text
git merge --no-ff --no-commit v1.0.0-rc.30
```

Expected: conflicts only in `model/log.go` and `model/main.go`; upstream adds
the token migration files.

- [ ] **Step 3: Resolve `model/log.go`**

Retain upstream's separate `rateStat` scan so RPM/TPM scanning cannot overwrite
the quota fields. Retain the ZETA log-event publication and every existing
usage-log query/filter behavior. Resolve only the conflict markers with
`apply_patch`.

Required observable behavior:

```go
var rateStat struct {
	Rpm int
	Tpm int
}
if err := rpmTpmQuery.Scan(&rateStat).Error; err != nil {
	return stat, errors.New("查询统计数据失败")
}
stat.Rpm = rateStat.Rpm
stat.Tpm = rateStat.Tpm
```

- [ ] **Step 4: Resolve `model/main.go`**

Keep the ZETA `postgresMigrationDialector` wrapper in `chooseDB`. Add the
upstream migration sequence at the beginning of `migrateDB`:

```go
if err := migrateTokenKeyUniqueness(DB); err != nil {
	return err
}
if err := migratePrefillGroupUniqueness(DB); err != nil {
	return err
}
```

Both calls must precede every `AutoMigrate` call. Preserve the rc.30
`LoginEncryptionKey` and `TaskPlugin` model registrations.

- [ ] **Step 5: Verify the resolved merge before committing**

Run:

```text
git diff --check
git diff --name-only --diff-filter=U
go test -count=1 ./model
go test -count=1 ./controller ./service
go test -count=1 ./relay ./relay/channel/openai ./relay/channel/zhipu_4v
```

Expected: no unresolved files; whitespace check passes; all focused packages
pass. PostgreSQL integration cases may skip only in this pre-container step.

- [ ] **Step 6: Create and verify the signed merge commit**

Commit message:

```text
build: integrate upstream rc30

Merge v1.0.0-rc.30 while preserving Zeta usage-log events and the
PostgreSQL named-unique migration guard.

Co-authored-by: Codex <noreply@openai.com>
```

Run the commit with `-S`, then verify:

```text
git verify-commit HEAD
git show -s --no-show-signature --format=%B HEAD
git rev-list --parents -n 1 HEAD
git status --short --branch
```

Expected: valid ED25519 signature, exact trailer, two parents, and clean tree.

### Task 2: Prove the rc.26-to-rc.30 schema path

**Files:**
- Test: `model/prefill_group_migration_test.go`
- Test: `model/token_migration_test.go`
- Test: `model/postgres_migration_test.go`
- Inspect: `model/password_crypto.go`
- Inspect: `model/task_plugin.go`
- Inspect: `model/prefill_group.go`
- Inspect: `model/main.go`

**Interfaces:**
- Consumes: Task 1 merge result and disposable database DSNs.
- Produces: evidence for fresh, rc.26, rc.27, rc.28, rc.29, failure-atomicity,
  idempotency, and rollback compatibility cases without touching production.

- [ ] **Step 1: Rebuild the schema-change inventory**

Run:

```text
git diff --name-status v1.0.0-rc.26..v1.0.0-rc.30 -- model
git diff --unified=3 v1.0.0-rc.26..v1.0.0-rc.30 -- model/main.go model/password_crypto.go model/task_plugin.go model/prefill_group.go model/token_migration.go
```

Expected inventory: `login_encryption_keys`, `task_plugins`,
`prefill_groups.deleted_at` plus `uk_prefill_name`, and standalone
`idx_tokens_key`. Any additional schema object requires a named fixture before
continuing.

- [ ] **Step 2: Start disposable database versions**

Use dedicated `--rm` containers with dummy local-only passwords:

```text
docker run --rm --name new-api-rc30-pg96 -e POSTGRES_PASSWORD=rc30_test_only -p 55496:5432 -d postgres:9.6
docker run --rm --name new-api-rc30-pg16 -e POSTGRES_PASSWORD=rc30_test_only -p 55416:5432 -d postgres:16
docker run --rm --name new-api-rc30-mysql57 -e MYSQL_ROOT_PASSWORD=rc30_test_only -e MYSQL_DATABASE=newapi_test -p 53357:3306 -d mysql:5.7
```

Wait for each engine's native readiness command, not a fixed sleep.

- [ ] **Step 3: Run the common real-PostgreSQL tests on 9.6 and 16**

For each PostgreSQL DSN, run:

```text
go test -count=1 -v ./model -run TestMigratePrefillGroupUniquenessPostgreSQL
go test -count=1 -v ./model -run TestMigrateTokenKeyUniquenessPostgreSQL
go test -count=1 -v ./model -run TestChooseDBPostgreSQL
```

Inject `TEST_POSTGRES_DSN` through an ignored temporary PowerShell script made
with `apply_patch`. The output must show executed cases rather than `SKIP`.

On PostgreSQL 9.6, separately query `cardinality`, `to_regclass`,
`pg_index.indisready`, and `pg_index.indexprs`; record successful values. Run
the `NULLS NOT DISTINCT` rejection case only on PostgreSQL 15+.

- [ ] **Step 4: Run SQLite and MySQL coverage**

Run the SQLite model suite normally. Inject the disposable MySQL DSN as
`TEST_MYSQL_DSN` and run the MySQL migration cases. Expected: no schema error,
all non-skipped cases pass, and repeated migration does not add duplicate
objects.

- [ ] **Step 5: Exercise version-shaped PostgreSQL fixtures**

Create independent schemas for rc.26, rc.27, rc.28, and rc.29. Populate at
least two preserved rows in `tokens` and `prefill_groups`, plus non-conflicting
composite and partial indexes. Apply the exact legacy constraint variants
covered by the upstream tests. Run the rc.30 migration sequence twice and
compare catalog snapshots.

Expected after rc.30: one standalone valid/ready unique `idx_tokens_key`, one
partial `uk_prefill_name`, preserved rows, preserved unrelated indexes, and no
SQLSTATE 42704.

- [ ] **Step 6: Exercise failure atomicity**

Use the upstream unknown-constraint and invalid-target-index cases. Expected:
the migration returns an error and transaction rollback preserves the original
constraint, index, and rows.

- [ ] **Step 7: Prove application rollback compatibility**

Against separate copies of the rc.30-migrated rc.26 fixture, run the rc.26
application startup twice and its `AutoMigrate` path twice. Repeat with rc.27
and rc.29 application versions. Expected: identical before-and-after catalog
snapshots, readable rows, enforced token uniqueness, and no SQLSTATE 42704 or
constraint deletion.

- [ ] **Step 8: Stop the disposable containers**

Run `docker stop` for exactly the three `new-api-rc30-*` containers. Because
they were created with `--rm`, successful stop removes only these disposable
containers.

### Task 3: Create the rc.30 development branch

**Files:**
- No tracked file changes.

**Interfaces:**
- Consumes: verified Task 1 release merge.
- Produces: `dev/v1.0.0-rc.30` at the integrated release tip.

- [ ] **Step 1: Verify the target branch is absent**

Run `git show-ref --verify refs/heads/dev/v1.0.0-rc.30`; expected: ref absent.

- [ ] **Step 2: Create an isolated development worktree**

From the repository root, create `.local-tests/worktrees/rc30-dev` on
`dev/v1.0.0-rc.30` at the Task 1 release merge. Verify its HEAD equals the
release tip and its worktree is clean.

### Task 4: Rebuild reasoning-model support

**Files:**
- Branch: `feat/v1.0.0-rc.30/reasoning-model-support`
- Test: `setting/reasoning/suffix_test.go`
- Test: `relaykit/relayconvert/reasoning/suffix_test.go`
- Test: provider reasoning and GLM tests under `relay/channel/`
- Test: pricing and model-matching tests under `model/` and `setting/ratio_setting/`

**Interfaces:**
- Consumes: clean rc.30 tag plus the 13 ordered rc.29 reasoning commits listed below.
- Produces: a clean rc.30 reasoning backup whose delta is contained in the integrated release.

- [ ] **Step 1: Create the backup worktree from the rc.30 tag**

Create `.local-tests/worktrees/rc30-reasoning` on
`feat/v1.0.0-rc.30/reasoning-model-support` at `v1.0.0-rc.30`.

- [ ] **Step 2: Cherry-pick the ordered theme history with `-x -S`**

Use this exact order:

```text
c242c586a6e9b3a5ea1eba435f382c85e78b0cf6
58388cc5749c069bf5dd98f3bcd4409c6f655944
80f411a0ba0c6f800969fa20774196dd085255b3
d0e62e6cbaa296e3bbc154dd394bca725afd3477
b65ebfce607d2fd128c7d4e312d8f74543271152
5e19a240b5537ac2db405b082f3a0cd33693f251
0c258052d2ac43e2aa179557542083fa703b2cb5
5409dcbc1612c5dcb68965a09086ab06cbd226f3
0fd6e8d9fa9a843cebfe411a8d76a8a8af67bc23
8c5721f815d891e42599a0073c782a27cae0d166
99184f095d652e2eafe7d23393b49ce6bf198f1c
c0d641470e3419efba35ca767181ea49449e9f50
91450384951ac181a8d25ce2774743afb6e3f62c
```

Resolve only rc.30 overlaps, retaining upstream behavior. Every recreated
commit must preserve the original subject, contain the `-x` source trailer,
contain exactly one Codex trailer, and have a valid signature.

- [ ] **Step 3: Run focused reasoning tests**

Run the full tests for `setting/reasoning`, `setting/ratio_setting`,
`relaykit/relayconvert/reasoning`, `relay/channel/claude`,
`relay/channel/deepseek`, `relay/channel/moonshot`, `relay/channel/openai`,
`relay/channel/xai`, `relay/channel/zhipu_4v`, `relay/helper`, and `model`.

- [ ] **Step 4: Verify theme isolation**

Compare `v1.0.0-rc.30..HEAD` with the reasoning portion of the integrated
release. Expected: no usage-log SSE, channel-affinity fixture, or PostgreSQL
migrator files in this backup.

### Task 5: Rebuild Chat Completions and Responses compatibility

**Files:**
- Branch: `feat/v1.0.0-rc.30/chatcompletions-responses-compat`
- Test: `service/openaicompat/chat_responses_compat_test.go`
- Test: conversion tests under `relaykit/relayconvert/internal/`

**Interfaces:**
- Consumes: clean rc.30 tag and source commit `965b46fd5bb3188cfb779b86b03f88fab8debb31`.
- Produces: isolated rc.30 compatibility backup.

- [ ] **Step 1: Create the backup worktree from the rc.30 tag**

Create `.local-tests/worktrees/rc30-chatcompat` on
`feat/v1.0.0-rc.30/chatcompletions-responses-compat`.

- [ ] **Step 2: Cherry-pick the source with `-x -S`**

Resolve any rc.30 conversion overlap while retaining upstream additions and
the chat-only built-in-tool cleanup plus reasoning-summary preservation.

- [ ] **Step 3: Verify the independent module and root paths**

Run root `service/openaicompat` focused tests. From `relaykit/`, run
`GOWORK=off go test ./...`, `go build ./...`, and `go vet ./...`.

- [ ] **Step 4: Verify theme isolation and signature**

Expected: only compatibility paths differ from rc.30 tag; commit signature,
source trailer, and Codex trailer are valid.

### Task 6: Rebuild usage-log realtime refresh

**Files:**
- Branch: `fix/v1.0.0-rc.30/usage-logs-realtime-refresh`
- Modify during conflict resolution: `model/log.go`
- Test: `model/log_event_test.go`
- Test: `web/src/features/usage-logs/components/__tests__/usage-logs-provider.test.tsx`
- Test: usage-log libraries under `web/src/features/usage-logs/lib/`

**Interfaces:**
- Consumes: clean rc.30 tag and ordered commits `601b53b61e155075cc68bda5b3f4ee0c5add04ae`, `78e4960269ed35c6ea1d90b10205fc347fd76976`.
- Produces: rc.30 usage backup containing both upstream quota-stat scan and ZETA event-driven refresh.

- [ ] **Step 1: Create the backup worktree from the rc.30 tag**

Create `.local-tests/worktrees/rc30-usage` on the named rc.30 usage branch.

- [ ] **Step 2: Cherry-pick both commits with `-x -S`**

Resolve `model/log.go` by preserving rc.30's `rateStat` scan and the log-event
publication. Keep event-driven SSE behavior and token-rotation resubscription;
do not introduce fixed-interval polling.

- [ ] **Step 3: Run backend and frontend focused tests**

Run `go test -count=1 ./model ./controller ./router`. From `web/`, run the three
usage-log test files, `bun run typecheck`, and scoped `oxlint`/`oxfmt` checks on
the modified files.

- [ ] **Step 4: Verify theme isolation**

Expected: no reasoning, compatibility, affinity, or PostgreSQL migrator delta.
Both rebuilt commits must have valid signatures and trailers.

### Task 7: Rebuild channel-affinity test isolation

**Files:**
- Branch: `fix/v1.0.0-rc.30/channel-affinity-test-isolation`
- Test: `controller/channel_test_internal_test.go`
- Test: `service/channel_affinity_usage_cache_test.go`

**Interfaces:**
- Consumes: clean rc.30 tag and source commit `bc4e43801de037257a6b00f6b44c23b8482b5cce`.
- Produces: isolated rc.30 affinity-test backup.

- [ ] **Step 1: Create the branch and cherry-pick with `-x -S`**

Create `.local-tests/worktrees/rc30-affinity` from the rc.30 tag and apply the
source commit. Preserve rc.30 test changes.

- [ ] **Step 2: Run focused tests repeatedly without timing tricks**

Run the affected controller and service tests with `-count=10`. Expected:
deterministic passes, no random inputs, fixed sleep, or log-only assertions.

- [ ] **Step 3: Verify isolation and commit metadata**

Expected: only affinity test fixtures differ from the rc.30 tag; signature,
source trailer, and Codex trailer are valid.

### Task 8: Rebuild PostgreSQL AutoMigrate compatibility

**Files:**
- Branch: `fix/v1.0.0-rc.30/postgres-automigrate-compat`
- Modify during conflict resolution: `model/main.go`
- Add: `model/postgres_migrator.go`
- Add: `model/postgres_migration_test.go`

**Interfaces:**
- Consumes: clean rc.30 tag and source commit `0a8422e16490cc7854dd20c9d0ca5767498dea9e`.
- Produces: rc.30 PostgreSQL backup containing upstream token/prefill migrations and the generic named-unique guard.

- [ ] **Step 1: Create the backup worktree and cherry-pick with `-x -S`**

Create `.local-tests/worktrees/rc30-postgres` from the rc.30 tag. Resolve
`model/main.go` so the upstream token and prefill calls remain before
`AutoMigrate`, while `chooseDB` still wraps only PostgreSQL.

- [ ] **Step 2: Run the real PostgreSQL focused matrix**

Run the upstream prefill/token cases and all four
`TestChooseDBPostgreSQL...` tests against PostgreSQL 9.6 and 16 without skips.
Verify savepoint rollback, `gorm.ErrDuplicatedKey` translation, identifier
length 63, repeated table-override `AutoMigrate`, and no SQLSTATE 42704.

- [ ] **Step 3: Verify isolation and patch equivalence**

Compare the backup against the rc.30 tag and the PostgreSQL paths of the
integrated release. Expected: no unrelated theme files; rebuilt source patch
is range-diff equivalent except for the rc.30 `main.go` reconciliation.

### Task 9: Generate the complete ZETA changelog

**Files:**
- Create: `CHANGELOG-ZETA.md`
- Reference: `docs/superpowers/specs/2026-09-01-rc30-upgrade-zeta-changelog-design.md`

**Interfaces:**
- Consumes: final local rc.30 release, development, and five backup refs plus all retained historical refs.
- Produces: readable Traditional Chinese release summary and exhaustive commit ledger.

- [ ] **Step 1: Capture the final ref snapshot**

Record local and origin `release/**`, `dev/**`, `feat/**`, and `fix/**` tips.
Record verified tags, `upstream/HEAD`, `upstream/main`, live origin `main`, and
the exact optional upstream refs used. Confirm origin `main` and upstream
rc.30 both point to `27ff6a876...` before treating origin `main` as upstream.

- [ ] **Step 2: Recompute the historical sets**

Prove the fixed pre-rc.30 inventory remains 82 released, 35 backup-only, and 3
unreleased OIDs. Apply the recorded rc.26 and rc.27 epoch boundaries from the
spec. Assign the design, plan, tag merge, backup rebuilds, and later rc.30
commits to rc.30.

- [ ] **Step 3: Write the readable summary**

Use newest-first Traditional Chinese sections only for versions with ZETA
changes. Each entry names the user-visible or maintenance behavior and links
to its canonical commit.

- [ ] **Step 4: Write the technical ledger**

Use this exact column contract:

```markdown
| Version | Class | Commit | Author date | Committer date | Subject | Provenance |
| --- | --- | --- | --- | --- | --- | --- |
```

Include every released, backup-only, and unreleased OID once. Display linked
12-character SHAs, both ISO 8601 dates, original subjects, and canonical source
or backup provenance. Include full 40-character OIDs and all source ref tips in
the reference block generated at the same timestamp.

- [ ] **Step 5: Validate ledger completeness**

Parse every ledger SHA, expand it to a full OID with `git rev-parse`, reject
duplicates, compare the set against the Git-derived expected set, and confirm
zero missing, extra, unclassified, or duplicate OIDs.

- [ ] **Step 6: Commit the changelog**

Commit message:

```text
docs: add Zeta release changelog

Record released, backup-only, and unreleased Zeta commits by version,
Git dates, and provenance through rc30.

Co-authored-by: Codex <noreply@openai.com>
```

Sign and verify the commit. Fast-forward `dev/v1.0.0-rc.30` to the updated
release tip if no development-only work exists; otherwise merge release into
development without rewriting history.

### Task 10: Run the complete release gate

**Files:**
- No planned tracked changes; a failed gate returns to the owning task through a `fix/rc30/...` branch.

**Interfaces:**
- Consumes: integrated release, aligned development branch, changelog, and five backups.
- Produces: current test/build/signature/container evidence suitable for remote publication.

- [ ] **Step 1: Verify Git graph and metadata**

Verify every new commit signature and exact Codex trailer. Verify tag-merge
parents, dev/release alignment, backup bases at rc.30 tag, clean trees,
`git diff --check`, theme isolation, and remote absence of rc.30 refs before
push.

- [ ] **Step 2: Run root Go gates**

Run:

```text
go test -count=1 ./...
go build ./...
go vet ./...
```

The two recorded Windows HTTP/2 GOAWAY cases require a matching unchanged-base
reproduction and a passing Linux run. Any other failure blocks publication.

- [ ] **Step 3: Run relaykit independently**

From `relaykit/`, run:

```text
GOWORK=off go test -count=1 ./...
GOWORK=off go build ./...
GOWORK=off go vet ./...
```

- [ ] **Step 4: Run frontend gates**

From `web/`, run:

```text
bun run test
bun run typecheck
bun run lint
bun run format:check
bun run build:check
```

- [ ] **Step 5: Build and smoke-test the Docker image**

Build a uniquely tagged local rc.30 image. Start it with a disposable SQLite
database and a loopback-only port, wait on `/api/status`, verify rc.30 runtime
metadata, then stop the exact temporary container.

- [ ] **Step 6: Run the authenticated usage-log regression**

Start the backend and frontend locally. After the owner completes browser
login, send one real `/v1/chat/completions` request using a temporary local
token without printing it. Verify backend log creation and the already-open
page's event-driven update. Keep servers running until the owner confirms
manual testing is complete.

### Task 11: Publish unmonitored refs and verify Dev

**Files:**
- No tracked file changes.

**Interfaces:**
- Consumes: Task 10 release evidence.
- Produces: verified origin rc.30 refs and healthy isolated Dev deployment while production stays on rc.26.

- [ ] **Step 1: Push backups first**

Push the five exact rc.30 backup refs to `origin`, then read each back with
`git ls-remote origin`. Stop on any mismatch.

- [ ] **Step 2: Push development and unmonitored release**

Push `dev/v1.0.0-rc.30`, verify it remotely, then push the unmonitored
`release/v1.0.0-rc.30` last and verify it. Confirm the production Zeabur
project still monitors rc.26 before the release push.

- [ ] **Step 3: Switch only the isolated Dev project**

Resolve the Dev project and New-API service by name at runtime. Point Dev to
`dev/v1.0.0-rc.30`, then inspect build logs, runtime logs, CPU, memory,
container-local `/api/status`, and version headers. Production remains
untouched.

### Task 12: Prepare and perform the production transition

**Files:**
- No secret or backup identifier is committed to the repository.

**Interfaces:**
- Consumes: healthy Dev rc.30 deployment and explicit owner permissions.
- Produces: rotated credentials, verified production backup, and—only after
  `確認發布`—healthy production rc.30.

- [ ] **Step 1: Obtain explicit authority for credential rotation**

Rotate the PostgreSQL runtime credential through Zeabur only after the owner
authorizes that external change. Update all dependent services atomically and
verify redacted connectivity without printing the value.

- [ ] **Step 2: Inspect the actual production catalog read-only**

Use the database selected by the New-API service `SQL_DSN`, rechecking the
database and schema names. Inventory all single-column UNIQUE constraints and
backing indexes inside a read-only transaction. Stop on unknown token/prefill
objects, invalid indexes, or blocking long transactions.

- [ ] **Step 3: Create and restore-test the production backup**

Create a current backup, record its identifier and timestamp outside Git, and
restore it into a disposable database. Run row-count and schema checks against
the restored copy.

- [ ] **Step 4: Present the production evidence gate**

Present test commands/results, release SHA, five backup refs, signatures,
redacted catalog inventory, credential-rotation result, backup status, restore
result, and rc.26 rollback evidence. Accept only exact `確認發布`.

- [ ] **Step 5: Switch production after exact approval**

Point the production project to `release/v1.0.0-rc.30`. Inspect build/runtime
logs, CPU, memory, container-local health, allowlisted API paths, and public
`/api/status`. Verify the reported version is rc.30.

- [ ] **Step 6: Inventory old refs before cleanup**

After production is healthy, list every local and origin superseded version
ref. Delete nothing until the owner explicitly approves that exact inventory.
