# GLM-5.2 reasoning effort suffixes

## Context

new-api exposes two Zhipu channel types. `Zhipu` uses the legacy V3
`model-api` protocol, while `Zhipu V4` uses the V4 Chat Completions protocol
with a Bearer API key and an OpenAI-compatible payload. The channel type
selects the protocol. The configured Base URL selects the China or
international host.

The China and international GLM-5.2 documentation both define top-level
`reasoning_effort`, with `none`, `high`, and `max` as the canonical values in
scope here. When thinking is enabled and the field is omitted, the provider
uses `max`. Both official request examples use the V4 Chat Completions
protocol, so this feature applies only to the `Zhipu V4` channel. The legacy
V3 `Zhipu` channel remains unchanged even though runtime probing found that
its upstream endpoint currently accepts the same fields.

Grok effort suffixes are already integrated into `dev/v1.0.0-rc.24`. The GLM
work starts from that development commit and remains in an independent topic
branch. The pending local release merge will not be pushed before GLM is
integrated, allowing both features to enter one release candidate.

## Selected design

Add an exact GLM-5.2 parser for these aliases only:

| Client model | Upstream model | Upstream `reasoning_effort` |
| --- | --- | --- |
| `glm-5.2-none` | `glm-5.2` | `none` |
| `glm-5.2-high` | `glm-5.2` | `high` |
| `glm-5.2-max` | `glm-5.2` | `max` |

The parser belongs in the independently buildable relaykit reasoning package
because the V4 adaptor and model matching need the same validated behavior.
It must require the exact base model `glm-5.2`; generic suffix stripping would
incorrectly reinterpret unrelated models whose real names end in `-high` or
`-max`.

The Zhipu V4 adaptor will inspect the mapped upstream model first. If the
upstream model is already `glm-5.2`, it may recover the alias from
`OriginModelName`, preserving suffix intent after model matching or an
explicit alias-to-base mapping. A valid suffix replaces any body-provided
effort. A bare `glm-5.2` preserves an explicitly provided body effort and
keeps an omitted effort omitted, allowing the provider's documented `max`
default to apply.

The adaptor will update the outgoing model, `RelayInfo.UpstreamModelName`, and
`RelayInfo.ReasoningEffort` together. It will not enable or disable thinking
implicitly. The caller's `thinking` field is transmitted unchanged.

## Request conversion

For the V4 channel, copy `ReasoningEffort` into the rebuilt
OpenAI-compatible whitelist payload only after the outgoing model is
`glm-5.2`. The existing `thinking` copy remains unchanged. This keeps body
effort omitted for unrelated and invalid GLM model names.

Add `glm-5.2` and its three aliases to the Zhipu V4 model list. Normalize only
these three exact aliases to `glm-5.2` in model matching and pricing fallback,
so a V4 channel configured with the base model can accept an alias while exact
alias configuration still wins. The V3 model list and adaptor are unchanged.

The resulting request path is:

```text
client alias
  -> exact alias or base-model channel match
  -> Zhipu V4 adaptor validates suffix
  -> model becomes glm-5.2
  -> reasoning_effort becomes none, high, or max
  -> V4 request is sent to the configured Base URL
```

Every validated GLM-5.2 alias is restricted to a Zhipu V4 channel in database
selection, memory-cache selection, and channel-affinity checks. Exact V4 alias
configuration still takes precedence over a V4 base-model fallback. Bare
`glm-5.2` retains its existing cross-channel selection behavior.

## Alternatives considered

Adding `max` to the generic OpenAI suffix list is smaller, but it affects
unrelated providers and can strip legitimate model names. Extending both
Zhipu adaptors would expose the aliases through the legacy protocol despite
the official GLM-5.2 examples using V4. Keeping the parser private inside the
V4 adaptor would also make model matching duplicate its validation rules. The
selected exact shared parser avoids these trade-offs.

## Error handling and boundaries

Unsupported names such as `glm-5.2-low`, `glm-5.2-xhigh`,
`glm-5.2-max-extra`, differently cased aliases, and other model families stay
unchanged. Existing nil-request errors remain unchanged. This feature covers
OpenAI Chat Completions conversion through the Zhipu V4 adaptor only. That
adaptor does not currently implement OpenAI Responses conversion, and this
work does not add it.

No Base URL selection, API-key-origin detection, automatic regional routing,
pricing value, response conversion, or production deployment behavior changes
as part of this feature.

## Verification

Tests will be written before production changes and will cover:

1. Exact relaykit parser results for all three aliases and negative names.
2. V4 converted structs plus exact serialized payload fields.
3. Suffix precedence over a conflicting body effort.
4. Bare-model preservation of explicit effort and omission of implicit max.
5. Preservation of caller-provided `thinking` in the V4 payload.
6. Alias recovery when `OriginModelName` contains the suffix and the upstream
   model is already `glm-5.2`.
7. V4 model-list exposure and model-match candidates `[alias, glm-5.2]`.
8. Confirmation that the V3 model list and request conversion remain
   unchanged.
9. relaykit independent build, focused package tests, root `go test ./...`,
   `go build ./...`, `go vet ./...`, formatting, and `git diff --check`.

The previously observed Windows HTTP/2 graceful-GOAWAY test can fail during a
full run because the host aborts the loopback connection. It passed when rerun
alone on the unmodified baseline and will be recorded separately if it appears
again after the GLM changes.
