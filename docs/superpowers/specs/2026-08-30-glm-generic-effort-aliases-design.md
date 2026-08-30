# Generic GLM Reasoning Effort Aliases

## Context

The rc.26 self-use release supports exact aliases for GLM-5.2 and GLM-5.3 on
Zhipu V4 only. Each supported base model and effort set is hard-coded in
`relaykit/relayconvert/reasoning/suffix.go`, and validated aliases are rejected
for OpenAI and OpenRouter channels before model mapping or parameter overrides
run.

GLM-5.3-Flash introduces another reasoning-capable base model with `low`,
`high`, and `max` effort levels. Repeating the base-model table, model-list
entries, selection rules, pricing normalization, and adaptor code for every
future GLM model would continue the maintenance pattern this change is meant
to remove.

Two prior implementation attempts are reference material only. The rc.25
branch `feat/rc25/glm-effort-openai-openrouter` widened channel selection and
added OpenAI adaptor translation, but later review found that its OpenRouter
reasoning object could defeat suffix intent and that Responses requests could
lose the recorded effort. This design replaces that provider-wide conversion
with a shared alias seam and three deliberately different channel strategies.

Official references:

- GLM-5.3-Flash model and parameters:
  <https://docs.bigmodel.cn/cn/guide/models/vlm/glm-5.3-flash>
- GLM Chat Completions `reasoning_effort`:
  <https://docs.bigmodel.cn/api-reference/%E6%A8%A1%E5%9E%8B-api/%E5%AF%B9%E8%AF%9D%E8%A1%A5%E5%85%A8>
- OpenRouter model ID `z-ai/glm-5.3-flash`:
  <https://openrouter.ai/z-ai/glm-5.3-flash>
- OpenRouter reasoning object:
  <https://openrouter.ai/docs/guides/best-practices/reasoning-tokens>
- New API channel parameter overrides:
  <https://docs.newapi.pro/zh/docs/guide/feature-guide/admin/channel#%E5%8F%82%E6%95%B0%E8%A6%86%E7%9B%96%E7%B3%BB%E7%BB%9F>

## Goals

1. Accept validated effort suffixes for reasoning-capable GLM base names
   without adding each base model to a parser table.
2. Preserve exact alias configuration before falling back to a base-model
   channel and base-model price.
3. Support Zhipu V4 natively, standard OpenAI-compatible Chat Completions
   relays in code, and OpenRouter Chat Completions through model mapping plus
   channel parameter overrides.
4. Add `glm-5.3-flash`, `glm-5.3-flash-low`,
   `glm-5.3-flash-high`, and `glm-5.3-flash-max` to the current Zhipu V4
   discovery list.
5. Keep the applied effort visible in request-correlated logs.

## Non-goals

The standard OpenAI third-party relay path covers `/v1/chat/completions` only.
It does not construct or use Zhipu-specific `/api/paas/v4` URLs and does not
add OpenAI Responses conversion.

The OpenRouter path covers Chat Completions only and does not add GLM-specific
effort conversion to the OpenRouter adaptor. It depends on administrator-owned
model mapping and parameter-override configuration. OpenAI Responses, Claude,
Gemini, embeddings, images, and other relay formats do not gain GLM alias
support through either the OpenAI or OpenRouter channel type.

This feature does not guess arbitrary trailing words as efforts, modify
sampling parameters, force `thinking.type`, change default effort when no
suffix is present, add pricing values, alter response conversion, or change
database schema.

## Shared alias module

The seam remains the independently buildable
`relaykit/relayconvert/reasoning` module. Its public interface stays small:

```go
func ParseGLMReasoningEffortSuffix(modelName string) (baseModel string, effort string, ok bool)
func IsGLMReasoningEffortModel(modelName string) bool
```

The implementation no longer maps exact base models to effort lists. It first
removes one recognized final effort suffix, then validates the remaining name
with a lexical base predicate: it must start with lowercase `glm-`, contain at
least one character after that prefix, not begin its remainder with another
hyphen, and not itself end in a recognized effort segment.
`IsGLMReasoningEffortModel` means exactly "this is a syntactically valid GLM
base model name" and exposes that predicate. It returns false for validated
aliases. A successful `ParseGLMReasoningEffortSuffix` result and a true
`IsGLMReasoningEffortModel` result are therefore mutually exclusive for the
same input. The recognized vocabulary is:

```text
none, minimal, low, medium, high, xhigh, max
```

Only the final segment is interpreted. Examples:

| Requested model | Base model | Effort | Valid alias |
| --- | --- | --- | --- |
| `glm-5.3-flash-low` | `glm-5.3-flash` | `low` | yes |
| `glm-5.3-flash-high` | `glm-5.3-flash` | `high` | yes |
| `glm-5.3-flash-max` | `glm-5.3-flash` | `max` | yes |
| `glm-5.3-flash` | unchanged | empty | no |
| `glm-5.3-flash-fast` | unchanged | empty | no |
| `glm-low` | unchanged | empty | no |
| `glm--low` | unchanged | empty | no |
| `custom-glm-5.3-high` | unchanged | empty | no |
| `GLM-5.3-high` | unchanged | empty | no |

The parser validates syntax, not a provider's per-model effort matrix. This is
what removes the per-base code update. Provider documentation and upstream
validation remain authoritative for whether a particular base model accepts a
particular recognized effort. Current first-class regression tests cover
GLM-5.2's released aliases and GLM-5.3 plus GLM-5.3-Flash with `low`, `high`,
and `max`. Predicate tests separately prove that bare bases are true while
valid aliases, `glm-low`, and `glm--low` are false.

## Matching, pricing, and channel selection

The model candidate order is:

1. the exact requested alias;
2. any validated model-family wildcard already supported by the repository;
3. the normalized GLM base model or other normalized billing model.

An administrator can therefore configure only `glm-5.3-flash` and accept all
validated effort aliases, while an explicitly configured
`glm-5.3-flash-max` retains exact precedence. Price lookup follows the same
exact-before-base behavior.

rc.27 centralizes channel constraints through
`model.ChannelSatisfiesFilters`. Extend its interface with the generic filter
kind `FilterAllowedChannelTypes` and an `AllowedChannelTypes []int` field on
`dto.ChannelFilter`. After the distributor parses a validated GLM effort
alias, it adds one such filter containing the three supported channel types.
Add the new kind to `filterEvalOrder` so every selection path evaluates it in
the same position. Evaluation fails closed when the filter is present but the
candidate channel ID is missing or cannot be resolved to an allowed type.
This replaces separate GLM checks in database selection, memory-cache
selection, affinity, and pinned-channel paths. A GLM effort alias may select
only:

- Zhipu V4;
- OpenAI;
- OpenRouter.

Legacy Zhipu V3 and unrelated channel types remain ineligible. Database and
cache selection must consume the same ordered candidates and the same filter
set. The distributor derives the allowed set from the incoming relay format:

- Chat Completions allows Zhipu V4, OpenAI, and OpenRouter;
- Responses allows Zhipu V4 only;
- Claude, Gemini, embeddings, images, and every other relay format reject the
  alias before channel selection.

This format-aware list prevents an OpenAI `/v1/responses` request from losing
its suffix after base-model mapping and keeps the two third-party strategies
confined to the user-requested Chat path.

## Alias normalization and model mapping

After a channel is selected and before its model-mapping chain runs, resolve a
validated GLM alias once. Preserve `RelayInfo.OriginModelName` exactly for
conditions and logs, retain the effort in suffix metadata, and use the base
model as the mapping source when no exact alias mapping exists.

`ModelMappedHelper` first attempts an exact alias key. When that key is absent,
it parses the validated GLM suffix and restarts the existing mapping chain from
the base key, preserving the original alias and the current cycle-detection
behavior. All mapping JSON decoding continues through `common.Unmarshal`;
business code must not call `encoding/json.Unmarshal` directly.

An explicit mapping for the complete alias remains authoritative. Otherwise,
this configuration is sufficient for OpenRouter:

```json
{
  "glm-5.2": "z-ai/glm-5.2",
  "glm-5.3": "z-ai/glm-5.3",
  "glm-5.3-flash": "z-ai/glm-5.3-flash"
}
```

Model mapping, rather than a late parameter override, owns model-name
redirection. It updates `UpstreamModelName` before adaptor conversion and keeps
request metadata, retries, logs, and the serialized payload aligned.
When a Zhipu mapping redirects the alias to another syntactically valid GLM
base, the mapped base remains authoritative and the original suffix effort is
still applied; provider validation decides whether that combination is valid.

Suffix metadata does not by itself claim the effort was applied. Each native
adaptor sets `RelayInfo.ReasoningEffort` only when it writes the corresponding
upstream field. OpenRouter parameter overrides use the existing post-override
sync to record the effort actually present in the final JSON.

## Zhipu V4 strategy

For Chat Completions, the Zhipu V4 adaptor changes the outgoing model to the
base model and writes the suffix to top-level `reasoning_effort`. The suffix
wins over a conflicting request-body effort. A bare GLM model preserves an
explicit body effort and keeps an absent effort absent.

For the rc.27 Responses path, the adaptor changes the outgoing model to the
base model and writes the suffix to `reasoning.effort`. The outgoing URL stays
the upstream-provided `/api/v1/responses` path. This conversion is confined to
Zhipu V4 and does not affect the standard OpenAI third-party relay design.

The existing Zhipu V4 discovery entries remain. Add the GLM-5.3-Flash base and
its three documented aliases for current discoverability. Future GLM bases can
be entered manually in a channel and use the generic suffix parser without a
new parser-table change.

## Standard OpenAI third-party strategy

The standard OpenAI channel supports GLM effort aliases only for
`/v1/chat/completions`. It keeps the incoming standard path and the configured
base URL; it must not construct `/api/paas/v4`, `/api/v1`, or other
Zhipu-specific paths.

The OpenAI adaptor writes the resolved effort to top-level
`reasoning_effort`, changes the request model to the mapped base model, and
updates relay logging metadata. The suffix wins over a conflicting body
effort. A bare model preserves an explicitly supplied `reasoning_effort` and
does not invent a default.

This behavior applies only when the selected channel type is OpenAI and the
requested name is a validated GLM effort alias. Existing OpenAI, Azure, and
non-GLM reasoning behavior remains unchanged.

A validated GLM effort alias requires body transformation. For Chat requests,
it therefore forces the ordinary adaptor path even when global or channel body
pass-through is enabled or Chat-Completions-via-Responses is configured. Bare
GLM models and non-GLM requests retain their existing pass-through and bridge
behavior.

## OpenRouter strategy

OpenRouter receives no new GLM conversion code in its adaptor. Shared routing
allows the alias to select an OpenRouter channel, shared normalization lets the
base-model mapping resolve to the provider-qualified ID, and the existing
post-serialization parameter override writes the final `reasoning` object.

The reusable channel parameter override is:

```json
{
  "operations": [
    {
      "path": "reasoning",
      "mode": "set",
      "value": { "effort": "none" },
      "conditions": [
        { "path": "original_model", "mode": "prefix", "value": "glm-" },
        { "path": "original_model", "mode": "suffix", "value": "-none" }
      ],
      "logic": "AND"
    },
    {
      "path": "reasoning",
      "mode": "set",
      "value": { "effort": "minimal" },
      "conditions": [
        { "path": "original_model", "mode": "prefix", "value": "glm-" },
        { "path": "original_model", "mode": "suffix", "value": "-minimal" }
      ],
      "logic": "AND"
    },
    {
      "path": "reasoning",
      "mode": "set",
      "value": { "effort": "low" },
      "conditions": [
        { "path": "original_model", "mode": "prefix", "value": "glm-" },
        { "path": "original_model", "mode": "suffix", "value": "-low" }
      ],
      "logic": "AND"
    },
    {
      "path": "reasoning",
      "mode": "set",
      "value": { "effort": "medium" },
      "conditions": [
        { "path": "original_model", "mode": "prefix", "value": "glm-" },
        { "path": "original_model", "mode": "suffix", "value": "-medium" }
      ],
      "logic": "AND"
    },
    {
      "path": "reasoning",
      "mode": "set",
      "value": { "effort": "high" },
      "conditions": [
        { "path": "original_model", "mode": "prefix", "value": "glm-" },
        { "path": "original_model", "mode": "suffix", "value": "-high" }
      ],
      "logic": "AND"
    },
    {
      "path": "reasoning",
      "mode": "set",
      "value": { "effort": "xhigh" },
      "conditions": [
        { "path": "original_model", "mode": "prefix", "value": "glm-" },
        { "path": "original_model", "mode": "suffix", "value": "-xhigh" }
      ],
      "logic": "AND"
    },
    {
      "path": "reasoning",
      "mode": "set",
      "value": { "effort": "max" },
      "conditions": [
        { "path": "original_model", "mode": "prefix", "value": "glm-" },
        { "path": "original_model", "mode": "suffix", "value": "-max" }
      ],
      "logic": "AND"
    }
  ]
}
```

Setting the complete object gives suffix intent precedence over any client
`enabled`, `max_tokens`, or conflicting effort. For example, a `-high` alias
replaces `{ "enabled": false, "max_tokens": 0 }` with
`{ "effort": "high" }` instead of merging contradictory controls.
`original_model` remains the
unmapped client alias, while `model` and `upstream_model` expose the mapped
provider ID. The existing override audit and reasoning-effort synchronization
record the final applied value.

## Error handling and compatibility

Unknown suffixes remain literal model names and receive no base fallback.
Malformed or differently cased GLM names remain unchanged. A recognized
effort that an upstream model does not support is forwarded as requested and
the upstream provider remains responsible for rejecting or normalizing it.

Failure to configure the OpenRouter model mapping leaves the complete client
alias in the outgoing request, which OpenRouter does not recognize. Failure to configure the parameter
override can produce a valid model request without the selected effort. Both
configuration requirements are part of the OpenRouter interface and receive
tests against the example configuration.

The old OpenRouter adaptor conversion and its
`ModelSuffixReasoningEffort` logging shortcut are not carried forward. The
feature also avoids direct `encoding/json` marshal or unmarshal calls in new
business code.

## Verification design

Tests are written before production changes and cover:

1. Relaykit parser positives for GLM-5.2, GLM-5.3,
   GLM-5.3-Flash, and a future-shaped GLM base; negatives for bare models,
   unknown suffixes, embedded `glm`, `glm-low`, `glm--low`, case changes, and
   extra trailing text.
2. Exact alias, existing wildcard, and base fallback candidate order.
3. Pricing normalization with exact-alias precedence and base fallback.
4. Database, memory-cache, affinity, and pinned-channel selection through
   rc.27's `ChannelSatisfiesFilters` seam for Zhipu V4, OpenAI, and OpenRouter,
   plus rejection of Zhipu V3 and an unrelated channel type. Format cases prove
   Chat allows all three, Responses allows only Zhipu V4, and OpenAI Responses,
   Claude, Gemini, and unrelated relay formats reject the alias.
5. Model mapping from a base key to `z-ai/glm-5.3-flash`, preservation of an
   exact alias mapping, preservation of `OriginModelName`, and cycle detection.
6. Zhipu V4 Chat serialization with top-level `reasoning_effort`, suffix
   precedence, bare-model preservation, and GLM-5.3-Flash discovery.
7. Zhipu V4 Responses serialization with `reasoning.effort`, correct model
   stripping, and applied-effort logging.
8. Standard OpenAI Chat serialization on `/v1/chat/completions`, top-level
   `reasoning_effort`, no Zhipu path construction, suffix precedence, and bare
   model behavior.
9. OpenRouter Chat serialization using the documented model mapping and
   parameter override, including all seven recognized effort values, complete
   reasoning-object replacement, mapped model ID, and final log metadata.
10. Regression coverage proving non-GLM OpenAI and OpenRouter requests remain
    unchanged.
11. `git diff --check`, focused package tests, root `go test -count=1 ./...`,
    `go build ./...`, `go vet ./...`, and independent relaykit tests and build
    with `GOWORK=off`.

## Branch and release placement

Implementation starts only after the rc.27 upgrade is complete and
`dev/v1.0.0-rc.27` exists. The working branch is
`feat/rc27/glm-generic-effort-suffixes`. After focused tests and review, it
merges into rc.27 development as part of the indivisible release candidate.

After release, append the feature's signed commits to
`feat/v1.0.0-rc.27/reasoning-model-support`, verify that the backup contains
only that theme, and push the backup before any monitored release push.

## Acceptance criteria

The feature is complete when a base-configured Zhipu V4, OpenAI, or OpenRouter
channel accepts a valid GLM effort alias; the final upstream model and effort
match the selected channel strategy; GLM-5.3-Flash supports `low`, `high`, and
`max`; OpenRouter uses only mapping plus parameter overrides for effort; all
selection paths and logs agree; non-GLM behavior is unchanged; every required
test and build gate passes; and the resulting commits and backup ref are
signed and verified.
