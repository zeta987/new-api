# GLM-5.3 reasoning effort suffixes

## Context

`dev/v1.0.0-rc.25` already carries the GLM-5.2 reasoning effort suffix
feature. It exposes `glm-5.2-none`, `glm-5.2-high`, and `glm-5.2-max` as client
model names, maps them to the base model `glm-5.2`, and sends the matching
top-level `reasoning_effort` through the `Zhipu V4` channel only.

Zhipu has since released GLM-5.3. Its documentation defines the same top-level
`reasoning_effort` field, but with a different value set. GLM-5.3 always
operates with reasoning enabled, `thinking.type` accepts `enabled` only, and
disabling reasoning is no longer supported. The accepted efforts are `low`
(Lightweight Reasoning), `high` (Enhanced Reasoning), and `max` (Deep
Reasoning), with `max` as the default when the field is omitted. GLM-5.2 keeps
its own `none`, `high`, and `max` set, so `none` is valid only for GLM-5.2 and
`low` is valid only for GLM-5.3.

The request contract and the official examples still use the V4 Chat
Completions protocol, so this feature stays limited to the `Zhipu V4` channel
for the same reason GLM-5.2 did. The legacy V3 `Zhipu` channel is unchanged.

## Selected design

Add exact GLM-5.3 aliases:

| Client model | Upstream model | Upstream `reasoning_effort` |
| --- | --- | --- |
| `glm-5.3-low` | `glm-5.3` | `low` |
| `glm-5.3-high` | `glm-5.3` | `high` |
| `glm-5.3-max` | `glm-5.3` | `max` |

Rather than adding a second single-model parser beside
`ParseGLM52ReasoningEffortSuffix`, generalize the existing one into a
table-driven `ParseGLMReasoningEffortSuffix` in the independently buildable
relaykit reasoning package. The table maps each GLM base model to the exact
effort values it accepts:

```text
glm-5.2 -> none, high, max
glm-5.3 -> low,  high, max
```

A table keeps the per-model value sets authoritative in one place, so
`glm-5.2-low` and `glm-5.3-none` both stay unrecognized instead of being
silently accepted by a shared suffix list. It also keeps every downstream call
site — model matching, pricing normalization, channel restriction, and the V4
adaptor — unchanged in shape as future GLM releases arrive.

A companion `IsGLMReasoningEffortModel` reports whether a name is one of the
table's base models. The V4 adaptor and the V4 request conversion use it in
place of the previous literal `glm-5.2` comparisons.

## Alias recovery across base models

The adaptor inspects the mapped upstream model first, and recovers the alias
from `RelayInfo.OriginModelName` when the upstream model is already a GLM base
model. With two base models in the table, recovery must additionally confirm
that the recovered alias belongs to the same base model that the request is
actually being sent to. Otherwise a channel mapping from `glm-5.2-high` to
`glm-5.3` would apply an effort the target model never advertised. The
recovered base model must equal the upstream model, or the request keeps
whatever effort the body already carried.

## Request conversion

`requestOpenAI2Zhipu` copies `ReasoningEffort` into the rebuilt
OpenAI-compatible whitelist payload when the outgoing model is any GLM base
model in the table. The existing `thinking` copy stays unchanged, and the
adaptor never enables or disables thinking implicitly. Because GLM-5.3 refuses
to disable reasoning, a caller that wants lighter reasoning uses
`glm-5.3-low`; this feature does not translate an absent or disabled
`thinking` field into anything else.

Add `glm-5.3` and its three aliases to the Zhipu V4 model list. Model matching
normalizes only these exact aliases to `glm-5.3`, so a V4 channel configured
with the base model accepts an alias while exact alias configuration still
wins. Every validated alias remains restricted to a Zhipu V4 channel in
database selection, memory-cache selection, and channel-affinity checks,
exactly as GLM-5.2 already is.

## Alternatives considered

Duplicating the GLM-5.2 parser as `ParseGLM53ReasoningEffortSuffix` would
require every call site to test both functions and would grow linearly with
each GLM release. Sharing one effort list across both models would wrongly
accept `glm-5.2-low` and `glm-5.3-none`. Extending the generic OpenAI suffix
list would affect unrelated providers. The table-driven exact parser avoids all
three problems.

## Error handling and boundaries

Unsupported names such as `glm-5.3-none`, `glm-5.3-xhigh`,
`glm-5.3-max-extra`, `glm-5.2-low`, differently cased aliases, and other model
families stay unchanged. No pricing value, Base URL selection, regional
routing, response conversion, or deployment behavior changes. This feature
covers OpenAI Chat Completions conversion through the Zhipu V4 adaptor only.

## Verification

1. Exact parser results for every GLM-5.2 and GLM-5.3 alias, including the
   cross-model negatives `glm-5.2-low` and `glm-5.3-none`.
2. V4 converted structs plus exact serialized payload fields for GLM-5.3.
3. Suffix precedence over a conflicting body effort.
4. Bare `glm-5.3` preserving an explicit effort and omitting an absent one.
5. Origin alias recovery only when the alias base model matches the upstream
   model, including the cross-base rejection.
6. V4 model-list exposure, V3 model-list exclusion, and model-match candidates
   `[alias, glm-5.3]`.
7. Pricing normalization and exact-alias precedence for GLM-5.3.
8. Zhipu V4 channel restriction for a GLM-5.3 alias.
9. relaykit independent build, focused package tests, root `go test ./...`,
   `go build ./...`, `go vet ./...`, formatting, and `git diff --check`.
