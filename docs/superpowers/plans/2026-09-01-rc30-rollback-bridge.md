# rc.30 Rollback Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote the self-use fork to full rc.30 prefill semantics through a rollback-safe Bridge deployment followed by a Contract deployment, while preserving the complete ZETA changelog.

**Architecture:** Bridge runs rc.30 and GORM `v1.25.12` while retaining the recognized legacy global prefill-name object beside the partial target, so the previous rc.26 snapshot remains usable. Contract removes only that recognized legacy object after Bridge is the previous successful snapshot; both stages use bounded transaction-local PostgreSQL migration timeouts and independently gated Production changes.

**Tech Stack:** Go 1.22+, GORM v2, PostgreSQL 9.6/16/18, MySQL 5.7.8+, SQLite, Docker, Bun, React 19, Git SSH signing, Zeabur CLI.

**Spec:** `docs/superpowers/specs/2026-09-01-rc30-upgrade-zeta-changelog-design.md`

## Global Constraints

- Work only in `fix/rc30/production-review-gaps` and the named rc.30 backup, development, and release branches; ordinary code commits never go directly onto development or release.
- Preserve SQLite, MySQL 5.7.8+, PostgreSQL 9.6+, and PostgreSQL 16/18 behavior.
- `relaykit/` remains independently buildable with `GOWORK=off`.
- All actual JSON marshal/unmarshal calls use `common.*`; `encoding/json` may remain for types.
- Bridge never creates the legacy prefill object when it is absent and never drops a recognized one when it is present.
- Contract drops only allowlisted legacy prefill objects and preserves valid non-conflicting constraints and indexes.
- PostgreSQL migration transactions use `lock_timeout=5s` and `statement_timeout=30s` before `ACCESS EXCLUSIVE` locks.
- Production Bridge and Contract each require a separate exact `確認發布` after current evidence is presented.
- Push no branch or tag to `upstream`; create no custom tag.
- Never print DSNs, credentials, channel keys, provider bodies, or tokens.
- Keep local servers on ports 3000, 5173, and 15072 running until manual testing is explicitly complete.
- Use `apply_patch` for source and documentation edits; use `gofmt` only as the formatting pass.
- Every agent-created commit is signed and ends with exactly one `Co-authored-by: Codex <noreply@openai.com>` trailer.

---

### Task 1: Complete bounded PostgreSQL timeout protection

**Files:**
- Create: `model/postgres_migration_timeout.go`
- Modify: `model/postgres_migration_test.go`
- Modify: `model/token_migration.go`
- Modify: `model/prefill_group_migration.go`
- Modify: `model/prefill_group.go`

**Interfaces:**
- Produces: `configurePostgresMigrationTimeouts(tx *gorm.DB) error`
- Consumes: existing `migrateTokenKeyUniqueness(*gorm.DB) error` and `migratePrefillGroupUniqueness(*gorm.DB) error`

- [ ] **Step 1: Preserve the recorded RED evidence**

Record in the task report that the first test failed with
`undefined: configurePostgresMigrationTimeouts`, and that removing the two
call sites caused the lock test to end at the 8-second context deadline
without SQLSTATE 55P03. Do not manufacture a second RED run after the helper
already exists.

- [ ] **Step 2: Verify the timeout helper implementation**

The helper must remain exactly scoped to the active transaction:

```go
const (
	postgresMigrationLockTimeout      = "5s"
	postgresMigrationStatementTimeout = "30s"
)

func configurePostgresMigrationTimeouts(tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("configure postgres migration timeouts: database is nil")
	}
	if err := tx.Exec(
		"SELECT set_config('lock_timeout', ?, true)",
		postgresMigrationLockTimeout,
	).Error; err != nil {
		return fmt.Errorf("configure postgres migration lock timeout: %w", err)
	}
	if err := tx.Exec(
		"SELECT set_config('statement_timeout', ?, true)",
		postgresMigrationStatementTimeout,
	).Error; err != nil {
		return fmt.Errorf("configure postgres migration statement timeout: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Verify both migration call sites**

Immediately after `HasTable` succeeds and before `LOCK TABLE`, both
migrations contain:

```go
if err := configurePostgresMigrationTimeouts(tx); err != nil {
	return err
}
```

- [ ] **Step 4: Keep the JSON wrapper cleanup behavior-neutral**

In `JSONValue.Scan`, retain `encoding/json` for `json.RawMessage`, but replace
the fallback call with:

```go
b, err := common.Marshal(v)
```

- [ ] **Step 5: Run the real timeout tests on PostgreSQL 9.6 and 16**

Run the focused tests with an actual `TEST_POSTGRES_DSN` for each disposable
engine:

```powershell
go test ./model -run 'TestConfigurePostgresMigrationTimeouts|TestPostgresUniquenessMigrationsBoundLockWait' -count=1 -v
```

Expected: the settings read back as `5s` and `30s`; token and prefill lock
subtests return SQLSTATE 55P03 after the database lock timeout, not the Go
context deadline.

- [ ] **Step 6: Run the focused scanner and model suite**

```powershell
go test ./model -run 'TestJSONColumn|TestConfigurePostgresMigrationTimeouts|TestPostgresUniquenessMigrationsBoundLockWait' -count=1
go test ./model -count=1
git diff --check
```

Expected: PASS; the default model run may skip external-engine cases only when
their DSNs are absent, while Step 5 supplies the real PostgreSQL runs.

### Task 2: Implement the rc.30 Bridge prefill migration

**Files:**
- Modify: `model/prefill_group_migration.go`
- Modify: `model/prefill_group_migration_test.go`
- Test: `model/postgres_migration_test.go`

**Interfaces:**
- Consumes: `configurePostgresMigrationTimeouts(*gorm.DB) error`
- Produces: Bridge behavior for `migratePrefillGroupUniqueness(*gorm.DB) error`

- [ ] **Step 1: Write the failing Bridge cases**

Extend the PostgreSQL table tests with literal expectations for recognized
legacy constraint and standalone-index cases:

```go
type prefillMigrationExpectation struct {
	legacyConstraintCount int64
	legacyIndexCount      int64
	deletedNameReusable   bool
}
```

For `legacy_constraint`, expect `1, 0, false`. For
`legacy_standalone_index`, expect `0, 1, false`. For `fresh`, expect
`0, 0, true`. Retain the unknown-object failure cases and preserved-index
assertions.

- [ ] **Step 2: Run RED on PostgreSQL 16**

```powershell
go test ./model -run 'TestMigratePrefillGroupUniquenessPostgreSQL/(legacy_constraint|legacy_standalone_index)' -count=1 -v
```

Expected: FAIL because the current Contract implementation drops the
recognized legacy object and permits deleted-name reuse.

- [ ] **Step 3: Implement Bridge preflight and early return**

After inspecting and validating recognized conflicts, inspect the target
partial index before opening the transaction:

```go
targetIndex, err := inspectPrefillGroupNameIndex(db, tableName)
if err != nil {
	return err
}
if targetIndex.exists && !targetIndex.valid {
	return fmt.Errorf(
		"prefill group index %q has an unexpected definition",
		prefillGroupNameIndex,
	)
}
if targetIndex.valid && db.Migrator().HasColumn(&PrefillGroup{}, "DeletedAt") {
	return nil
}
```

Inside the transaction, apply the timeout helper, re-inspect the allowlisted
conflicts, add `DeletedAt` when absent, and create or validate
`uk_prefill_name`. Remove the two loops that drop recognized constraints and
indexes. If no conflict existed before the transaction, retain the existing
early return so Bridge never creates a legacy object.

- [ ] **Step 4: Run GREEN on PostgreSQL 9.6 and 16**

```powershell
go test ./model -run 'TestMigratePrefillGroupUniquenessPostgreSQL|TestConfigurePostgresMigrationTimeouts|TestPostgresUniquenessMigrationsBoundLockWait' -count=1 -v
```

Expected on each engine: all cases PASS; recognized global objects coexist
with a valid partial target, and repeated Bridge migration is idempotent.

- [ ] **Step 5: Run SQLite and MySQL compatibility**

```powershell
go test ./model -run 'TestMigratePrefillGroupUniquenessSQLite' -count=1 -v
```

With a disposable MySQL 5.7.44 DSN:

```powershell
go test ./model -run 'TestMigratePrefillGroupUniquenessMySQL' -count=1 -v
```

Expected: PASS with no PostgreSQL SQL emitted on either engine.

### Task 3: Prove rc.26 rollback from Bridge and retain the Contract negative control

**Files:**
- Create ignored helper: `.local-tests/prefill-rollback-check/main.go`
- Create ignored runner: `.local-tests/run-prefill-rollback-check.ps1`
- Create evidence: `.superpowers/sdd/2026-09-01-rc30-upgrade-zeta-changelog/rollback-bridge-report.md`

**Interfaces:**
- Consumes: Bridge candidate and actual rc.26 commit `c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7`
- Produces: row/catalog hashes and actual historical-application startup evidence

- [ ] **Step 1: Create a fresh Bridge database**

Use an exact disposable database name inside the PostgreSQL 16 container.
Verify absence before creation; never drop a computed or broad target.

```powershell
docker exec new-api-rc30-timeout-pg16 createdb -U postgres rc30_prefill_bridge_test
```

- [ ] **Step 2: Prepare Bridge-compatible rows**

Run the Bridge `InitDB()` path, retain both prefill objects, and create a
soft-deleted row plus active rows with distinct names. The helper verifies:

```text
legacy_constraint=1
partial_index=1
same_name_duplicates=0
```

- [ ] **Step 3: Run the actual rc.26 application twice**

For each run, invoke the helper from the rc.26 worktree so it imports rc.26
code and GORM `v1.25.2`:

```powershell
go -C D:\Data\Coding_Github\Projects\_AI\_Proxy\new-api run ./.local-tests/prefill-rollback-check verify
go -C D:\Data\Coding_Github\Projects\_AI\_Proxy\new-api run ./.local-tests/prefill-rollback-check verify
```

Expected: both `InitDB()` runs exit 0; row map and catalog SHA-256 remain
identical; no SQLSTATE 23505, 42704, or 42710 appears.

- [ ] **Step 4: Record the required negative control**

Retain the already observed Contract-only fixture result in the report:

```text
rc26_on_contract_duplicate=FAIL_EXPECTED
sqlstate=23505
check_sentinel=FAIL_EXPECTED
sentinel_sqlstate=42710
```

This evidence proves why direct Contract-to-rc.26 rollback is prohibited; it
is not a failed Bridge gate.

- [ ] **Step 5: Remove only the reviewed Bridge database**

List exact targets first. Remove the named disposable database by connecting
to `postgres`, then verify it is absent. Keep the ignored helper through the
Contract rollback test in Task 8, and keep the Bridge report without any DSN,
credential, or raw data.

### Task 4: Commit and review the Bridge topic

**Files:**
- Commit: spec commit `a94284fe5c6b10112c43bfb7832caa82945c8dff`
- Commit: Task 1 and Task 2 production/test files
- Review: complete topic diff from `97c9413e88ce48033d2638529c61810902adc842`

**Interfaces:**
- Produces: signed Bridge topic tip

- [ ] **Step 1: Run focused and root backend gates**

```powershell
go test -count=1 ./model
go test -count=1 ./...
go build ./...
go vet ./...
git diff --check
```

Expected: all 82 root packages reached and PASS; build and vet exit 0.

- [ ] **Step 2: Verify no direct JSON calls were introduced**

```powershell
sg -p 'json.Marshal($$$ARGS)' -l go model/prefill_group.go
```

Expected: no match. `encoding/json` remains only for `json.RawMessage`.

- [ ] **Step 3: Commit only Bridge code and tests**

Use this exact message structure:

```text
fix: add rc30 rollback bridge

Retain recognized prefill global uniqueness for the first rc30
deployment and bound PostgreSQL migration lock waits.

Co-authored-by: Codex <noreply@openai.com>
```

- [ ] **Step 4: Verify signature, trailer, and tree**

```powershell
git verify-commit HEAD
git log -1 --format=%B
git diff --check 97c9413e88ce48033d2638529c61810902adc842..HEAD
git status --short
```

Expected: Good SSH signature, one final Codex trailer, clean worktree.

- [ ] **Step 5: Request a read-only Bridge review**

The reviewer reads the spec, topic diff, PG9.6/16 outputs, Task 13 report, and
actual rc.26 evidence. Any finding is verified against local code before a
change is accepted.

### Task 5: Append Bridge to the PostgreSQL backup theme

**Files:**
- Modify branch: `fix/v1.0.0-rc.30/postgres-automigrate-compat`
- Create: `model/postgres_migration_timeout.go`
- Modify: `model/postgres_migration_test.go`
- Modify: `model/token_migration.go`
- Modify: `model/prefill_group_migration.go`
- Modify: `model/prefill_group_migration_test.go`
- Modify: `model/prefill_group.go`

**Interfaces:**
- Consumes: Bridge topic behavior
- Produces: signed, theme-isolated Bridge backup tip based on upstream rc.30

- [ ] **Step 1: Apply the Bridge delta to the existing backup worktree**

Use the current rc.30 PostgreSQL backup worktree, which descends from
`27ff6a8767e728f879d52770c273d4f73214a430`. Apply only database compatibility,
timeout, scanner-wrapper, and their tests; do not include the release spec,
GLM, SSE, affinity, or changelog files.

- [ ] **Step 2: Run the PostgreSQL backup gates**

Run PG9.6 and PG16 focused matrices plus:

```powershell
go test -count=1 ./model
go build ./...
go vet ./...
git diff --check v1.0.0-rc.30..HEAD
```

- [ ] **Step 3: Commit the Bridge backup delta**

```text
fix: add rc30 rollback bridge

Co-authored-by: Codex <noreply@openai.com>
```

- [ ] **Step 4: Verify theme isolation and push order**

Verify the branch diff contains only the PostgreSQL theme. Push and read back
this backup before development and release refs. The other four already
published rc.30 backup refs must remain unchanged.

### Task 6: Integrate, promote, and deploy Bridge to Dev

**Files:**
- Merge into: `dev/v1.0.0-rc.30`
- Promote into: `release/v1.0.0-rc.30`
- Update evidence: `rollback-bridge-report.md`

**Interfaces:**
- Produces: one signed Bridge release SHA shared by local dev/release refs

- [ ] **Step 1: Merge the working topic into development**

Merge without squash. Preserve the signed design and Bridge commits.

- [ ] **Step 2: Promote development with a signed release merge**

On `release/v1.0.0-rc.30`, use a signed `--no-ff` merge from development.
Verify both parents, signature, and trailer-bearing topic commits.

- [ ] **Step 3: Run the full Bridge release gate**

Run root test/build/vet, relaykit `GOWORK=off` test/build/vet, web test,
typecheck, lint, format check, build check, Docker revision smoke, authenticated
SSE, database matrices, and `git diff --check`. Baseline exceptions require
the already documented clean-base reproduction.

- [ ] **Step 4: Realign development locally**

Fast-forward development to the signed Bridge release merge, then verify local
development and release refs have the same SHA.

- [ ] **Step 5: Push Bridge refs in safe order**

Push/read backup refs first, Bridge development next, and the still-unmonitored
Bridge release last. Verify exact origin OIDs; push no tag, `main`, or upstream
ref.

- [ ] **Step 6: Verify the Bridge Dev deployment**

Confirm exact branch/SHA, PostgreSQL startup, both prefill objects, absence of
SQLSTATE 42704/23505/42710, container/public status rc.30, and current logs.

### Task 7: Gate and deploy Bridge to Production

**Files:**
- Evidence only: `rollback-bridge-report.md`
- External state: `GCP-TW_AI-Rel`

**Interfaces:**
- Produces: healthy Bridge Production snapshot and an rc.26 rollback edge

- [ ] **Step 1: Recheck Production immediately before approval**

Verify exact rc.26 branch/SHA, `prefill_groups=0`, both recognized prefill
objects, no invalid indexes, no blockers/long transactions, credential
rotation status, and latest successful backup.

- [ ] **Step 2: Create and restore a fresh Production backup**

Use PostgreSQL 18.6 with no SQL filtering. Require exact restore exit 0,
34-table row-map equality, catalog equality, and named-object validation.

- [ ] **Step 3: Present the Bridge gate and stop**

Present exact SHA, all commands/results, backup refs, signatures, live catalog,
backup/restore, Dev, and rollback evidence. Do not switch until the owner types
exactly `確認發布`.

- [ ] **Step 4: Switch Production to Bridge**

Update only `GCP-TW_AI-Rel`: point it at `release/v1.0.0-rc.30`, set VERSION to
`v1.0.0-rc.30`, and wait for the exact Bridge deployment.

- [ ] **Step 5: Verify Bridge and its rollback snapshot**

Check build/runtime logs, metrics, container/public status, allowlisted API
paths, Anubis, both prefill objects, and that the previous successful snapshot
is rc.26. Roll back immediately if any required gate fails. After Bridge is
healthy, create the local-only ref
`fix/rc30/rollback-bridge-snapshot` from
`fix/rc30/production-review-gaps`; never push this inspection ref.

### Task 8: Implement the rc.30 Contract

**Files:**
- Modify: `model/prefill_group_migration.go`
- Modify: `model/prefill_group_migration_test.go`
- Update evidence: `.superpowers/sdd/2026-09-01-rc30-upgrade-zeta-changelog/rollback-contract-report.md`

**Interfaces:**
- Consumes: Bridge release SHA and timeout helper
- Produces: final Contract semantics and a Bridge-compatible rollback edge

- [ ] **Step 1: Synchronize the working topic with Bridge development**

Merge the updated development branch into the retained working topic without
rewriting published history.

- [ ] **Step 2: Write Contract RED expectations**

Change the recognized legacy cases back to literal expectations `0, 0, true`:
after migration there is no global object, the partial target is valid, and a
soft-deleted name can be reused.

- [ ] **Step 3: Run RED on PostgreSQL 16**

```powershell
go test ./model -run 'TestMigratePrefillGroupUniquenessPostgreSQL/(legacy_constraint|legacy_standalone_index)' -count=1 -v
```

Expected: FAIL because Bridge retains the recognized global object.

- [ ] **Step 4: Restore the allowlisted Contract drop loops**

After target validation, restore:

```go
for _, constraintName := range conflicts.constraints {
	if err := migrator.DropConstraint(&PrefillGroup{}, constraintName); err != nil {
		return fmt.Errorf(
			"drop conflicting prefill group constraint %q: %w",
			constraintName,
			err,
		)
	}
}
for _, indexName := range conflicts.indexes {
	if err := migrator.DropIndex(&PrefillGroup{}, indexName); err != nil {
		return fmt.Errorf(
			"drop conflicting prefill group index %q: %w",
			indexName,
			err,
		)
	}
}
```

Retain the `configurePostgresMigrationTimeouts(tx)` call before the table lock.

- [ ] **Step 5: Run Contract GREEN on PostgreSQL 9.6 and 16**

Run prefill/token matrices, timeout/lock tests, failure atomicity, fresh×3,
rc.26/27/28/29 shapes, SQLite, and MySQL 5.7.44. Expected: all applicable
tests PASS with no unexpected skip.

- [ ] **Step 6: Prove actual Bridge rollback from Contract**

Create a Contract clone containing two same-name prefill rows, one soft-deleted
and one active. Run the exact Bridge commit twice through `InitDB()` and require
`total=2`, `active=1`, duplicate rejection, and identical catalog hashes. The
Bridge run must not recreate the legacy global object.

Create an exact detached Bridge worktree and invoke the retained helper:

```powershell
git worktree add --detach .local-tests/worktrees/rc30-bridge-rollback-check fix/rc30/rollback-bridge-snapshot
```

Use `apply_patch` to add this ignored probe as
`.local-tests/prefill-rollback-check/main.go` inside the detached worktree:

```go
package main

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const fixtureName = "rc30-review-shared-prefill"

func main() {
	common.InitEnv()
	if err := model.InitDB(); err != nil {
		panic(err)
	}

	var total, active, legacy, partial int64
	if err := model.DB.Unscoped().Model(&model.PrefillGroup{}).
		Where("name = ?", fixtureName).Count(&total).Error; err != nil {
		panic(err)
	}
	if err := model.DB.Model(&model.PrefillGroup{}).
		Where("name = ?", fixtureName).Count(&active).Error; err != nil {
		panic(err)
	}
	if err := model.DB.Raw(`
SELECT count(*)
FROM pg_catalog.pg_constraint
WHERE conrelid = to_regclass('prefill_groups')
  AND conname = 'idx_prefill_groups_name'`).Scan(&legacy).Error; err != nil {
		panic(err)
	}
	if err := model.DB.Raw(`
SELECT count(*)
FROM pg_catalog.pg_index AS index_meta
JOIN pg_catalog.pg_class AS index_class
  ON index_class.oid = index_meta.indexrelid
WHERE index_meta.indrelid = to_regclass('prefill_groups')
  AND index_class.relname = 'uk_prefill_name'
  AND index_meta.indisunique
  AND index_meta.indisvalid
  AND index_meta.indisready`).Scan(&partial).Error; err != nil {
		panic(err)
	}
	if total != 2 || active != 1 || legacy != 0 || partial != 1 {
		panic(fmt.Sprintf(
			"unexpected rollback state: total=%d active=%d legacy=%d partial=%d",
			total, active, legacy, partial,
		))
	}
	duplicate := model.PrefillGroup{
		Name:  fixtureName,
		Type:  "model",
		Items: model.JSONValue(`[]`),
	}
	if err := duplicate.Insert(); err == nil {
		panic("active duplicate unexpectedly succeeded")
	}

	var definitions []string
	if err := model.DB.Raw(`
SELECT indexname || ':' || indexdef
FROM pg_catalog.pg_indexes
WHERE schemaname = current_schema()
  AND tablename = 'prefill_groups'
ORDER BY indexname`).Scan(&definitions).Error; err != nil {
		panic(err)
	}
	digest := sha256.Sum256([]byte(strings.Join(definitions, "\n")))
	fmt.Printf(
		"BRIDGE_ROLLBACK total=%d active=%d legacy=%d partial=%d catalog_sha256=%x\n",
		total, active, legacy, partial, digest,
	)
}
```

Use `apply_patch` to add this ignored PowerShell runner as
`.local-tests/run-bridge-rollback-probe.ps1` in the repository root:

```powershell
$env:SQL_DSN = 'postgresql://postgres:rc30_timeout_test@127.0.0.1:57516/rc30_prefill_contract_test?sslmode=disable'
$env:LOG_SQL_DSN = ''
$bridgeDir = 'D:\Data\Coding_Github\Projects\_AI\_Proxy\new-api\.local-tests\worktrees\rc30-bridge-rollback-check'
go -C $bridgeDir run ./.local-tests/prefill-rollback-check
exit $LASTEXITCODE
```

Run the probe twice:

```powershell
pwsh -NoProfile -File .local-tests/run-bridge-rollback-probe.ps1
pwsh -NoProfile -File .local-tests/run-bridge-rollback-probe.ps1
```

Expected: both runs print `total=2 active=1 legacy=0 partial=1` and the same
catalog SHA-256.

Record the resolved Bridge SHA and worktree path in ignored evidence, then
remove that exact disposable worktree and Contract database after review.

- [ ] **Step 7: Commit and verify Contract**

```text
fix: complete prefill uniqueness contract

Co-authored-by: Codex <noreply@openai.com>
```

Verify Good SSH signature, exact trailer, focused tests, and a clean topic.

### Task 9: Integrate Contract, update backup, and regenerate changelog

**Files:**
- Merge into: `dev/v1.0.0-rc.30`
- Promote into: `release/v1.0.0-rc.30`
- Append branch: `fix/v1.0.0-rc.30/postgres-automigrate-compat`
- Modify: `CHANGELOG-ZETA.md`

**Interfaces:**
- Produces: final signed Contract candidate and final exhaustive ZETA ledger

- [ ] **Step 1: Merge and promote Contract**

Merge the topic into development without squash, then create a signed release
`--no-ff` merge. Verify ancestry and parents.

- [ ] **Step 2: Append Contract to the PostgreSQL backup**

Apply only the Contract delta, run PostgreSQL/theme gates, commit with the same
Contract subject and trailer, then push/read this backup before release.

- [ ] **Step 3: Regenerate the changelog inventory**

Use the signed Contract release merge as the new snapshot upper bound. Build
the union of local and origin `release/**`, `dev/**`, `feat/**`, and `fix/**`,
exclude the two malformed archive refs and their 26 OIDs, then subtract the
verified upstream/tag/main set. Include every post-142 design, Bridge,
Contract, merge, and backup OID exactly once; exclude only the final changelog
commit itself.

- [ ] **Step 4: Validate the final changelog**

Require zero missing, extra, duplicate, unclassified, and archive-overlap OIDs;
validate full-OID references, author/committer dates, newest-first readable
sections, relative commit links, and one balanced Markdown fence.

- [ ] **Step 5: Commit final changelog documentation**

```text
docs: refresh zeta changelog

Co-authored-by: Codex <noreply@openai.com>
```

This signed docs commit becomes the final Contract candidate SHA. Fast-forward
development locally to the same SHA.

### Task 10: Run Contract release gates and Dev deployment

**Files:**
- Evidence: `rollback-contract-report.md`
- External state: `GCP-TW_AI-Dev`

**Interfaces:**
- Produces: final Contract Dev runtime evidence

- [ ] **Step 1: Run every release gate on the final docs SHA**

Run root 82-package test/build/vet, relaykit independent gates, web 62-file
tests/typecheck/build-check, scoped and baseline lint/format checks, Docker OCI
revision/status smoke, authenticated SSE, PG9.6/16/18, MySQL 5.7.44, SQLite,
Bridge-on-Contract rollback, changelog validator, signatures, trailers, and
all worktree locks/status checks.

- [ ] **Step 2: Push safe non-production refs**

Push/read Contract backup refs first and Contract development next. Do not push
the now-monitored release ref.

- [ ] **Step 3: Verify Contract Dev**

Require exact branch/SHA, RUNNING status, rc.30 status body/header, no fatal
migration markers, valid partial prefill index without the legacy global
object, metrics classified by actual availability, and current logs.

### Task 11: Obtain complete external final review

**Files:**
- Update ignored Oracle bundle inputs and manifest
- Include Task 2, Task 8, Task 13, Task 14, Git manifest, and final patch

**Interfaces:**
- Produces: GPT-5.6 Sol Pro, Claude, and agy verdicts bound to the final SHA

- [ ] **Step 1: Build a complete source bundle**

Include every file in `v1.0.0-rc.30..final-contract`, the root governance,
spec, plan, changelog, committed tests, evidence reports, `git show --raw` for
the final SHA, signature output, parent list, and exact patch. Verify the bundle
entry count and SHA-256 without exposing secrets.

- [ ] **Step 2: Run the GPT-5.6 Sol Pro review**

Use the existing `Production 發布審查` conversation with model GPT-5.6 Sol and
UI effort Pro. Require the reviewer to inspect attachments and return the full
final SHA, zero Blocker, and zero Major before approval.

- [ ] **Step 3: Run Claude and agy read-only reviews**

Use xurl outside the sandbox. Give both reviewers the final patch, Bridge and
Contract reports, actual rollback evidence, and Pro findings. Verify their
claims against local files and live readbacks.

- [ ] **Step 4: Resolve findings through the review reception process**

Classify each finding as verified, attachment-only, already proven, or
incorrect. A verified Blocker or Major stops the release and returns to the
corresponding TDD task.

### Task 12: Gate and deploy Contract to Production

**Files:**
- Evidence: final `rollback-contract-report.md`
- External state: `GCP-TW_AI-Rel`

**Interfaces:**
- Produces: final healthy rc.30 Contract Production deployment

- [ ] **Step 1: Recheck Bridge Production and rollback edge**

Require exact Bridge SHA/RUNNING state, current previous-successful snapshot,
both Bridge schema objects, no invalid indexes, no blocking/long transactions,
and current health.

- [ ] **Step 2: Create a new Production backup and exact restore**

Repeat PostgreSQL 18.6 unfiltered restore, row/catalog equality, and named
object validation. Record identifiers only in ignored evidence.

- [ ] **Step 3: Present the Contract gate and stop**

Present final Contract SHA, tests, backup refs, signatures, external verdicts,
Bridge runtime, live catalog, fresh backup, restore drill, and rollback target.
Do not push the monitored release ref until the owner types exactly
`確認發布` a second time.

- [ ] **Step 4: Push the exact Contract release SHA**

Use an exact head-to-head refspec only for `release/v1.0.0-rc.30`. Verify
origin readback before observing deployment.

- [ ] **Step 5: Verify final Production**

Require exact deployment SHA/ref, build/runtime logs, metrics, container and
public rc.30 status, Anubis, allowlisted API paths, no migration SQLSTATE,
valid partial `uk_prefill_name`, absence of legacy global uniqueness, and
soft-deleted name reuse.

- [ ] **Step 6: Run one bounded Zhipu native GLM smoke**

Use one channel, one attempt, and a finite client timeout. Record only status
class and safe metadata. A non-2xx result stops further promotion claims and
uses the verified Bridge rollback snapshot when application behavior is
implicated.

### Task 13: Final audit and branch inventory

**Files:**
- Final evidence reports
- Local and origin branch inventory

**Interfaces:**
- Produces: requirement-by-requirement completion proof

- [ ] **Step 1: Realign development after Contract**

Fast-forward development to the final release SHA and read back both origin
refs. Verify every retained rc.30 backup tip and signature.

- [ ] **Step 2: Perform the completion audit**

Verify rc.30 Production, final changelog coverage, GLM hybrid behavior,
PostgreSQL safety, both confirmation records, backups/restores, reviews,
runtime health, local servers, and clean/lock-free worktrees from current
authoritative state.

- [ ] **Step 3: Present exact cleanup inventory**

List superseded rc.26/27/29 development, release, backup, working, and malformed
archive refs separately for local and origin. Do not delete anything until the
owner explicitly approves that exact inventory.

- [ ] **Step 4: Keep local test servers until manual verification ends**

Do not stop ports 3000, 5173, or 15072 until the owner explicitly confirms
manual testing is complete. After authorization, stop only the recorded
processes and verify all three ports are released.

- [ ] **Step 5: Mark the goal complete only after all evidence is current**

The final state is rc.30 Contract in Production, an exhaustive updated
`CHANGELOG-ZETA.md`, healthy runtime, verified rollback, and no unapproved
cleanup remaining.
