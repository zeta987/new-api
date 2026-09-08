# Kimi Formula 官方工具使用指南

本指南說明如何在 new-api 原生 Moonshot 渠道（channel type `25`）啟用 Kimi 官方 Formula 工具 `web-search`、`fetch` 與 `code-runner`。適用範圍只有 `/v1/chat/completions`，Playground 可直接使用。Claude 端點、Responses 端點與純 body passthrough 都不會觸發，Kimi 渠道也必須排除在 Chat 轉 Responses 的 conversion policy 之外。官方文件請見 [使用官方工具](https://platform.kimi.ai/docs/guide/use-official-tools) 與 [思考模型](https://platform.kimi.com/docs/guide/use-thinking-models)。

## 運作方式與限制

渠道參數覆蓋注入閘道專用欄位 `kimi_tools`，Moonshot adaptor 消費後即移除，上游永遠看不到。真正的 function declarations 由閘道以同一渠道的 origin 與 API key 向官方 Formula 路徑查詢：先 GET tools 取得 schema，模型回傳 `tool_calls` 後逐一 POST fibers 執行，再把完整 assistant 訊息（含 `reasoning_content`）與 `role: tool` 結果附回，重複直到產生最終答案。

```
client ──▶ /v1/chat/completions（kimi_tools 注入後被消費）
  ├─ GET  {origin}/v1/formulas/moonshot/<name>:latest/tools   取得工具 schema
  ├─ POST {origin}/v1/chat/completions（stream=false）         模型回傳 tool_calls
  ├─ POST {origin}/v1/formulas/moonshot/<name>:latest/fibers   逐一執行
  ├─ 附回完整 assistant 訊息與 role:tool 結果
  └─ 重複直到 finish_reason 不是 tool_calls
```

同源 `/v1` 必須真的提供 Formula 端點，第三方轉發或自訂 base prefix 不保證可用；中國區與國際區的帳號和 key 不能混用。上限為 8 輪 Chat、16 次工具執行，受管請求體 8 MiB，工具結果總量亦有邊界。同一批 `tool_calls` 混入客戶端自訂工具時，整批原樣回給客戶端，由客戶端自行處理。任何錯誤、非成功狀態或取消都直接結束，迴圈後不會自動重試。客戶端指定 `stream: true` 時，輸出的是緩衝後的最終答案 SSE 幀，並非逐 token 推送。整體迴圈受 `STREAMING_TIMEOUT` 秒數限制，預設 300，非正值回退為 300。

## 參數覆蓋設定

三個工具全開，並在渠道測試時刪除既存標記：

```json
{
  "operations": [
    {
      "path": "kimi_tools",
      "mode": "set",
      "value": ["web-search", "fetch", "code-runner"]
    },
    {
      "path": "kimi_tools",
      "mode": "delete",
      "conditions": [
        { "path": "is_channel_test", "mode": "full", "value": true }
      ]
    }
  ]
}
```

關鍵字變體請直接使用 [關鍵字參數覆蓋](kimi-formula-keywords.json)。它依最後一則 user 訊息判斷：含「上網」「搜尋」「搜索」時啟用 web-search 與 fetch，含「代碼」「程式碼」「求解」「計算」「驗證」「驗算」時啟用 code-runner，兩組同時命中會合併而非互相覆蓋。

## 計費

每一輪內部 Chat 的 token 各自結算，最終 usage 為所有輪次的總和，各輪可能落在不同的價格 tier。工具費用在 `tool_price_setting.prices` 設定，值為以 `kimi_web_search`、`kimi_fetch`、`kimi_code_runner` 為鍵的物件，單位為 USD 每 1000 次呼叫。鍵缺席或為 0 表示閘道不加收，並不代表供應商免費。只有 Fiber 回傳 `succeeded` 的呼叫才計費，嘗試次數另外記錄。迴圈中途失敗時，已消耗的輪次 token 與已成功的工具仍會記錄並結算。
