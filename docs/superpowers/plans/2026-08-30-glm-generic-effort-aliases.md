# Generic GLM Effort Aliases Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Accept validated `glm-<base>-<effort>` aliases without per-model parser changes and route them correctly through Zhipu V4, OpenAI-compatible Chat Completions, and OpenRouter configuration.

**Architecture:** One independently buildable relaykit parser owns GLM suffix syntax. Model matching, channel constraints, pricing, and mapping reuse the parsed base before provider adaptors run. Zhipu V4 and OpenAI Chat write native effort fields in code; OpenRouter receives only provider-qualified model mapping in code and uses administrator parameter overrides for `reasoning.effort`.

**Tech Stack:** Go 1.22+, Gin, GORM v2, `testify`, relaykit, New API parameter overrides

**Spec:** `docs/superpowers/specs/2026-08-30-glm-generic-effort-aliases-design.md`

## Global Constraints

- Start from completed `dev/v1.0.0-rc.27` on `feat/rc27/glm-generic-effort-suffixes`.
- Parse only lowercase names beginning with `glm-` and ending in one recognized final effort segment.
- Recognized efforts are exactly `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max`.
- Reject malformed bases such as `glm-low` and `glm--low`; leave unknown suffixes unchanged.
- Preserve exact alias configuration before wildcard and base fallback.
- GLM Chat aliases may select only Zhipu V4, OpenAI, or OpenRouter channels;
  GLM Responses aliases may select only Zhipu V4.
- OpenAI third-party support is Chat Completions only and uses the ordinary `/v1/chat/completions` path.
- OpenRouter gets no GLM-specific adaptor conversion; model redirection uses model mapping and effort selection uses parameter overrides.
- Use `common.Unmarshal` for root-module mapping JSON and keep `relaykit/` independent of root packages.
- Every production change follows RED, GREEN, REFACTOR and ends in a signed commit with the exact Codex trailer.

---

### Task 1: Replace the per-model GLM suffix table with syntax parsing

**Files:**
- Modify: `relaykit/relayconvert/reasoning/suffix.go`
- Modify: `relaykit/relayconvert/reasoning/suffix_test.go`

**Interfaces:**
- Consumes: a client model name string
- Produces: `ParseGLMReasoningEffortSuffix(modelName string) (baseModel string, effort string, ok bool)` and `IsGLMReasoningEffortModel(modelName string) bool`

- [ ] **Step 1: Write failing table tests**

Add deterministic cases asserting:

```go
tests := []struct {
    model, base, effort string
    ok                  bool
}{
    {"glm-5.2-none", "glm-5.2", "none", true},
    {"glm-5.3-high", "glm-5.3", "high", true},
    {"glm-5.3-flash-max", "glm-5.3-flash", "max", true},
    {"glm-future-model-xhigh", "glm-future-model", "xhigh", true},
    {"glm-5.3-flash", "glm-5.3-flash", "", false},
    {"glm-5.3-flash-fast", "glm-5.3-flash-fast", "", false},
    {"glm-low", "glm-low", "", false},
    {"glm--low", "glm--low", "", false},
    {"GLM-5.3-high", "GLM-5.3-high", "", false},
    {"custom-glm-5.3-high", "custom-glm-5.3-high", "", false},
}
```

Also assert `IsGLMReasoningEffortModel("glm-5.3-flash")` is true and malformed,
uppercase, or suffixed aliases are false. Assert the predicate and parser never
both report true for the same input.

- [ ] **Step 2: Run the relaykit parser test and confirm RED**

Run from `relaykit/`:

```powershell
$env:GOWORK='off'; go test -count=1 ./relayconvert/reasoning
```

Expected: future-shaped base and malformed-name cases fail under the exact-model table.

- [ ] **Step 3: Implement minimal lexical parsing**

Remove the GLM base-to-level map. Split the final hyphen segment, validate it
against the seven-value effort set, then require a nonempty lowercase GLM base
whose portion after `glm-` is neither empty nor hyphen-prefixed and whose own
final segment is not an effort. Return the original model unchanged on every
failure. Make `IsGLMReasoningEffortModel` test only that base predicate, so
valid aliases return false.

- [ ] **Step 4: Run parser tests and independent relaykit gate**

Run:

```powershell
$env:GOWORK='off'; go test -count=1 ./relayconvert/reasoning
$env:GOWORK='off'; go test -count=1 ./...
$env:GOWORK='off'; go build ./...
```

Expected: all commands pass.

- [ ] **Step 5: Commit the parser contract**

Commit the two files with signed message `feat: parse generic GLM effort aliases` and the exact Codex trailer.

### Task 2: Normalize matching, pricing, and allowed channel types

**Files:**
- Modify: `model/model_match.go`
- Modify: `dto/channel_constraints.go`
- Modify: `model/channel_constraint.go`
- Modify: `middleware/distributor.go`
- Modify: `setting/ratio_setting/model_ratio.go`
- Test: `model/model_match_test.go`
- Test: `model/channel_constraint_test.go`
- Test: `model/channel_glm_effort_routing_test.go`
- Test: `setting/ratio_setting/glm_reasoning_effort_test.go`

**Interfaces:**
- Consumes: the Task 1 parser and rc.27 `dto.ChannelConstraints`
- Produces: exact-before-base candidates, `FilterAllowedChannelTypes`, and base-price fallback

- [ ] **Step 1: Write failing candidate and price tests**

Assert `ModelMatchCandidates("glm-5.3-flash-high")` returns exact alias followed by `glm-5.3-flash`, exact alias pricing wins when configured, and the base price is used otherwise. Include an unrelated `custom-glm-*` case that never normalizes.

- [ ] **Step 2: Write failing filter tests**

Add `FilterAllowedChannelTypes` cases for Zhipu V4, OpenAI, OpenRouter, Zhipu
V3, and an unrelated type. Include a present filter with a nil channel and a
channel ID that cannot be resolved; both must fail closed and report the new
filter kind. Add format cases proving Chat allows all three supported types,
Responses allows only Zhipu V4, and Claude, Gemini, embeddings, and images
allow none.

- [ ] **Step 3: Run the focused tests and confirm RED**

Run:

```powershell
go test -count=1 ./model ./setting/ratio_setting ./middleware
```

Expected: generic base fallback and allowed-channel filter cases fail before implementation.

- [ ] **Step 4: Implement candidate, pricing, and filter behavior**

Append the parsed GLM base after exact and wildcard candidates. Add
`FilterAllowedChannelTypes` plus `AllowedChannelTypes []int`, insert the new
kind into `filterEvalOrder`, and evaluate it through
`ChannelSatisfiesFilters`. In the distributor, attach `ChannelTypeZhipu_v4`,
`ChannelTypeOpenAI`, and `ChannelTypeOpenRouter` for Chat aliases; attach only
`ChannelTypeZhipu_v4` for Responses; reject every other relay format before
selection.

- [ ] **Step 5: Prove every selection path shares the filter**

Tests must cover database selection, memory cache, pinned channel, and affinity. Expected behavior is identical candidate order in each path; Zhipu V3 exact aliases remain ineligible while an allowed base-configured channel is selected.

- [ ] **Step 6: Run focused tests and commit**

Run `go test -count=1 ./model ./setting/ratio_setting ./middleware`; then commit with signed message `feat: route generic GLM effort aliases` and the exact trailer.

### Task 3: Fall back from exact alias mapping to the GLM base key

**Files:**
- Modify: `relay/helper/model_mapped.go`
- Test: `relay/helper/model_mapped_test.go`

**Interfaces:**
- Consumes: `RelayInfo.OriginModelName`, channel `model_mapping`, and Task 1 parser
- Produces: mapped `RelayInfo.UpstreamModelName` while preserving the original alias

- [ ] **Step 1: Write failing mapping tests**

Cover these exact mappings:

```json
{
  "glm-5.3-flash": "z-ai/glm-5.3-flash",
  "glm-5.3-flash-max": "vendor/special-max"
}
```

Assert `glm-5.3-flash-high` falls back through the base key, `glm-5.3-flash-max` uses the exact key, `OriginModelName` remains unchanged, self-maps retain existing semantics, and multi-key cycles still return `model_mapping_contains_cycle`.

- [ ] **Step 2: Run helper tests and confirm RED**

Run `go test -count=1 ./relay/helper`; expected: base-key fallback case fails.

- [ ] **Step 3: Implement exact-first mapping fallback**

Decode with `common.Unmarshal`. Start the existing chain at the exact alias when it has a nonempty mapping; otherwise parse a validated GLM alias and start at its base key. Keep the original alias in `OriginModelName`, update only `UpstreamModelName`, and preserve the current visited-model cycle checks.

- [ ] **Step 4: Run helper tests and commit**

Run `go test -count=1 ./relay/helper`; then commit with signed message `feat: map GLM aliases through base models` and the exact trailer.

### Task 4: Support native Zhipu V4 Chat and Responses effort fields

**Files:**
- Modify: `relay/channel/zhipu_4v/adaptor.go`
- Modify: `relay/channel/zhipu_4v/constants.go`
- Test: `relay/channel/zhipu_4v/glm_reasoning_effort_test.go`
- Test: `relay/channel/zhipu_4v/responses_test.go`

**Interfaces:**
- Consumes: parsed alias metadata, `GeneralOpenAIRequest`, and rc.27 `OpenAIResponsesRequest`
- Produces: base model with top-level Chat `reasoning_effort` or Responses `reasoning.effort`

- [ ] **Step 1: Write failing GLM-5.3-Flash discovery and Chat tests**

Assert the model list includes `glm-5.3-flash` plus `-low`, `-high`, and `-max`. For Chat, assert suffix effort overrides a conflicting body effort, bare base preserves explicit effort, an absent bare effort stays absent, and applied effort updates `RelayInfo.ReasoningEffort`.

- [ ] **Step 2: Write failing Responses tests**

For `glm-5.3-flash-high`, assert the serialized request model is `glm-5.3-flash`, `reasoning.effort` is `high`, a conflicting input is replaced, and the URL remains Zhipu V4's rc.27 `/api/v1/responses` endpoint.

- [ ] **Step 3: Run Zhipu tests and confirm RED**

Run `go test -count=1 ./relay/channel/zhipu_4v`; expected: generic Flash and Responses effort cases fail.

- [ ] **Step 4: Implement both formats**

Reuse the same parser and origin fallback in Chat and Responses conversions. Set `ReasoningEffort` metadata only after writing the upstream field. Do not construct a Zhipu `/api/paas/v4` path for Responses.

- [ ] **Step 5: Run Zhipu tests and commit**

Run `go test -count=1 ./relay/channel/zhipu_4v`; then commit with signed message `feat: add GLM effort to Zhipu requests` and the exact trailer.

### Task 5: Support standard OpenAI-compatible Chat relays

**Files:**
- Modify: `relay/channel/openai/adaptor.go`
- Test: `relay/channel/openai/glm_reasoning_effort_test.go`

**Interfaces:**
- Consumes: selected OpenAI channel, mapped GLM base, and original validated alias
- Produces: ordinary OpenAI Chat payload with top-level `reasoning_effort`

- [ ] **Step 1: Write failing Chat-only tests**

Assert `glm-5.3-flash-low` sent through an OpenAI channel serializes as:

```json
{"model":"glm-5.3-flash","reasoning_effort":"low"}
```

Assert suffix effort overrides a conflicting body value, a bare GLM model
preserves an explicit body effort, non-GLM requests remain unchanged, and the
request URL is the configured standard `/v1/chat/completions` path without
`/api/paas/v4`. Add a distributor-level negative test proving an OpenAI
`/v1/responses` request with a GLM alias cannot select the channel or silently
lose effort.

- [ ] **Step 2: Run OpenAI adaptor tests and confirm RED**

Run `go test -count=1 ./relay/channel/openai`; expected: validated GLM suffix is not yet converted outside OpenAI's native model families.

- [ ] **Step 3: Implement the narrow OpenAI Chat conversion**

In Chat conversion only, recover the validated alias from origin metadata after base mapping, write the base model and top-level `reasoning_effort`, and call `info.SetReasoningEffort`. Leave OpenAI Responses behavior untouched.

- [ ] **Step 4: Run tests and commit**

Run `go test -count=1 ./relay/channel/openai`; then commit with signed message `feat: relay GLM effort through OpenAI chat` and the exact trailer.

### Task 6: Verify OpenRouter mapping plus parameter overrides

**Files:**
- Modify: `relay/common/override_test.go`
- Test: `relay/helper/model_mapped_test.go`
- Test: `relay/channel/openrouter/glm_effort_config_test.go`
- Modify: `docs/superpowers/specs/2026-08-30-glm-generic-effort-aliases-design.md` only when the tested administration JSON needs correction

**Interfaces:**
- Consumes: model mapping from Task 3 and the seven-operation override JSON in the spec
- Produces: provider model `z-ai/glm-5.3-flash` and final `reasoning: {effort: ...}` without adaptor conversion

- [ ] **Step 1: Add a configuration-level integration test**

For `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max`, set
`OriginModelName` to the corresponding `glm-5.3-flash-` alias, map the base to
`z-ai/glm-5.3-flash`, apply the spec's override operations, and assert the
final JSON contains the mapped model and exactly one reasoning field with the
selected effort.

- [ ] **Step 2: Add conflict and metadata cases**

Start with client `reasoning: {"enabled":false,"max_tokens":0}` and assert mode `set` replaces the entire object with `{"effort":"high"}`. Assert `original_model` conditions see the unmapped alias and post-override synchronization records `RelayInfo.ReasoningEffort`.

- [ ] **Step 3: Run tests and confirm behavior**

Run:

```powershell
go test -count=1 ./relay/common ./relay/helper ./relay/channel/openrouter
```

Expected: all seven efforts, whole-object replacement, mapping, and metadata tests pass without any OpenRouter adaptor modification.

- [ ] **Step 4: Commit the configuration contract**

Commit tests and any corrected spec JSON with signed message `test: cover OpenRouter GLM effort config` and the exact trailer.

### Task 7: Complete verification, review, integration, and backup

**Files:**
- All files changed by Tasks 1 through 6
- Branch: `feat/rc27/glm-generic-effort-suffixes`
- Branch: `dev/v1.0.0-rc.27`
- Branch: `release/v1.0.0-rc.27`
- Branch: `feat/v1.0.0-rc.27/reasoning-model-support`

**Interfaces:**
- Consumes: all focused commits
- Produces: verified feature in development and release, with reasoning-theme backup updated before any production action

- [ ] **Step 1: Run formatting and focused regression gates**

Run:

```powershell
gofmt -w relaykit/relayconvert/reasoning/suffix.go relaykit/relayconvert/reasoning/suffix_test.go model/model_match.go dto/channel_constraints.go model/channel_constraint.go middleware/distributor.go relay/helper/model_mapped.go relay/channel/zhipu_4v/adaptor.go relay/channel/openai/adaptor.go
git diff --check
go test -count=1 ./middleware ./model ./setting/ratio_setting ./relay/helper ./relay/common ./relay/channel/zhipu_4v ./relay/channel/openai ./relay/channel/openrouter
```

Expected: no formatting diff remains and all focused packages pass.

- [ ] **Step 2: Run complete backend and relaykit gates**

Run from root:

```powershell
go test -count=1 ./...
go build ./...
go vet ./...
```

Run from `relaykit/`:

```powershell
$env:GOWORK='off'; go test -count=1 ./...
$env:GOWORK='off'; go build ./...
```

Expected: all commands pass, subject only to the unchanged and separately reproduced Windows graceful-GOAWAY exception documented by the rc.27 upgrade spec.

- [ ] **Step 3: Obtain a read-only code review and address validated findings**

Review the complete topic diff against `dev/v1.0.0-rc.27`, paying particular attention to exact-alias precedence, nil channels, retries, mapping cycles, Responses metadata, and absence of OpenRouter adaptor changes. Apply only findings confirmed against code and tests, then rerun the affected focused gate.

- [ ] **Step 4: Merge the topic into development and promote the indivisible candidate**

Merge the signed topic commits into `dev/v1.0.0-rc.27`, rerun the release gate, then merge development into release with `git merge --no-ff -S`. Verify the merge signature and exact Codex trailer.

- [ ] **Step 5: Append the released delta to the reasoning backup**

Apply the feature's released commits to `feat/v1.0.0-rc.27/reasoning-model-support`, rerun its focused tests, and verify its diff from clean `v1.0.0-rc.27` contains only reasoning-model support.

- [ ] **Step 6: Stop at the production confirmation gate**

Present the release SHA, all test results, signed backup ref, PostgreSQL backup state, `task_plugins` migration evidence, and video-price comparison. No monitored release push or Zeabur switch occurs until the owner enters exactly `確認發布`.
