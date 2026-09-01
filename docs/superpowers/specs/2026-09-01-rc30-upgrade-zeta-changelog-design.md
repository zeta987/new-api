# rc.30 Upgrade and Zeta Changelog Design

## Purpose

This change promotes the integrated self-use release from
`v1.0.0-rc.29` to `v1.0.0-rc.30`, preserves every active customization,
verifies every upstream schema change between production rc.26 and rc.30,
and adds a repository-level `CHANGELOG-ZETA.md` covering ZETA-authored
history from its first released customization through rc.30.

A live Zeabur check on 2026-09-01 shows production running
`release/v1.0.0-rc.26` at
`c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7`, despite the earlier rc.27
assumption. The deployment status is `RUNNING`, and the same rc.26 release ref
exists on origin. This state must be checked again before any branch switch.
Production uses a two-stage rollback bridge described below. Each monitored
release change requires its own exact confirmation phrase `確認發布` after the
corresponding release evidence and database backup status have been presented.

## Verified starting state

- Prior integrated release:
  `release/v1.0.0-rc.29` at
  `dccf03595850addfd0901523e5ace279ecb9da83`.
- Target upstream tag:
  `v1.0.0-rc.30` at
  `27ff6a8767e728f879d52770c273d4f73214a430`.
- The common ancestor is the rc.29 upstream tag
  `2b6f1dfefbe217fed31fc0726717cc7de6958e8e`.
- The rc.29 integrated release and rc.30 tag have diverged, so rc.30 requires
  a signed merge from the prior integrated release.
- Upstream rc.30 adds a quota-statistics correction plus the
  `tokens.key` legacy uniqueness migration. Its rc.29-to-rc.30 file delta is
  limited to `model/log.go`, `model/main.go`, `model/token_migration.go`, and
  `model/token_migration_test.go`.
- The rc.29 development branch is one release merge behind the rc.29 release
  branch. It is not a valid rc.30 base.
- Before this release worktree was created, no local or origin rc.30 release,
  development, or backup refs existed.

The first rc.29 baseline run in the new worktree reached every package and
reproduced only these two recorded Windows HTTP/2 GOAWAY timing failures:

- `TestUpstreamGetBody_HTTP2RetryAfterGracefulGoAway_PassThrough`
- `TestUpstreamGetBody_HTTP2CannotRetryWithoutGetBody`

Every other package in that run passed. These two cases must be checked again
on the final candidate and in Linux; a new failure outside this recorded pair
blocks the release.

## Release graph

Create `release/v1.0.0-rc.30` from the prior integrated rc.29 release. The
approved release design and plan may be committed on this branch before the
tag merge. Merge the rc.30 tag with a signed, non-fast-forward merge and retain
both parents. The expected merge has an rc.30 release tip descended from
`dccf03595850addfd0901523e5ace279ecb9da83` as its first parent and the exact
rc.30 tag object as its second parent.

The merge is expected to require manual reconciliation in:

- `model/log.go`, where upstream quota-statistics work overlaps the ZETA
  usage-log customization.
- `model/main.go`, where upstream token migration startup overlaps the ZETA
  PostgreSQL migrator wrapper.

Both behaviors must remain present after reconciliation. The upstream token
migration runs before the prefill migration and before GORM `AutoMigrate`.
The ZETA PostgreSQL dialector wrapper remains active during `AutoMigrate`.

After the integrated rc.30 release passes its initial checks, create
`dev/v1.0.0-rc.30` from that release. Any newly discovered code correction
uses a `fix/rc30/...` branch from the rc.30 development branch, enters the
development candidate through an ordinary signed history-preserving merge,
and returns to release through a signed `--no-ff` promotion.

## Carry-forward inventory

The following five rc.29 themes remain required on rc.30 because upstream
rc.30 does not provide equivalent code paths and regression coverage:

1. `feat/v1.0.0-rc.30/reasoning-model-support`
2. `feat/v1.0.0-rc.30/chatcompletions-responses-compat`
3. `fix/v1.0.0-rc.30/usage-logs-realtime-refresh`
4. `fix/v1.0.0-rc.30/channel-affinity-test-isolation`
5. `fix/v1.0.0-rc.30/postgres-automigrate-compat`

Each reusable branch is rebuilt from the clean rc.30 tag after the integrated
release is fixed. Existing rc.29 backup branches are evidence sources rather
than merge bases. Each rc.30 backup must contain only its named theme, retain
the rc.30 upstream files, and use signed commits with the required Codex
co-author trailer for newly created commits.

`CHANGELOG-ZETA.md` is release documentation rather than a runtime theme. It
travels through the integrated release ancestry and does not create a sixth
backup branch.

## Zeta changelog

Create `CHANGELOG-ZETA.md` at the repository root in Traditional Chinese. The
document has two layers:

1. A readable, newest-first summary of the ZETA changes first released in
   each version.
2. A complete commit ledger that makes every included OID traceable.

The verified historical set before rc.30 contains 120 distinct ZETA commit
OIDs:

| State | Commit count |
| --- | ---: |
| Included in `release/v1.0.0-rc.29` | 82 |
| Versioned backup-only commits | 35 |
| Unreleased rc.25 topic commits | 3 |

The 82 released commits are assigned to the version in which they first
entered the canonical ZETA release graph:

| Version | Released commits |
| --- | ---: |
| v1.0.0-rc.15 | 5 |
| v1.0.0-rc.18 | 1 |
| v1.0.0-rc.20 | 4 |
| v1.0.0-rc.21 | 21 |
| v1.0.0-rc.24 | 15 |
| v1.0.0-rc.25 | 9 |
| v1.0.0-rc.26 | 4 |
| v1.0.0-rc.27 | 15 |
| v1.0.0-rc.29 | 8 |

Versions without a ZETA release epoch do not receive empty headings. The
rc.30 section is added after its integration commits exist.

### Ledger fields

Every ledger row contains:

- 12-character SHA linked to the corresponding commit in this fork.
- Author name and author date in ISO 8601 form.
- Committer name and committer date in ISO 8601 form.
- Version or unreleased target.
- Class: Feature, Fix, Test, Docs, Governance, Integration,
  Carry snapshot, Backup rebuild, or Upstream copy.
- Original commit subject.
- Provenance showing the canonical released source, backup ref, cherry-pick
  trailer, or equivalent patch relationship.

Both dates are kept because carried commits can have an author date months
before their release committer date. The displayed version comes from Git
topology plus recorded release-branch epoch boundaries, never from a date
guess.

### Selection and assignment

Build the ledger from the union of local and origin `release/**`, `dev/**`,
`feat/**`, and `fix/**` refs. Exclude `main`, `pr/**`, and every exact object
reachable from all verified `refs/tags/**`, `refs/remotes/upstream/HEAD`, and
`refs/remotes/upstream/main`. Other upstream remote-tracking branches are
optional evidence and enter the exclusion set only when their exact names and
SHAs are recorded in the final reference block. Include
`refs/remotes/origin/main` in the upstream exclusion set only after a live
comparison proves that it matches the upstream mirror; it matched the rc.30
tag at
`27ff6a8767e728f879d52770c273d4f73214a430` during this review. Before final
generation, record the local `main`, live origin `main`, and target upstream
tag SHAs, then fetch only the required stale remote-tracking ref.

Assign a commit with this precedence:

1. An exact upstream object is excluded.
2. A canonical release ancestor enters that release version's `Released`
   ledger.
3. An object reachable only from `feat/v<version>/**` or
   `fix/v<version>/**` enters that version's `Backup rebuilds` ledger after
   verifying that the branch is based on the matching upstream tag.
4. An object reachable only from a development or unversioned topic ref
   enters `Unreleased`.
5. An unmatched object is marked `Unclassified` for manual review.

Walk the canonical release first-parent history from oldest to newest.
Creating a named `release/v<new-version>` branch starts that version's epoch,
including approved design and plan commits made before the upstream tag merge.
For this release, `e8e2d3469cfaa8df759957731a6ff3c2306393e2` belongs to rc.30.
The pre-rc.30 counts remain the fixed 82 released, 35 backup-only, and 3
unreleased OIDs. An upstream tag merge confirms the active epoch when its
non-first parent is the exact peeled upstream tag object. A normal topic merge
assigns the newly introduced fork-only subtree to the active epoch. The tag
merge itself is an Integration row, while its upstream parent and upstream
subtree are excluded.

Some historical release refs were pruned, so the following boundaries are
recorded release-branch evidence and override a pure tag-merge inference:

- rc.26 starts with `e595d819211dba119c8f9e9543a287e0c1122798` and includes
  `fd81ba893e649c3cb9cb946dec023e60a2fa7b5f`.
- rc.27 starts with `3084ef43418370ebc3ab9c266ad11105df2a207c` and includes
  `74e223c6cdff076ea55e611074f127e80e69ed63` plus
  `7add487c63e295287b4a0419ae5d983eac6040c9` before the tag merge.

These recorded boundaries are part of the reproducible assignment input and
must appear in the changelog reference block.

### Duplicate handling

The same OID appears once even when multiple refs contain it. Equivalent but
different OIDs remain visible as separate technical-history rows and point to
one canonical source. Equivalence evidence is accepted in this order:

1. An exact `cherry picked from` trailer.
2. A matching stable patch-id or `git cherry -v` minus result.
3. Manual range-diff and tree comparison for carry snapshots.

A shared subject, date, or author is not equivalence evidence. Merge commits
are never patch-id deduplicated. Documentation, governance, tests, and release
integration commits remain in the ledger.

The three commits on the unmerged rc.25 GLM topic remain under `Unreleased`.
Similar GLM behavior implemented later does not convert those OIDs into
released history.

Generate the final summary, ledger, and reference block together only after
the rc.30 release, development, and five backup refs exist. The finished
changelog records that generation timestamp, every source ref tip used for the
snapshot, and all full 40-character OIDs in a compact reference block so
future branch pruning cannot erase the audit trail. Any earlier draft snapshot
is regenerated rather than mixed with the final rc.30 inventory.

## PostgreSQL migration design

### Production rc.26 to rc.30 schema inventory

The implementation plan must repeat a tag-to-tag inspection of every
`model/` schema-related change between `v1.0.0-rc.26` and
`v1.0.0-rc.30`, then map each item to a fixture before the tag merge is
accepted. The current inspection identifies these schema changes:

- A new `login_encryption_keys` table with a unique slot index.
- A new `task_plugins` table with a composite unique key/version index and an
  active-state index.
- A nullable `prefill_groups.deleted_at` column plus replacement of legacy
  global name uniqueness with the partial `uk_prefill_name` index.
- Replacement of legacy `tokens.key` constraint-backed uniqueness with the
  standalone `idx_tokens_key` unique index.

The PostgreSQL prepared-statement setting, JSON scan normalization, task model
alias logic, and quota-statistics correction are recorded as database-adjacent
changes but do not add or alter schema objects. Any additional schema object
found by the repeated tag diff requires a named fixture before promotion.

### Compatibility behavior

rc.28 upgraded GORM to `v1.25.12`. On PostgreSQL, GORM can derive a default
`uni_<table>_<column>` constraint name and issue `DROP CONSTRAINT` even when
the actual single-column uniqueness comes from another named constraint or
index. That mismatch produces SQLSTATE 42704.

The rc.29 ZETA PostgreSQL migrator wrapper protects the generic path by
checking the derived constraint before dropping it. It also forwards the
PostgreSQL dialector configuration, savepoint operations, error translation,
and the 63-character identifier limit. Upstream rc.30 adds table-specific
prefill and token migrations but does not replace this generic protection, so
`model/postgres_migrator.go` and its regression tests remain required.

The rc.30 token migration recognizes only these legacy single-column unique
constraint names on `tokens.key`:

- `idx_tokens_key`
- `tokens_key_key`
- `uni_tokens_key`

It rejects unknown, deferrable, unvalidated, or `NULLS NOT DISTINCT`
constraints. When a known legacy constraint exists, it takes an
`ACCESS EXCLUSIVE` table lock in a transaction, removes the constraint, and
creates or validates the standalone unique index `idx_tokens_key`.

### Two-stage prefill rollback bridge

A real PostgreSQL 16 fixture on 2026-09-01 disproved the earlier assumption
that the rc.26 application could always read the final rc.30 prefill schema.
The fixture contained one soft-deleted prefill row and one active replacement
with the same name. The actual rc.26 commit
`c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7`, using GORM `v1.25.2`, attempted
to add the global constraint `idx_prefill_groups_name` during `InitDB()` and
failed with SQLSTATE 23505. An always-true CHECK constraint with that name was
also tested and failed with SQLSTATE 42710 because the old migrator still
issued `ADD CONSTRAINT ... UNIQUE (name)`. Neither model-tag equivalence nor a
same-name sentinel is rollback evidence.

Production currently has zero `prefill_groups` rows and both the legacy global
constraint `idx_prefill_groups_name` and partial index `uk_prefill_name`.
Migration therefore proceeds as an expand/contract sequence:

| State | Schema behavior | Previous successful application snapshot |
| --- | --- | --- |
| rc.26 | Global and partial name uniqueness coexist. | rc.26 |
| rc.30 Bridge | GORM `v1.25.12`; retain a recognized legacy global object and ensure the partial target exists. Never create the legacy object when it is absent. | rc.26 |
| rc.30 Contract | Remove the recognized legacy global object and keep the partial target, enabling reuse of a soft-deleted name. | rc.30 Bridge |

The Bridge migration validates the same object allowlist as the Contract
migration. If the recognized global object and valid partial target already
coexist, it performs no DDL and takes no table lock. If the partial target is
missing, it adds `deleted_at` when needed and creates or validates the partial
index while leaving the recognized global object in place. Repeated Bridge
startups are idempotent. When rolled back after Contract, Bridge sees no legacy
object, does not recreate one, and GORM `v1.25.12` keeps the partial target.

Contract restores the intended rc.30 behavior by dropping only the recognized
legacy global constraint or standalone index. Before either migration obtains
an `ACCESS EXCLUSIVE` lock, its transaction must set `lock_timeout` to 5
seconds and `statement_timeout` to 30 seconds with transaction-local
`set_config`. PostgreSQL 9.6 and 16 tests must read back both values and prove
that token and prefill lock contention ends with SQLSTATE 55P03. The JSONValue
scanner fallback must use `common.Marshal`; `encoding/json` remains permitted
only for the `json.RawMessage` type.

### Test matrix

Use disposable databases and schema clones. No migration test may target the
production database directly.

1. **Fresh database**: run migrations three times; verify stable catalog
   counts, token duplicate rejection, active prefill duplicate rejection, and
   reuse of a soft-deleted prefill name.
2. **rc.26 production-shaped database**: clone the real production schema
   after a read-only catalog inventory; verify row preservation, canonical
   token and prefill indexes, bigint quota columns, and an identical catalog
   snapshot on the second startup.
3. **rc.27-shaped database**: create a separate representative fixture from
   the recorded rc.27 application and schema path; verify the same row and
   catalog invariants without treating it as the live production source.
4. **rc.28-shaped database**: cover `tokens_key_key`, a constraint named
   `idx_tokens_key`, legacy prefill constraint and standalone-index forms,
   existing target indexes, non-conflicting composite and partial indexes,
   and the `login_encryption_keys` table.
5. **rc.29-shaped database**: preserve canonical prefill state, migrate the
   token legacy form, then run the generic named-constraint fixture through
   the ZETA dialector wrapper without SQLSTATE 42704.
6. **Failure atomicity**: insert an unsupported constraint or invalid target
   index; verify startup stops and the transaction leaves the schema and rows
   unchanged.
7. **Rollback compatibility**: start the live rc.26 release, the recorded
   rc.27 application, and the rc.29 ZETA release against separate Bridge
   clones; verify existing rows remain readable and token uniqueness remains
   enforced. The Bridge clone must retain both recognized prefill objects and
   must reject same-name replacement while a soft-deleted row exists. Run the
   actual rc.26 application twice and require unchanged rows and catalogs with
   no startup SQLSTATE. Then migrate an independent Bridge clone to Contract,
   create a same-name soft-deleted/active pair, and run the actual Bridge
   application twice; it must preserve two total rows, one active row,
   duplicate rejection, and an unchanged catalog. Running rc.26 directly on
   that Contract-only duplicate fixture is a required negative control and is
   expected to fail with SQLSTATE 23505; it is not an approved rollback path.
8. **Database range**: run SQLite, MySQL 5.7.8 or later, PostgreSQL 9.6, and a
   current PostgreSQL 16 instance. Run the common PostgreSQL fixture corpus on
   both versions. The `NULLS NOT DISTINCT` rejection fixture requires
   PostgreSQL 15 or later; on PostgreSQL 9.6, record that the syntax is
   unavailable rather than reporting a skipped common fixture. On PostgreSQL
   9.6, explicitly verify that `cardinality`, `to_regclass`, `indisready`, and
   `indexprs` return the values required by the inspection queries.
   PostgreSQL tests must execute with an actual `TEST_POSTGRES_DSN`; skipped
   common integration tests do not satisfy this matrix.

Before a production switch, run a read-only catalog inspection against the
database selected by the New-API service `SQL_DSN`, not the PostgreSQL
service's default database. The previously observed target was database
`newapi`, schema `public`; this is time-sensitive and must be rechecked without
printing connection credentials.

Inventory every single-column PostgreSQL UNIQUE constraint with its table,
column, name, definition, deferrable and validated flags, plus backing-index
unique, valid, ready, partial, and expression state. Any unsupported token or
prefill object blocks the switch until a separately approved migration plan
exists.

Deploy in a maintenance window after checking active long transactions and
blocking locks. The application-level transaction-local timeout settings are
mandatory in addition to that live preflight.

### Credentials and backup

A runtime PostgreSQL credential entered an internal diagnostic output during
the preflight. Its value must never be copied into the repository, logs, or
future commands. Rotate it through Zeabur before any further live catalog
inspection, update every dependent service atomically, and verify application
connectivity with redacted evidence.

Immediately before the production switch, create a production PostgreSQL
backup, record the backup identifier and timestamp outside the repository,
and document exact restoration commands without secret values. Verify the
backup with a restore drill into a disposable database before accepting it as
the production safety point.

## Verification gates

The final rc.30 candidate must pass all applicable current-repository gates:

- `git diff --check`
- Focused tests for all five customization themes.
- Focused PostgreSQL prefill, token, and generic migrator tests without skips.
- `go test -count=1 ./...`
- `go build ./...`
- `go vet ./...`
- `GOWORK=off go test ./...`, `go build ./...`, and `go vet ./...` from
  `relaykit/`.
- `bun run test`, `bun run typecheck`, `bun run lint`,
  `bun run format:check`, and `bun run build:check` from `web/`.
- Docker image build and container-local `/api/status` smoke test.
- Repeated local startup against SQLite, MySQL, and PostgreSQL fixtures.
- Authenticated event-driven usage-log regression triggered by a real
  `/v1/chat/completions` request while the local page is open.

The current tree contains a single `web/` frontend and no `web/classic`
package, so no obsolete classic-build command is added. The `test`,
`typecheck`, `lint`, `format:check`, and `build:check` scripts were verified in
`web/package.json`, and `relaykit/go.mod` was verified before these commands
were made mandatory.

The two recorded Windows HTTP/2 tests require a fresh final-candidate run and
a Linux run. If they fail only on Windows with the recorded socket-abort
symptoms while the unchanged rc.29 base reproduces them and Linux passes, the
release record may list the qualified baseline exception.

## Push and deployment order

1. Build and verify the Bridge candidate. Its PostgreSQL backup theme retains
   the legacy global prefill object, includes the timeout guard, and matches
   the integrated Bridge release.
2. Push and read back all five Bridge backup refs, then the Bridge development
   ref, and finally the still-unmonitored Bridge release ref.
3. Deploy Bridge to the isolated Dev project. Verify the full application
   gates, the Bridge schema fixture, runtime logs, metrics, health, version,
   and actual rc.26 rollback fixture.
4. Present the Bridge SHA, tests, backups, signatures, live catalog, credential
   rotation, fresh production backup, and restore drill. Accept only the exact
   phrase `確認發布`, then point Production at the Bridge release.
5. Verify Bridge build/runtime logs, metrics, health, allowlisted API paths,
   `/api/status`, both prefill objects, and the new previous-successful
   snapshot before preparing Contract.
6. Append the Contract change on the same rc.30 working theme. Verify that it
   drops only the recognized global object, preserves the partial target, and
   passes the actual Bridge-on-Contract rollback fixture. Append the same
   signed delta to the PostgreSQL backup theme and read it back before release.
7. Promote Contract locally, then regenerate `CHANGELOG-ZETA.md` after every
   Contract release, development, and backup tip exists. Its new snapshot
   upper bound is the signed Contract release merge; the final changelog
   commit excludes only itself from the ledger. Re-run the complete ledger,
   relative-link, archive-exclusion, and Markdown-fence validators before
   treating that documentation commit as the Contract candidate SHA.
8. Push the Contract backups and development ref, deploy Contract to Dev, and
   repeat the database, application, and runtime gates. Create and verify a
   new production backup and restore drill.
9. Because the release branch is now monitored, present the Contract SHA and
   evidence and accept a second exact `確認發布` before pushing Contract to the
   production release ref.
10. Verify Contract build/runtime logs, metrics, health, allowlisted paths,
   `/api/status`, removal of the legacy object, validity of `uk_prefill_name`,
   soft-deleted name reuse, and rollback to the Bridge snapshot.

No branch or tag is pushed to `upstream`. No custom tag is created. Old
version branches remain until rc.30 production is healthy and the owner has
approved an exact local-and-origin deletion inventory.

## Rollback

If Bridge deployment fails, inspect the newest deployment logs and metrics,
then roll the Zeabur application back to the previous successful rc.26
snapshot. Bridge retains global prefill uniqueness, so its schema cannot
contain the same-name soft-deleted/active pair that breaks rc.26 AutoMigrate.
The branch fallback remains origin `release/v1.0.0-rc.26` at
`c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7` until Contract is healthy.

If Contract deployment fails, roll back only to the previous successful
rc.30 Bridge snapshot. Bridge uses GORM `v1.25.12` and does not recreate the
legacy global object when it is absent, so it remains compatible after
Contract enables same-name reuse. Direct rc.26 rollback from Contract is
prohibited unless an immediate read-only preflight proves that no duplicate
name exists across active and soft-deleted rows. No destructive rename or
deletion may be performed merely to satisfy that preflight.

The token constraint-to-index migration is not reversed by an application
rollback. The rc.26, rc.27, and rc.29 clone tests therefore prove whether
prior applications can read the migrated schema. Restore the production
database only for demonstrated data or schema damage and only from the
verified backup created immediately before the switch.

The rc.26 branch remains until Contract is healthy. The Bridge and Contract
snapshots, exact SHAs, schema state, and allowed rollback edge must be recorded
before old-version branch pruning.

## Completion criteria

The work is complete only when all of the following are proven from current
state:

- rc.30 release and development refs contain the intended signed ancestry.
- All five rc.30 reusable backup refs exist, contain only their themes, and
  match the integrated release behavior.
- `CHANGELOG-ZETA.md` contains every released, backup-only, and unreleased ZETA
  OID once, with Git topology plus recorded release-branch epoch assignment
  and both Git dates. The earlier 142-OID result remains a historical base;
  the final Contract ledger includes the post-review design, Bridge, Contract,
  merge, and backup commits through its new snapshot upper bound.
- The full application, frontend, relaykit, container, and database gates have
  current results.
- PostgreSQL rc.26 production, rc.27, rc.28, rc.29, fresh-install,
  failure-atomicity, and rollback fixtures satisfy the documented invariants.
- Bridge retains global prefill uniqueness and passes actual rc.26 rollback;
  Contract removes it only after Bridge is the previous successful snapshot
  and passes actual Bridge rollback with a same-name soft-delete replacement.
- Both monitored release changes received separate exact `確認發布` phrases.
- The runtime PostgreSQL credential has been rotated and dependent services
  reconnect successfully without exposing secret values.
- Production has a verified backup and restore drill.
- Dev rc.30 deployment is healthy.
- Production remains on verified rc.26 until Bridge approval, then on verified
  Bridge until Contract approval.
- Final Production runs healthy rc.30 Contract and reports the rc.30 version
  through runtime and health evidence.
