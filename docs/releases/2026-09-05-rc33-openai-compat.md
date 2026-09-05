# rc.33 OpenAI compatibility and reasoning

This candidate integrates upstream `v1.0.0-rc.33` from the published rc.32
commit `2a673c04b2f2b22b85b700c188c5f5e5fc02e2be`.
The signed upstream integration is
`b6691ae6a5270df12d58621e686fee818eed04aa`.

## Runtime fixes

The rc.32 conversion registry rejected native Responses tools added to a
Chat Completions request by existing channel parameter overrides. It failed
before the older Chat-to-Responses converter could preserve those tools.
The registry now accepts the flat tool shape, strips the chat-only function
object, and preserves search types, filters, and code-interpreter containers.
Native tool payloads carried through `custom` also remain available.

Structured parameter-override operations using the body path
`reasoning_effort` now resolve to `reasoning.effort` when the outgoing request
format is Responses. Resolution happens before `keep_origin`, so a requested
`-low` or `-max` effort is preserved. The saved channel configuration is not
rewritten. Chat requests still apply the original Chat field before conversion.
This compatibility alias covers path-based body operations; move/copy endpoints
and legacy flat override maps retain their existing behavior.

Upstream rc.33 preserves native effort values instead of implicitly projecting
`max` to `xhigh`. The superseded local model-specific projection wrapper was
retired. Astra validation, model-mapping identity, and sampling cleanup remain
active on both the new fast path and the conversion path.

## Pricing and model discovery

Astra uses the upstream self-contained expression default, including the
272K context threshold and cache categories. Model-family lookup applies this
default to valid aliases. Explicit administrator billing modes, expressions,
ratio prices, per-call prices, and valid wildcard overrides remain effective;
zero prices remain valid overrides. Existing stored Astra ratio prices are
preserved rather than silently migrated to expression pricing.

Base-only model registration, pricing inheritance, and automatic model-list
expansion from rc.32 remain included. The five existing customization themes
remain the release backup structure.

## Validation

The local backend used a separate PostgreSQL 18.6 database restored from an
existing production backup. Login and relay requests used the normal API
authentication paths. The original channel parameter override was unchanged.
All three final curl requests reached the real configured upstream:

| Request | Requested model | HTTP | Answer | Recorded effort |
| --- | --- | --- | --- | --- |
| POST /v1/chat/completions | gpt-5.6-luna-low | 200 | RC33-OK | low |
| POST /v1/responses | gpt-5.6-luna-low | 200 | RC33-OK | low |
| POST /v1/responses | gpt-5.6-luna-pro-max | 200 | RC33-OK | max |

The reported invalid-channel-ID error did not reproduce with the valid local
test token and channel. Authentication and pinned-channel validation were not
relaxed. The original client-side channel-selection cause remains unconfirmed.

Passed checks:

- `go test ./relay/common ./relay ./controller ./setting/billing_setting ./setting/ratio_setting`
- `go test ./service ./relay/channel/deepseek ./relay/channel/xai ./relay/channel/volcengine`
- `go test ./relay/channel/openai ./relay/helper`
- `go test ./middleware ./model`
- From `relaykit`: `go test ./relayconvert/...`
- From `relaykit`: `GOWORK=off go build ./...`
- Root build of the complete rc.33 backend.
- Scoped `go vet`, `gofmt -d`, and `git diff --check`.

Unaffected passing checks were reused. No new database schema or driver changes
were introduced, and frontend sources are unchanged from the rc.32 release.
Regression cases were consolidated into existing test files.

Claude Fable 5.1 with effort high reviewed the tool fix and the final rc.33
effort, override, and billing integration. No blocking findings remained.
Deployment evidence is recorded separately after promotion.

## Astra omitted-effort follow-up

A bare Astra request could still inherit `none` from a shared legacy channel
default (`set`, `keep_origin: true`). Astra rejects that value. The override
executor now leaves effort absent for this specific default and lets the
upstream model choose its default. Explicit request values, forced overrides,
valid channel defaults, and GPT-5.6 behavior remain unchanged.

With the original channel override and `^gpt-6.*$` conversion policy on the
local database copy, both Chat and Responses requests for bare `gpt-6-astra`
returned HTTP 200 and `RC33-OK` from the configured real upstream. Scoped
`go test ./relay/common ./relay/channel/openai`, `go vet ./relay/common`,
formatting, and the complete backend build passed. Claude Fable 5.1 at high
effort reviewed the follow-up without blocking findings.
