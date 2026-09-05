# Model family discovery and pricing

## Approved behavior

- Register only the base model in an OpenAI channel and configure that base model's pricing once.
- Supported families: `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna`, `gpt-6-astra`.
- Publish concrete valid effort/mode variants in `/v1/models`, the user model selector, and the OpenAI channel model selector. Generate them from the existing validated suffix grammar, not an independent list of aliases.
- Keep group and token model restrictions, including exact-only token permissions. Deduplicate variants when both base and wildcard registrations exist.
- A wildcard-only registration publishes suffix variants, not the bare base. Keep unknown vendor IDs and explicitly registered variants unchanged.
- Skip configured thinking-suffix exemptions. Configuration-only family wildcard keys are not callable model IDs and are not published.
- The unset-price selector offers a base model once. Existing explicit alias/wildcard price overrides remain editable and retain priority; no stored pricing entries are deleted or rewritten.
- A recognized wildcard pricing item inherits the base price only when it has no explicit value. Apply this to token, completion, cache, fixed-price and expression lookup through the shared candidate logic. Zero is a configured value.
- Preserve provider ownership and inherit endpoint metadata. OpenAI base and effort-only names support Chat and Responses; mode aliases advertise Responses. Chat-only clients still require the existing Chat-to-Responses setting for mode selection. Do not enable bridge settings automatically.
- Preserve Sora model choices alongside OpenAI text models in the channel selector.

## Implementation

Pure family-name generation lives in `relaykit/relayconvert/reasoning/model_family.go`; the host wrapper applies suffix-exemption settings and deduplication. Controllers use those names for discovery while enabled-model pricing candidates collapse to their registered base. Pricing map lookup retains existing specificity and adds a base fallback for recognized configuration wildcards.

No schema, persisted settings, credentials, or production services are changed. The frontend consumes existing API response shapes; no locale or frontend source changes are required.

## Validation scope

Focused API tests cover family expansion, group/token filtering, duplicate removal, selectors, retrieval and metadata. Pricing tests cover inheritance, zero prices and preserved overrides. Existing tests in affected packages are run once for regression coverage. Relaykit is built independently with `GOWORK=off`. No repeated smoke tests or live provider requests are needed for these deterministic transformations.

Base for review: `13650f6cb2e6a934fce04b6712bf1511c45f4f5a`.

## Results

- PASS: `go test ./controller ./model ./common ./relay/channel/openai ./setting/reasoning ./setting/ratio_setting ./setting/billing_setting`.
- PASS: focused model-family API, group-limit, selector and pricing regressions after review adjustments.
- PASS: root build, affected-package `go vet`, and independent `GOWORK=off` relaykit build.
- Standards and Spec reviews: separate Claude Fable 5.1 sessions, effort `high`; neither reported blocking findings. Applied the empty-group guard, complete fixture setup, clearer conditions and OpenAI-only selector scope. Public model counts remain intentional contract assertions. The exact endpoint-cache lookup remains necessary for legacy wildcard configuration keys.
- The frontend source and locale files are unchanged. Existing selectors consume the updated API data.
- Local implementation only. No remote push, production deployment, or real provider request was performed.

## Qwen 3.8 and later max and flash families

**Configuration.** The channel is a standard OpenAI channel type with the custom base URL `https://dashscope-intl.aliyuncs.com/compatible-mode`. Only the two base names, `qwen3.8-flash` and `qwen3.8-max`, are registered in the channel model list and in base pricing. No built-in provider prices were added; the local fixture used two test billing expressions that are not published DashScope prices. Exact administrator suffix or wildcard price overrides keep their higher priority where existing pricing supports them, and zero prices remain valid explicit values.

**Scope.** Names with a numeric version of 3.8 or later and the `max` or `flash` family, optionally followed by an `MMDD` or `YYYY-MM-DD` snapshot, are recognized, so `qwen3.9-max` and `qwen3.10-flash` expand automatically. Qwen 3.7 and the coder, VL, embedding and other families are excluded. Each base yields only `-none`, `-low`, `-medium` and `-xhigh`; no `minimal`, `high` or `max` aliases exist. Provider support of this four-effort contract for future versions is assumed, not verified.

**Verified behavior.** `/v1/models` lists the two bases plus eight aliases, while the price catalog and public pricing page keep exactly two base entries. A suffix alias is sent upstream as the base model with a top-level `reasoning_effort`, and a concrete Qwen suffix clears conflicting `enable_thinking` and `thinking_budget`. A bare model with no caller-supplied effort or vendor thinking controls omits effort so the provider default applies; explicit body effort and vendor thinking controls on a bare model are preserved. Suffix beats conflicting body effort, explicit `@effort` or `@thinking` within one name beats the suffix, mapped model controls beat original controls, and channel parameter overrides remain final. 18 real DashScope requests through a loopback observation proxy returned HTTP 200 with the expected outgoing fields and reasoning presence; consume-log charges were independently recalculated for the initial 16. The provider does not echo the effective effort, so verification relies on the sent field and returned reasoning and usage. Suffix-bearing Chat requests stay on the Chat path; the native DashScope Responses API is not claimed as supported.
