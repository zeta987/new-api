# Usage-log refresh and global rate limits

Inspected on 2026-09-07 at checkout `bf5deb488d1f17515b911b386e131ef4e468353a`.
Scope: source inspection and isolated local reproductions. The deployed image,
effective environment, proxy configuration, and production network traffic were
not inspected. No production behavior was modified.

The findings below describe the original checkout. The local remediation and
its verification are recorded in the final section.

## Findings

### Event-driven dashboard requests can exhaust the configured API allowance

`createLog` publishes an event after a successful database insertion
([model/log.go](../../model/log.go#L101)). The controller forwards these events on
one SSE connection and sends `ready` when connecting
([controller/log_stream.go](../../controller/log_stream.go#L12)). The browser
handles both `ready` and `log`
([log-stream.ts](../../web/src/features/usage-logs/lib/log-stream.ts#L31)).

Each callback immediately invalidates `logs` and `usage-logs-stats`, with no
minimum refresh interval
([usage-logs-provider.tsx](../../web/src/features/usage-logs/components/usage-logs-provider.tsx#L76)).
The common-log page has an active
[list query](../../web/src/features/usage-logs/components/usage-logs-table.tsx#L128)
and an active
[statistics query](../../web/src/features/usage-logs/components/common-logs-stats.tsx#L56).
For an administrator viewing all logs, these fetch `/api/log` and
`/api/log/stat`; self view fetches `/api/log/self` and `/api/log/self/stat`
([api.ts](../../web/src/features/usage-logs/api.ts#L38)). The administrator list
request can incur Gin's trailing-slash redirect before reaching `/api/log/`.
That redirect does not create a second successful passage through the limiter.

The 10-second query `staleTime` does not impose a minimum refresh interval;
explicit invalidation triggers active refetches
([main.tsx](../../web/src/main.tsx#L53)). Identical in-flight HTTP GETs are joined,
but that protection ends when the request settles
([http-client.ts](../../web/src/lib/http-client.ts#L52)). Consequently, two counted
management requests per event is achievable when successive events arrive after
the previous requests complete. Simultaneous events can coalesce; an exact
two-requests-per-log ratio is not guaranteed for every traffic pattern.

With `GLOBAL_API_RATE_LIMIT=180` and duration 180 seconds, 90 completed refresh
pairs consume the entire allowance before accounting for initial page requests,
SSE connection attempts, other dashboard actions, or additional browser tabs.
One event per second can request 360 counted list/stat operations in 180 seconds.

### A final rate-limited event can leave cached logs stale

Production query retry configuration excludes only HTTP 401 and 403 and stops
when `failureCount > 3`
([main.tsx](../../web/src/main.tsx#L56)). With the installed TanStack Query retryer,
429 therefore receives four retries after the initial attempt, ordinarily at
1, 2, 4, and 8 second delays. Two failing queries can make ten requests across
approximately 15 seconds when no other trigger intervenes.

The server supplies `Retry-After`
([middleware/rate-limit.go](../../middleware/rate-limit.go#L151)), but the frontend
HTTP interceptor does not use it
([http-client.ts](../../web/src/lib/http-client.ts#L100)). New events can initiate
another refetch while a prior query is waiting for a retry. After retries are
exhausted, neither common-log query schedules polling; window-focus refetching
is also disabled. If no further event or reconnect occurs, expiration of the
180-second server limit does not itself cause a new browser fetch. Cached list
and statistics data can remain stale until a new event, successful SSE `ready`,
filter action, remount, or another query refetch trigger.

The SSE wrapper also does not interpret 429 or `Retry-After`; the installed
`sse.js` default reconnect delay is 3 seconds, with unlimited retries. This is
relevant when the stream connection itself is rejected or disconnected; an
already established stream does not spend one API allowance per SSE message.

## What the supplied environment variables limit

| Traffic | Limiter | Configured allowance |
| --- | --- | --- |
| Registered `/api` management routes, including log list, stats, and SSE connection establishment | `GA` plus `ClientIP()` | 180 requests / 180 seconds |
| `/dashboard/billing` compatibility routes | Same `GA` bucket for the same client IP | Shares the allowance above |
| Unmatched routes handled by the embedded frontend, including document and asset requests | `GW` plus `ClientIP()` | 60 requests / 180 seconds |
| Registered `/v1/chat/completions`, `/v1/messages`, and Gemini relay routes | Separate model-request controls | Not governed by these six variables |

Sources: [API registration](../../router/api-router.go#L15),
[dashboard registration](../../router/dashboard.go#L12),
[web fallback](../../router/web-router.go#L22),
[relay registration](../../router/relay-router.go#L69), and
[limiter keys/factories](../../middleware/rate-limit.go#L43).

The limiter counts requests by route category and resolved client IP; it does
not identify F5 presses. Browser-generated log queries spend the browser's IP
allowance even when the originating model request came from a different machine.
Administrator subscriptions include every user's log event
([controller](../../controller/log_stream.go#L13),
[broker](../../model/log_event.go#L44)). An administrator selecting self view
still receives the administrator-wide stream because its scope is chosen from
the authenticated role, independently of the frontend view filter.

Redis uses an atomic fixed window, while the in-memory implementation keeps
accepted-request timestamps. Both reject request 181 when 180 requests have
already been admitted in the relevant interval. Their boundary and Retry-After
semantics differ: Redis reports remaining TTL, while memory reports the full
configured duration
([Redis and memory adapters](../../middleware/rate-limit.go#L118),
[in-memory storage](../../common/rate-limit.go#L48)).

The inspected checkout defaults to API 360 and web 120, both over 180 seconds;
the supplied explicit values override those defaults with half the allowance
([common/init.go](../../common/init.go#L122)). The
[official environment-variable guide](https://docs.newapi.pro/en/docs/installation/config-maintenance/environment-variables)
describes IP-based counts and second-based durations but lists older 180/60
defaults. The fork's source controls its current defaults and route attachment.

## Prior repair evidence

The rc.33 reusable backup is
`b56f8270881a2b843ff0beabd4cb85eb8e9748b6`, titled
`fix: back up rc.33 live usage logs`; it contains the broker, SSE controller,
browser event bridge, provider invalidation, and subscription tests.
Commit `74373bad18bd878d610660d95f368efefb7757ad`, titled
`fix: resubscribe logs after token rotation`, recreates the stream when the access
token changes and closes it after sign-out. Both behaviors remain in this
checkout. Searches of the available agent conversation index returned no matching
conversation; Git history and current source supplied the repair evidence.

Existing tests cover event delivery, subscriber scope, cleanup, and token
rotation. They do not cover request amplification, 429 cooldown, or recovery
after the last rejected refresh
([stream tests](../../web/src/features/usage-logs/lib/log-stream.test.ts),
[browser-event tests](../../web/src/features/usage-logs/lib/refresh-events.test.ts),
[provider tests](../../web/src/features/usage-logs/components/__tests__/usage-logs-provider.test.tsx),
[broker tests](../../model/log_event_test.go)).

## Local reproduction results

Temporary investigation fixtures were removed after execution; no production
source files or existing tests were changed.

1. `bun .tmp-usage-rate-audit.mjs`: extracted the current provider's `refreshLogs`
   callback and executed it against the installed real TanStack Query core with
   two active observers, 10-second staleTime, and completed previous requests.
   91 events produced exactly 91 list calls and 91 statistics calls: 182 total.
   This isolates query scheduling; these were local query-function invocations,
   not requests against a running application server.
2. `go test ./middleware -run TestUsageRateAudit -count=1 -v`: used the actual
   rate-limit implementation with httptest, once with in-memory storage and once
   with miniredis running the Lua limiter. 180 management requests were admitted,
   request 181 returned HTTP 429 with `Retry-After: 180`, and the same-IP web
   bucket and a different-IP API bucket remained available. Both subtests passed.
   Endpoint handlers were stubs; relay route attachment was verified separately
   by source inspection, not by calling an upstream provider.

## Repair scope suggested by this evidence

Coalesce incoming events with a bounded refresh interval and retain a pending
refresh while queries are running. Treat 429 as a shared dashboard cooldown
using `Retry-After`, retaining a final catch-up fetch after the cooldown even
when no new log event arrives. Apply the same cooldown to stream reconnection
when its handshake is rejected. Higher API allowance can postpone the observed
threshold, but does not bound the existing event-to-query fanout.

The current broker is process-local and the stream has no periodic heartbeat.
Cross-instance delivery and idle-proxy disconnect behavior are additional
deployment-dependent concerns, not established explanations for an observed
production incident in this audit.

## Local remediation and verification

Implemented on `fix/rc33/usage-logs-background-refresh`, based on the checkout
above. Production release and environment variables were not changed.

- Coalesce incoming events with a 10-second minimum interval between automatic
  list/stat refreshes. Retain a pending refresh during existing requests and
  preserve their transports. No recurring polling is started when there are no
  events or failures.
- Pause hidden tabs and close their SSE subscriptions. Resume through the same
  deadline, retaining 429 cooldowns across visibility changes and token rotation.
- Honor `Retry-After` for usage queries and rejected stream handshakes, with a
  conservative fallback when the header is missing. Preserve a catch-up after
  the final rejected event. Network failures, 408, and 5xx also schedule recovery;
  permanent authorization errors and cancellations do not trigger that retry.
- Suppress repeated 429 toasts only for requests handled by this usage-log
  recovery path. Other HTTP error handling retains its previous behavior.
- Opt the usage table into interaction during background fetching. Existing
  DataTablePage consumers retain their default fetching behavior.
- Own the selected details snapshot at the page provider rather than in a row
  that can be replaced by a refresh. New rows do not close the dialog or replace
  the log being inspected.

### Verification

`bun run test -- src/features/usage-logs`: 12 files, 40 tests passed. New tests
first demonstrated excessive refetches, missing final catch-up, hidden-tab
requests, pointer-event blocking, disappearing details, early stream reopening,
and missing recovery after a transient manual-query failure. The updated tests
passed after implementation. The interaction fixture uses the real table,
columns, dialog, and pointer checks; only decorative third-party icon assets
are mocked to avoid their unsupported Node ESM directory imports.

`bun run typecheck` and `bun run build:check` passed. Oxlint and Oxfmt checks
passed for all 12 affected TypeScript/TSX files. No coverage percentage was
measured. An independent review identified two P2 cases; both were reproduced,
fixed, tested, and accepted in the bounded follow-up review.

The release review additionally exercised the production QueryCache error
handler. Usage requests now opt out of the global HTTP 500 error-page redirect
so their scheduled recovery remains mounted. A 500-to-200 regression test first
failed with an unexpected navigation, then passed with automatic data recovery;
an unrelated query retains its existing error-page navigation.

The user initialized the isolated local database and logged into the Codex
in-app browser manually. Backend port 3000 used the supplied API 180/180s and
web 60/180s settings; the frontend ran at `http://127.0.0.1:5173`.
A deterministic upstream bound to `127.0.0.1:18080` handled five real local
`POST /v1/chat/completions` requests using model `gpt-4.1-mini`.
All five returned HTTP 200 and the backend wrote exactly five consume logs.
The already-open browser advanced from one login log to six total logs without
a reload. A login-details dialog remained open with the original request ID
and content while these rows arrived. After closing it, the new list and updated
usage statistics were visible.

Runtime logs showed a list/stat refresh pair at 05:05:35 and the retained
catch-up at 05:05:45, rather than one pair for every completed model request.
The local servers and isolated fixtures remain available for manual testing.
The temporary relay token is limited to loopback, the test model, a small quota,
and a two-hour expiry; its value is not included in this document.

### Commit suggestion

`fix: keep live usage logs responsive`

### Pull request draft

**Title:** Keep live usage logs responsive and rate aware

**Summary / 功能說明:** 合併連續用量事件造成的背景查詢，並在 429 等待結束後補查，避免用量頁面因大量模型請求而反覆觸發管理 API 限流。背景更新期間可操作列表，已開啟的細節視窗保留選取的紀錄。

**Implementation Details:** 用量頁的事件排程控制器限制自動查詢間隔，保留查詢期間到達的新事件，並統一管理暫時錯誤與 Retry-After。SSE 重新建立前檢查相同截止時間。列表使用共用表格的背景互動選項，細節狀態移至頁面 Provider。

**Tests:** 40 項用量功能測試通過；型別、修改檔案的 lint/format 與正式建置通過；本機 5 次真實轉發請求均為 200，新增紀錄與已開啟視窗均完成瀏覽器驗證。覆蓋率未量測，未提供推估百分比。

**Potential Risks / Breaking Changes:** 用量自動更新最多延後約 10 秒；429 等待時間依伺服器回應。其他頁面維持既有表格行為。沒有後端、資料庫結構或正式環境設定變更。

**Checklist:**

- [x] 程式碼已通過 Linter / Formatter
- [x] 已新增或更新單元測試
- [x] 已新增或更新文件
