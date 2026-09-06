# rc.33 Kimi tools and responsive usage logs

The release combines native Kimi Formula execution with rate-aware background
usage-log refreshes. It stays on `release/v1.0.0-rc.33` and introduces no schema,
database-driver, dependency, or production channel-configuration changes.

## Included changes

- `0f01c5742305ab7f24cfadf0866f6b77e03d965c`: native Moonshot Chat executes
  explicitly enabled search, webpage, and Python Formula tools. Complete
  assistant reasoning and tool IDs survive the loop. Token billing is evaluated
  per round; paid tools are reserved and successful work is settled on partial
  failure. Streaming returns buffered final output. Execution and response sizes
  are bounded, with an overall request deadline.
- `4484c00ad7c11c933a02a55e5c39d2ea6f527421`: usage logs coalesce automatic
  refreshes to one list/stat pair per ten seconds, honor Retry-After, and retain
  recovery after transient errors. Table interactions and a selected details
  snapshot remain available during background updates. A production QueryCache
  HTTP 500 regression was fixed before integration; other queries retain their
  existing error-page navigation.
- `29ca685d9f99e1a52583ff977b57e9098833d64e`: explicitly type the navigation
  callback used by the new recovery regression test.

## Verification

Commands ran from the repository root, or `web/` for Bun commands.

- `go test ./... -count=1`: all packages passed except two Windows HTTP/2 cases,
  `TestUpstreamGetBody_HTTP2RetryAfterGracefulGoAway_PassThrough` and
  `TestUpstreamGetBody_HTTP2CannotRetryWithoutGetBody`. Both failures were
  reproduced on the unchanged `bf5deb488d1f17515b911b386e131ef4e468353a` base.
- `go build ./...`, `go vet ./...`, and `git diff --check`: passed.
- `bun run test -- src/features/usage-logs`: 12 files, 40 tests passed.
- `bun run build:check`: TypeScript checking and the production build passed.
- Changed-file Oxlint and Oxfmt checks: all 14 TypeScript/TSX files passed.
  Full `bun run lint` reports 359 existing errors; full `bun run format:check`
  reports 26 existing files. Both sets match the unchanged frontend base, with
  no new errors or files introduced by this candidate.
- `pwsh -NoProfile -File .local-tests/worktrees/kimi-tool-loop/.local-tests/run-kimi-e2e.ps1`:
  the integrated gateway passed 12 HTTP cases each on SQLite 3.50.4, MySQL
  5.7.44, and PostgreSQL 9.6.24. Assertions include per-round charges, exact
  wallet/log agreement, partial failure, paid tools on free models,
  insufficient-token rejection before tool execution, and upstream request IDs.
- Authenticated browser integration: three native Kimi API requests returned
  their final answers and consumed 8250 quota. The already-open usage page
  advanced from 152 to 155 records through SSE-triggered refreshes while the
  original details snapshot remained open. Controls remained enabled afterward.
  A stale development bundle was identified and hard-reloaded before this check.

Provider protocol tests use a local deterministic Formula fixture. A live Kimi
provider key was not used. Production channel overrides and tool prices remain
operator-managed; see [the Kimi guide](../channel/kimi-formula-tools.md).

## Carry-forward inventory

- Append Kimi Formula support to
  `feat/v1.0.0-rc.33/reasoning-model-support`, alongside the existing Moonshot
  model and shared relay/pricing customizations.
- Append background refresh, details retention, and HTTP 500 recovery to
  `fix/v1.0.0-rc.33/usage-logs-realtime-refresh`.

Both backup deltas preserve their existing history and contain only their
respective topic files. They must be pushed and verified before the monitored
release branch. Other rc.33 backup refs are retained unchanged.
