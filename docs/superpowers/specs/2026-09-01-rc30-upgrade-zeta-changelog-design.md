# rc.30 Upgrade and Zeta Changelog Design

## Purpose

This change promotes the integrated self-use release from
`v1.0.0-rc.29` to `v1.0.0-rc.30`, preserves every active customization,
verifies the PostgreSQL migrations that appeared in rc.28 through rc.30,
and adds a repository-level `CHANGELOG-ZETA.md` covering ZETA-authored
history from its first released customization through rc.30.

Production currently remains on rc.27 according to the owner. Live Zeabur
state must be checked again before any branch switch. The production switch
requires the exact confirmation phrase `確認發布` after the release evidence
and database backup status have been presented.

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
topology, never from a date guess.

### Selection and assignment

Build the ledger from the union of local and origin `release/**`, `dev/**`,
`feat/**`, and `fix/**` refs. Exclude `main`, `pr/**`, and every exact object
reachable from verified upstream refs or upstream tags.

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
An upstream tag merge begins a new version epoch when its non-first parent is
the exact peeled upstream tag object. A normal topic merge assigns the newly
introduced fork-only subtree to the active epoch. The tag merge itself is an
Integration row, while its upstream parent and upstream subtree are excluded.

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

The finished changelog records its generation timestamp, every source ref tip
used for the snapshot, and all full 40-character OIDs in a compact reference
block so future branch pruning cannot erase the audit trail.

## PostgreSQL migration design

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

### Test matrix

Use disposable databases and schema clones. No migration test may target the
production database directly.

1. **Fresh database**: run migrations three times; verify stable catalog
   counts, token duplicate rejection, active prefill duplicate rejection, and
   reuse of a soft-deleted prefill name.
2. **rc.27-shaped database**: clone the real production schema after a
   read-only catalog inventory; verify row preservation, canonical token and
   prefill indexes, bigint quota columns, and an identical catalog snapshot on
   the second startup.
3. **rc.28-shaped database**: cover `tokens_key_key`, a constraint named
   `idx_tokens_key`, legacy prefill constraint and standalone-index forms,
   existing target indexes, non-conflicting composite and partial indexes,
   and the `login_encryption_keys` table.
4. **rc.29-shaped database**: preserve canonical prefill state, migrate the
   token legacy form, then run the generic named-constraint fixture through
   the ZETA dialector wrapper without SQLSTATE 42704.
5. **Failure atomicity**: insert an unsupported constraint or invalid target
   index; verify startup stops and the transaction leaves the schema and rows
   unchanged.
6. **Rollback compatibility**: start the rc.29 ZETA release and rc.27
   application against separate rc.30-migrated clones; verify existing rows
   remain readable and token uniqueness remains enforced.
7. **Database range**: run SQLite, MySQL 5.7.8 or later, PostgreSQL 9.6, and a
   current PostgreSQL 16 instance. PostgreSQL tests must execute with an actual
   `TEST_POSTGRES_DSN`; skipped integration tests do not satisfy this matrix.

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

Set bounded PostgreSQL `lock_timeout` and `statement_timeout` values for the
deployment drill. Deploy in a maintenance window after checking active long
transactions and blocking locks.

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
package, so no obsolete classic-build command is added.

The two recorded Windows HTTP/2 tests require a fresh final-candidate run and
a Linux run. If they fail only on Windows with the recorded socket-abort
symptoms while the unchanged rc.29 base reproduces them and Linux passes, the
release record may list the qualified baseline exception.

## Push and deployment order

1. Verify the integrated release, development candidate, five backup diffs,
   commit signatures, and required trailers locally.
2. Push and read back the five rc.30 backup refs.
3. Push and read back `dev/v1.0.0-rc.30`.
4. Push and read back the unmonitored `release/v1.0.0-rc.30` last.
5. Point the isolated Dev Zeabur project at the rc.30 development branch and
   verify build logs, runtime logs, CPU, memory, container-local health, and
   version headers.
6. Present the release SHA, test record, five backup refs, signature results,
   production catalog inventory, credential-rotation status, backup status,
   and restore-drill result.
7. Accept only the exact owner phrase `確認發布` as production-switch approval.
8. Point the production Zeabur project at the rc.30 release branch and inspect
   build/runtime logs, metrics, container-local health, public API allowlisted
   paths, and `/api/status`.

No branch or tag is pushed to `upstream`. No custom tag is created. Old
version branches remain until rc.30 production is healthy and the owner has
approved an exact local-and-origin deletion inventory.

## Rollback

If the rc.30 application deployment fails, inspect the newest deployment logs
and metrics, then roll the Zeabur application back to the previous successful
snapshot while keeping the current release branch available for repair.

The token constraint-to-index migration is not reversed by an application
rollback. The rc.27 and rc.29 clone tests therefore prove whether the prior
application can read the migrated schema. Restore the production database
only for demonstrated data or schema damage and only from the verified backup
created immediately before the switch.

## Completion criteria

The work is complete only when all of the following are proven from current
state:

- rc.30 release and development refs contain the intended signed ancestry.
- All five rc.30 reusable backup refs exist, contain only their themes, and
  match the integrated release behavior.
- `CHANGELOG-ZETA.md` contains every released, backup-only, and unreleased ZETA
  OID once, with topology-based version assignment and both Git dates.
- The full application, frontend, relaykit, container, and database gates have
  current results.
- PostgreSQL rc.27, rc.28, rc.29, fresh-install, failure-atomicity, and
  rollback fixtures satisfy the documented invariants.
- The runtime PostgreSQL credential has been rotated and dependent services
  reconnect successfully without exposing secret values.
- Production has a verified backup and restore drill.
- Dev rc.30 deployment is healthy.
- Production remains on rc.27 until the exact phrase `確認發布` is received.
- After that approval, production rc.30 deployment is healthy and reports the
  rc.30 version through runtime and health evidence.
