# GLM reasoning effort suffixes on OpenAI and OpenRouter channels

## Context

`dev/v1.0.0-rc.25` exposes the GLM reasoning effort aliases `glm-5.2-none`,
`glm-5.2-high`, `glm-5.2-max`, `glm-5.3-low`, `glm-5.3-high`, and
`glm-5.3-max`. Both earlier designs deliberately restricted every validated
alias to the `Zhipu V4` channel, because the documented request contract was
the V4 Chat Completions protocol.

That restriction is enforced by `requiresZhipuV4Channel` in
`model/model_match.go`, which is consulted by database channel selection
(`model/ability.go`), memory-cache channel selection (`model/channel_cache.go`),
and channel affinity (`model/channel_satisfy.go`). Every candidate whose
`channel.Type` is not `ChannelTypeZhipu_v4` is dropped. The alias-to-effort
translation itself lives only in `applyGLMReasoningEffort` in
`relay/channel/zhipu_4v/adaptor.go`.

The practical consequence is that an alias requested against an `OpenAI`
channel or an `OpenRouter` channel never reaches an adaptor at all: channel
selection removes every candidate and the request fails with no available
channel. Zhipu's own OpenAI-compatible endpoint and OpenRouter both accept the
same reasoning effort concept, so the restriction now blocks working
configurations rather than protecting an unsupported protocol.

## Selected design

Widen the restriction from a single channel type to an allowlist of channel
types whose adaptors translate a GLM alias, and add the missing translation to
the OpenAI adaptor.

### Channel allowlist

`constant.SupportsGLMReasoningEffortAlias` reports whether a channel type can
serve a GLM effort alias. The allowlist holds `ChannelTypeZhipu_v4`,
`ChannelTypeOpenAI`, and `ChannelTypeOpenRouter`. `requiresZhipuV4Channel`
becomes `requiresGLMEffortChannel`, and each of the four selection call sites
tests the allowlist instead of the single constant. The affinity query that
previously joined on `channels.type = ?` now uses `channels.type IN ?`, which
behaves identically on SQLite, MySQL, and PostgreSQL through GORM.

Channel types outside the allowlist keep the previous behavior: a validated
alias never selects them.

### Alias normalization ahead of model mapping

Channel model redirection resolves its chain from `RelayInfo.OriginModelName`,
so a channel that maps `glm-5.3` to `z-ai/glm-5.3` never matches the client
name `glm-5.3-high` and the alias reaches the upstream verbatim. Channel
selection already treats the alias as its base model through
`ModelMatchCandidates`, so model mapping must see the same base model.

`ModelMappedHelper` therefore normalizes a validated alias to its base model
before resolving the redirect chain, for allowlisted channel types other than
`Zhipu V4`. The recovered effort is recorded on
`RelayInfo.ModelSuffixReasoningEffort`, and `RelayInfo.ReasoningEffort` is set
so logging reports the effort actually applied. `IsModelMapped` is set because
the upstream model name genuinely differs from the requested one.

`Zhipu V4` is excluded from this step so its existing adaptor path, its alias
recovery from `OriginModelName`, and its logging behavior stay byte-for-byte
unchanged.

### OpenAI adaptor translation

`ConvertOpenAIRequest` in `relay/channel/openai/adaptor.go` gains a GLM step
that runs before the OpenRouter reasoning block, because that block consumes
`request.ReasoningEffort` and rewrites it into OpenRouter's `reasoning` object.
The step covers two shapes:

1. The upstream model is still the alias, which happens when the channel
   configures no redirect. The alias is split, `request.Model` and
   `info.UpstreamModelName` become the base model, and the effort is written to
   `request.ReasoningEffort`.
2. `ModelSuffixReasoningEffort` is already recorded, which happens when a
   redirect rewrote the base model into a vendor-prefixed upstream name. The
   recorded effort is written to `request.ReasoningEffort`.

In both shapes the suffix wins over an effort the request body already carried,
matching the Zhipu V4 precedence rule.

### Disabled reasoning on OpenRouter

`glm-5.2-none` means reasoning is disabled. The OpenRouter block currently
drops a `none` effort without emitting any `reasoning` field, which leaves the
upstream default in place instead of disabling reasoning. When the `none`
effort came from a GLM alias, the block emits `reasoning` with `enabled` set to
`false`. A `none` effort that a client sent directly keeps its current
behavior, so no unrelated OpenRouter model changes.

`GLM-5.3` has no `none` level, so only `GLM-5.2` reaches this branch.

## Alternatives considered

Teaching `ParseGLMReasoningEffortSuffix` to accept a vendor prefix such as
`z-ai/glm-5.3-high` would let a client address the OpenRouter model directly,
but it would also make the parser accept any prefix from any provider and would
not fix the redirect chain for a channel that lists the plain base model.
Normalizing inside the adaptor alone cannot work, because model redirection has
already run by then. Removing the channel restriction entirely would route an
alias to providers that never advertised the effort levels.

## Error handling and boundaries

Unsupported names such as `glm-5.3-none`, `glm-5.2-low`, `glm-5.3-xhigh`, and
differently cased aliases stay unrecognized and unchanged. No pricing value,
Base URL selection, regional routing, response conversion, or deployment
behavior changes. Channel types outside the allowlist keep refusing aliases.
This feature covers OpenAI Chat Completions conversion only.

## Verification

1. Allowlist membership for `Zhipu V4`, `OpenAI`, and `OpenRouter`, and
   exclusion for an unrelated channel type.
2. Channel selection for an alias on an `OpenAI` channel and an `OpenRouter`
   channel, plus continued rejection of a non-allowlisted channel type.
3. Channel affinity for an alias on an allowlisted channel type.
4. `ModelMappedHelper` normalizing an alias to its base model, resolving a
   redirect from the base model, and recording the effort.
5. `ModelMappedHelper` leaving `Zhipu V4` requests untouched.
6. OpenAI adaptor payloads for an un-redirected alias and for a recorded
   suffix effort, including suffix precedence over a body effort.
7. OpenRouter payloads for `high` and for the `none` disable case.
8. relaykit independent build, focused package tests, root `go test ./...`,
   `go build ./...`, `go vet ./...`, formatting, and `git diff --check`.
