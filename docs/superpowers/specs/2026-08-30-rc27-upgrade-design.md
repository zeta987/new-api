# v1.0.0-rc.27 Upgrade Design

## Context

The current self-use release and development refs both point to signed commit
`c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7`. Upstream tag
`v1.0.0-rc.27` resolves to signed commit
`eb48396d5fe97d27772d0cd5e3ca8aa5caa4f3e9` and contains seven commits after
upstream `v1.0.0-rc.26`.

The upstream delta changes 352 files. Its largest change replaces built-in
task adaptors with a sandboxed JavaScript plugin system, adds a persisted
`TaskPlugin` model, and adds administration, routing, billing, artifact, and
frontend support for those plugins. It also adds GLM `/v1/responses` support,
Ollama Responses and Claude passthrough, a tiered-billing time-rule fix, an
administrator unbinding fix, and a Docker development build fix.

The upstream release notice explicitly marks the plugin system as
experimental, warns that the architecture changed substantially, requires all
video-model prices to be configured again after upgrading, and does not
recommend rc.27 for production use. This upgrade therefore separates local
integration from any production cutover.

Source: <https://github.com/QuantumNous/new-api/releases/tag/v1.0.0-rc.27>

## Goals

1. Integrate upstream `v1.0.0-rc.27` into a new self-use release without
   losing any behavior already released in rc.26.
2. Preserve the four approved customization themes and rebuild their rc.27
   backup refs from the clean upstream tag.
3. Verify the new task-plugin architecture, additive database migration,
   billing changes, frontend, relaykit independence, signatures, and runtime.
4. Create `dev/v1.0.0-rc.27` only from the completed rc.27 release.
5. Keep production on rc.26 until video pricing is reconfigured, every release
   gate passes, and the owner enters the exact confirmation phrase.

## Non-goals

This upgrade does not implement the new generic GLM effort-alias feature. That
feature starts afterward from `dev/v1.0.0-rc.27` on its own topic branch.

The unreleased `feat/rc25/glm-effort-openai-openrouter` branch is reference
material only. None of its commits are merged or cherry-picked because its
OpenRouter and Responses behavior failed the later review.

This upgrade does not remove upstream branding, create custom release tags,
push to `upstream`, solve the Anubis challenge, delete older refs, or switch
Zeabur automatically.

## Workspace and branch topology

Integration occurs in the ignored linked worktree:

`D:\Data\Coding_Github\Projects\_AI\_Proxy\new-api\.local-tests\worktrees\rc27`

The worktree is on `release/v1.0.0-rc.27`, created from
`release/v1.0.0-rc.26` at `c0b9df91c`. The primary checkout stays on the
production-tracking rc.26 release and is not modified.

The branch order is:

1. Commit the approved rc.27 design documents on
   `release/v1.0.0-rc.27`.
2. Merge `v1.0.0-rc.27` into that branch with a signed, non-fast-forward
   merge.
3. Resolve overlaps by preserving both upstream rc.27 behavior and every
   released customization contract.
4. Run the complete rc.27 release gate.
5. Rebuild and verify these branches from the clean rc.27 tag:
   - `feat/v1.0.0-rc.27/reasoning-model-support`
   - `feat/v1.0.0-rc.27/chatcompletions-responses-compat`
   - `fix/v1.0.0-rc.27/usage-logs-realtime-refresh`
   - `fix/v1.0.0-rc.27/channel-affinity-test-isolation`
6. Create `dev/v1.0.0-rc.27` from the completed release.
7. Push and verify backup refs before the unmonitored rc.27 release ref.
8. Keep Zeabur on rc.26 until the separate production gate is satisfied.

## Conflict forecast

`git merge-tree --write-tree release/v1.0.0-rc.27 v1.0.0-rc.27` reports five
conflicted paths:

| Path | Required resolution |
| --- | --- |
| `AGENTS.md` | Keep all self-use release governance while accepting upstream repository guidance that does not weaken it. |
| `middleware/distributor.go` | Preserve custom model-candidate and token-limit behavior inside rc.27's new channel-constraint and task-plugin selection flow. |
| `model/ability.go` | Preserve exact-first model fallback and released constraints while adopting rc.27's unified `ChannelSatisfiesFilters` path. |
| `model/channel_cache.go` | Preserve the same candidate order and behavior as the database path while adopting rc.27's filter pipeline. |
| `setting/billing_setting/tiered_billing_test.go` | Keep existing billing-expression coverage and add upstream's time-rule regression coverage without duplicate assertions. |

The four rc.26 backup themes overlap the upstream delta as follows:

| Theme | Changed paths | Upstream overlaps |
| --- | ---: | ---: |
| reasoning model support | 59 | 9 |
| Chat Completions and Responses compatibility | 7 | 0 |
| usage-log realtime refresh | 11 | 3 |
| channel-affinity test isolation | 1 | 0 |

The reasoning overlaps are `controller/channel-test.go`,
`middleware/distributor.go`, `model/ability.go`, `model/channel_cache.go`,
`model/option.go`, `model/pricing.go`, `relay/channel/zhipu_4v/adaptor.go`, and
the tiered-billing implementation and test. Usage-log overlaps are
`model/log.go`, `router/api-router.go`, and the default frontend usage-log
provider.

## Channel-selection integration

rc.27 introduces `dto.ChannelConstraints` and centralizes channel validation
behind `model.ChannelSatisfiesFilters`. Conflict resolution uses that seam
instead of restoring separate checks in database selection, memory-cache
selection, channel affinity, and pinned-channel handling.

The released `ModelMatchCandidates` behavior remains authoritative: exact
configuration wins, GPT-5.6 validated wildcard configuration comes next, and
the normalized billing model is the final fallback. Both database and cache
selection must consume the same ordered candidates before applying rc.27
filters. Token model limits and affinity checks must use the same normalized
behavior.

During the upgrade itself, existing GLM aliases retain their rc.26 restriction
to Zhipu V4. Widening GLM aliases to OpenAI and OpenRouter channels belongs to
the later GLM topic branch, where it receives independent tests and review.

## Database and plugin transition

rc.27 adds `TaskPlugin` to GORM `AutoMigrate`. The resulting table contains a
surrogate key, unique `(key, version)` index, active/enabled state, source text,
source hash, timestamps, and remarks. This is an additive schema change, but it
still requires SQLite, MySQL, and PostgreSQL migration verification under the
repository release gate.

Before any monitored release push, the production PostgreSQL database must be
backed up and restoration steps recorded. The prior rc.26 quota-column proof
remains valid only after its live state is rechecked; it does not replace the
new `task_plugins` migration check.

Task plugins are enabled by default through `TASK_PLUGIN_ENABLED` and
`TASK_PLUGIN_OVERRIDE_ENABLED`. Local verification must cover startup with
factory plugins, persisted overrides, plugin disablement, channel binding,
protocol routing, task settlement, and restart persistence. Marketplace
network access is not required to prove built-in plugin operation.

## Video-price transition

The owner must inventory the currently configured video models and preserve
their effective prices before production cutover. After rc.27 is running in a
non-production verification environment, every video model used in production
must be configured in the new task-pricing system and compared against its
prior effective charge.

Missing, zeroed, or semantically different prices block production. The
release branch may be built and tested locally while this configuration work
is pending, but Zeabur remains on rc.26.

## Verification design

The local integration gate includes:

1. `git diff --check` and focused diffs against both rc.26 release and clean
   upstream rc.27.
2. Conflict-specific tests for model selection, channel constraints, token
   limits, affinity, tiered billing, Zhipu Responses, and usage-log refresh.
3. All new task-plugin, plugin protocol, artifact-access, task-pricing, and
   frontend tests introduced by rc.27.
4. `go test -count=1 ./...`, `go build ./...`, and `go vet ./...` after
   `web/dist` is built.
5. `GOWORK=off go test -count=1 ./...` and `GOWORK=off go build ./...` inside
   `relaykit/`.
6. From `web/`: `bun run lint`, `bun run format:check`, `bun run test`, and
   `bun run build:check`.
7. From `web/`: `bun run lint:plugins` and `bun run format:plugins:check`.
8. A local runtime smoke covering startup migration, `/api/status`, plugin
   registry initialization, and one built-in task-plugin discovery path.
9. Signed commit and merge verification for release, development, and every
   rebuilt backup ref.

The unchanged rc.26 baseline reproduces the Windows HTTP/2 graceful-GOAWAY
exception in
`TestUpstreamGetBody_HTTP2RetryAfterGracefulGoAway_PassThrough`: a five-run
focused baseline passed four times and failed once with `wsarecv`. Its sibling
`TestUpstreamGetBody_HTTP2CannotRetryWithoutGetBody` passed five focused runs.
The release gate may record the former only as the existing qualified Windows
exception when all touched-path tests pass and no new HTTP transport change is
present.

## Production gate and rollback

Before switching Zeabur, present the release SHA, signed merge proof, test and
build results, four backup refs, production database backup status,
`task_plugins` migration proof, and completed video-price comparison. The
monitored release push or branch switch waits for the exact phrase
`確認發布`.

If rc.27 fails after cutover, first inspect deployment logs and service
metrics, then use the previous successful Zeabur snapshot for immediate
application recovery. The additive `task_plugins` table remains compatible
with rc.26, but video pricing and task-plugin configuration are operational
state and must be restored or disabled separately.

## Acceptance criteria

The upgrade is locally complete when the signed rc.27 release contains all
seven upstream commits and all rc.26 customizations, the four rc.27 backup
branches contain only their themes, development points to the completed
release, every applicable gate passes or carries only the documented Windows
exception, and the worktree is clean.

Production completion additionally requires a verified PostgreSQL backup,
additive migration proof, complete video-price reconfiguration, healthy
Zeabur runtime evidence, and the exact production confirmation phrase.
