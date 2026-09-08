# Kimi K3 reasoning effort 研究筆記

調查日期：2026-07-19。本文記錄 Kimi 官方 API 契約、固定基底的程式碼診斷，以及本 feature branch 的實作結果。

官方資料只採用 Kimi API Platform；2026-07-19 的參數複核以 `platform.kimi.com` 現行文件為準。基底診斷固定在 `dev/v1.0.0-rc.21` 的 `3e55e6428489f0528681dec8f919d015d1cf26ef`，實作結果則採用本 feature branch 的原始碼。官方頁面可能持續更新，因此所有官方敘述都附上可點擊來源。

## 官方 API 契約

### K3 與 K2.6 的固定參數矩陣

Kimi 官方的現行模型參數表把 `temperature` 標記為固定參數；傳入不符合模型與思考模式的值會回傳錯誤，官方建議不要顯式傳入。K3 固定使用 `1.0`；K2.6 在思考模式固定使用 `1.0`，在非思考模式固定使用 `0.6`。來源：[模型參數參考的參數對照表與 temperature 說明](https://platform.kimi.com/docs/api/models-overview#temperature)、[Kimi K2.6 參數變動說明](https://platform.kimi.com/docs/guide/kimi-k2-6-quickstart#参数变动说明)、[Kimi K3 重要限制](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#重要限制)。

同一份官方參數表也把 K3 與 K2.6 的其他 sampling 欄位固定為 `top_p: 0.95`、`n: 1`、`presence_penalty: 0`、`frequency_penalty: 0`，且不適用 `top_k`。官方建議固定參數不要顯式傳入；new-api 前往 Moonshot 時省略 client 傳入的 `temperature`、`top_p`、`top_k` 與 `n`，而 client 明確傳入的兩個 penalty 會保持 non-nil 並正規化為官方固定值 `0`。來源：[模型參數參考](https://platform.kimi.com/docs/api/models-overview)、[Kimi K2.6 參數變動說明](https://platform.kimi.com/docs/guide/kimi-k2-6-quickstart#参数变动说明)、[Kimi K3 重要限制](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#重要限制)。

| 模型與模式 | 思考控制欄位 | `reasoning_effort` | 合法 `temperature` | 官方來源 |
| --- | --- | --- | --- | --- |
| `kimi-k3` | 不支援 K2.x 的 `thinking`；K3 始終推理 | request 頂層，可省略；目前唯一支援值是 `"max"` | 固定 `1.0` | [模型參數參考](https://platform.kimi.com/docs/api/models-overview)、[Kimi K3](https://platform.kimi.com/docs/guide/kimi-k3-quickstart)、[思考力度](https://platform.kimi.com/docs/guide/use-thinking-effort) |
| `kimi-k2.6`，`thinking.type` 省略或為 `"enabled"` | request 頂層 `thinking` 物件；`enabled` 是預設值 | 不支援 | 固定 `1.0` | [模型參數參考](https://platform.kimi.com/docs/api/models-overview#thinking)、[思考模式](https://platform.kimi.com/docs/guide/use-kimi-k2-thinking-model)、[Kimi K2.6](https://platform.kimi.com/docs/guide/kimi-k2-6-quickstart#参数变动说明) |
| `kimi-k2.6`，`thinking.type: "disabled"` | request 頂層 `thinking` 物件 | 不支援 | 固定 `0.6` | [模型參數參考](https://platform.kimi.com/docs/api/models-overview#temperature)、[K2.6 禁用思考範例](https://platform.kimi.com/docs/guide/kimi-k2-6-quickstart#k26-禁用思考能力示例) |

這個矩陣表示 `kimi-k2.6` 不能對所有請求一律使用 `temperature=1.0`：有效的 `thinking.type` 如果在送往上游前成為 `disabled`，temperature 就必須是 `0.6` 或直接省略；開啟思考時才是 `1.0`。`kimi-k3` 則不受 `thinking` 切換影響，任何非 `1.0` 的 temperature 都不符合官方契約。這是依官方參數矩陣作出的直接推論。來源：[模型參數參考](https://platform.kimi.com/docs/api/models-overview#temperature)。

錯誤 `invalid temperature: only 1 is allowed for this model` 表示上游判定該請求屬於 K3，或屬於啟用思考的 K2.6，且實際收到的 temperature 不是 `1.0`。這個訊息本身不能區分兩個模型；非思考 K2.6 的官方固定值是 `0.6`。這是依官方固定值與「其他值會報錯」規則對錯誤訊息作出的判讀。來源：[模型參數參考](https://platform.kimi.com/docs/api/models-overview#temperature)。

欄位位置也依模型分開：K3 使用 request 頂層 `reasoning_effort`，目前只接受 `"max"`；K2.6 不支援 `reasoning_effort`，它使用 request 頂層 `thinking` 物件，其中 `thinking.type` 接受 `"enabled"` 或 `"disabled"`，`thinking.keep` 可省略或設為 `"all"`。來源：[模型參數參考的 reasoning_effort 說明](https://platform.kimi.com/docs/api/models-overview#reasoning_effort)、[思考力度欄位說明](https://platform.kimi.com/docs/guide/use-thinking-effort#字段说明)、[思考模式的 K2.6 欄位說明](https://platform.kimi.com/docs/guide/use-kimi-k2-thinking-model#用-thinking-参数控制-kimi-k26-的思考行为)。

### 支援值與欄位位置

Kimi K3 永遠執行推理。目前 `reasoning_effort` 唯一支援的字串值是 `"max"`；欄位可省略，官方將 `max` 列為預設檔位。官方也說明未來才會增加較低的 effort level，因此 `low`、`medium`、`high`、`xhigh` 等值目前都不是 K3 的有效選項。來源：[模型參數參考](https://platform.kimi.com/docs/api/models-overview#reasoning_effort)、[思考力度](https://platform.kimi.com/docs/guide/use-thinking-effort)、[Kimi K3](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#思考力度)。

`reasoning_effort` 位於 request body 頂層，與 `model`、`messages` 同層。它不位於 `thinking` 物件內；K3 quickstart 明確要求不要對 K3 使用 K2.x 的 `thinking` 參數，而模型參數參考將 `thinking` 定義為 K2.x 專用。來源：[Kimi K3](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#思考力度)、[模型參數參考](https://platform.kimi.com/docs/api/models-overview#thinking)。

合法的最小 body 形狀如下：

```json
{
  "model": "kimi-k3",
  "messages": [
    {
      "role": "user",
      "content": "Derive the general formula for this sequence: 1, 4, 9, 25, 64, ..."
    }
  ],
  "reasoning_effort": "max"
}
```

這個欄位位置與值直接對應官方思考力度範例及 API reference 的 K3 request schema。來源：[思考力度 request 範例](https://platform.kimi.com/docs/guide/use-thinking-effort#设置推理力度)、[建立對話補全 request schema](https://platform.kimi.com/docs/api/chat)。

### Chat Completions 正式端點與完整請求

`platform.kimi.com` 現行文件提供的正式端點是 `POST https://api.moonshot.cn/v1/chat/completions`。完整 HTTP request 可寫成：

```http
POST /v1/chat/completions HTTP/1.1
Host: api.moonshot.cn
Authorization: Bearer $MOONSHOT_API_KEY
Content-Type: application/json

{
  "model": "kimi-k3",
  "messages": [
    {
      "role": "user",
      "content": "Derive the general formula for this sequence: 1, 4, 9, 25, 64, ..."
    }
  ],
  "reasoning_effort": "max"
}
```

官方 curl 範例使用相同 URL、Authorization header、Content-Type header 與 body。來源：[思考力度 curl 範例](https://platform.kimi.com/docs/guide/use-thinking-effort#设置推理力度)、[建立對話補全 API reference](https://platform.kimi.com/docs/api/chat)。

### reasoning_content 行為要求

| 情境 | K3 契約 | 官方來源 |
| --- | --- | --- |
| 一般單輪對話 | K3 永遠推理，但官方使用「may return」描述 `reasoning_content`，所以 client 應把它視為可能存在的 response 欄位。非 streaming response 中，它位於 `choices[0].message.reasoning_content`，與 `content` 同層。單輪請求沒有下一次 request，因此沒有歷史回傳動作。Structured Output 只解析最終 `message.content`，不要把 `reasoning_content` 當成結構化答案。 | [思考力度](https://platform.kimi.com/docs/guide/use-thinking-effort)、[建立對話補全 response schema](https://platform.kimi.com/docs/api/chat)、[Kimi K3 結構化輸出](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#结构化输出) |
| Streaming 對話 | `reasoning_content` 與最終答案 `content` 使用分開的 delta。官方 K3 quickstart 示範分別讀取兩者；Thinking Mode 文件指出 reasoning delta 先於 final-answer content。 | [Kimi K3 streaming 範例](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#流式输出)、[思考模式](https://platform.kimi.com/docs/guide/use-kimi-k2-thinking-model#从响应中读取-reasoning_content) |
| 工具呼叫 | 下一次 request 必須先放回 API 回傳的完整 assistant message，包含 `reasoning_content` 與 `tool_calls`，再加入每個 `role: "tool"` 結果；tool result 的 `tool_call_id` 必須對應 assistant message 內的 call id。不可只保留 `content`。 | [思考力度欄位說明](https://platform.kimi.com/docs/guide/use-thinking-effort#字段说明)、[Kimi K3 工具範例](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#自定义工具与-tool_choice) |
| 多輪對話 | Kimi API 本身無狀態，client 必須維護 `messages`。對 K3 而言，每一個歷史 assistant message 都要使用 API 回傳的完整物件原樣加入下一次 request，包含任何 `reasoning_content`；只重建 `{role, content}` 不符合 K3 的明文要求。 | [思考力度欄位說明](https://platform.kimi.com/docs/guide/use-thinking-effort#字段说明)、[Kimi K3 重要限制](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#重要限制)、[思考模式 FAQ](https://platform.kimi.com/docs/guide/use-kimi-k2-thinking-model#常见问题) |

Create Chat Completion API reference 另有一段使用 `kimi-k2.6` 的通用 multi-turn 範例，只追加 `{role, content}`。該範例的模型是 K2.6；K3 專頁、思考力度與模型參數參考都對 K3 明確要求完整 assistant message，因此 K3 應採用較具體的 K3 契約。來源：[建立對話補全 multi-turn section](https://platform.kimi.com/docs/api/chat)、[模型參數參考 FAQ](https://platform.kimi.com/docs/api/models-overview#常见问题)。

### kimi-k3-max 官方狀態

截至調查日期，官方 Model List 只列 `kimi-k3`，Create Chat Completion 的 K3 `model` enum 也只列 `kimi-k3`；官方文件沒有把 `kimi-k3-max` 定義為 model ID。`max` 是 top-level `reasoning_effort` 的值，並不是官方 model name 的一部分。來源：[模型列表](https://platform.kimi.com/docs/models)、[建立對話補全 K3 schema](https://platform.kimi.com/docs/api/chat)、[Kimi K3](https://platform.kimi.com/docs/guide/kimi-k3-quickstart#思考力度)。

因此，`kimi-k3-max` 在 new-api 範圍內只能標記為本地便利 suffix 語法。它表達的 upstream API 語意是 `model: "kimi-k3"` 加上 `reasoning_effort: "max"`；Kimi 官方端點沒有承諾接受帶 suffix 的 model ID。來源：[思考力度 request 範例](https://platform.kimi.com/docs/guide/use-thinking-effort#设置推理力度)、[模型列表](https://platform.kimi.com/docs/models)。

## `.cn` live API 探索性驗證

2026-07-19 以臨時 key 直接 POST `https://api.moonshot.cn/v1/chat/completions`，未經 new-api。這一節只記錄當日可觀察行為，不把未文件化結果視為官方契約。受控 prompt 為 `only just echo 'hi'`，每個已知模式各執行三次；response 的 `reasoning_content` 與 `usage.completion_tokens_details.reasoning_tokens` 都直接取自 Moonshot 原生 JSON。

| Request 變體 | HTTP | 三次 reasoning tokens | `reasoning_content` |
| --- | --- | --- | --- |
| 無 effort / thinking 參數 | 200 | 66、128、84 | 有 |
| `thinking: {"type":"disabled"}` | 200 | 0、0、0 | 無 |
| `reasoning_effort: "max"` | 200 | 43、83、47 | 有 |
| `reasoning_effort: "xhigh"` | 200 | 87、56、58 | 有 |
| `reasoning_effort: "high"` | 200 | 61、76、42 | 有 |
| `reasoning_effort: "medium"` | 200 | 69、123、90 | 有 |
| `reasoning_effort: "low"` | 200 | 58、80、73 | 有 |
| `reasoning_effort: "minimal"` | 200 | 74、47、54 | 有 |
| `reasoning_effort: "none"` | 200 | 0、0、0 | 無 |

另外，`maxx`、`banana`、`off` 與大寫 `MAX` 也都回傳 200、產生 `reasoning_content`，reasoning tokens 分別為 80、88、23、34。這表示當日 `.cn` endpoint 沒有拒絕未知 effort 字串；精確小寫 `none` 具有可重複的關閉思考效果，其他值則沒有呈現可辨識的 effort 強度順序。官方仍只承諾 `max`，因此 new-api 不應把 `xhigh`、`high`、`medium`、`low` 或 `minimal` 宣告為 K3 的正式 effort level。

## new-api 固定基底診斷

本節保留實作前的可重複診斷結果，所有「缺少」或「不支援」描述都只代表上述固定基底，不代表本 feature branch 的實作後狀態。

### Moonshot Chat Completions adaptor

Moonshot adaptor 會依 relay mode 組出 `/v1/chat/completions` URL；一般 Chat Completions 分支位於 [`relay/channel/moonshot/adaptor.go:49-73`](../../relay/channel/moonshot/adaptor.go#L49-L73)。

固定基底的 `ConvertOpenAIRequest` 只會把 `kimi-k2.6` 的非 1.0 temperature 正規化成 1.0，隨後原樣回傳 request。該版本沒有呼叫 reasoning suffix parser，也沒有 K3 專用分支，因此 `kimi-k3-max` 不會在 Moonshot adaptor 被轉成 `kimi-k3` 與 `reasoning_effort: "max"`。

Moonshot adaptor 的 Responses 轉換目前回傳 `not implemented`，所以這個 channel 現有的 Kimi Chat 路徑是 Chat Completions。來源：[`relay/channel/moonshot/adaptor.go:101-104`](../../relay/channel/moonshot/adaptor.go#L101-L104)。

### OpenAI adaptor

OpenAI Chat Completions adaptor 只在 upstream model 被判定為 `o1`、`o3`、`o4` 或 `gpt-5` family 時執行 `ParseOpenAIReasoningEffortFromModelSuffix`。`kimi-k3-max` 不符合這些 family gate，因此不會進入 suffix 解析區塊。來源：[`relay/channel/openai/adaptor.go:321-348`](../../relay/channel/openai/adaptor.go#L321-L348)、[`dto/openai_request.go:216-224`](../../dto/openai_request.go#L216-L224)。

如果 client 已經直接送 top-level `reasoning_effort`，OpenAI Chat Completions adaptor 沒有刪除該欄位；固定基底缺少的是從 `kimi-k3-max` 推導欄位的 Kimi path。欄位本身由 General OpenAI request DTO 提供，來源：[`dto/openai_request.go:29-44`](../../dto/openai_request.go#L29-L44)。

OpenAI Responses adaptor 會在 request model 或 origin model 上呼叫 `ParseOpenAIReasoningModelSuffix`，並把解析結果寫入 nested `reasoning.effort` 或 `reasoning.mode`。不過目前 parser 對非 GPT-5.6 model 使用的通用 OpenAI suffix 清單不含 `-max`，所以這條路徑也不會辨識 `kimi-k3-max`。來源：[`relay/channel/openai/adaptor.go:597-628`](../../relay/channel/openai/adaptor.go#L597-L628)、[`setting/reasoning/suffix.go:99-119`](../../setting/reasoning/suffix.go#L99-L119)。

### Request DTO

Chat Completions 使用的 `GeneralOpenAIRequest` 已有 top-level `ReasoningEffort string`，JSON key 是 `reasoning_effort`。這與 K3 官方欄位位置一致。來源：[`dto/openai_request.go:29-44`](../../dto/openai_request.go#L29-L44)。

Chat message DTO 可以表示 K3 歷史 assistant message 需要的 `reasoning_content` 與 `tool_calls`，兩者分別是 `ReasoningContent *string` 與 `ToolCalls json.RawMessage`。來源：[`dto/openai_request.go:289-298`](../../dto/openai_request.go#L289-L298)。

Responses API 使用不同 shape：`OpenAIResponsesRequest` 內是 `Reasoning *Reasoning`，其 JSON key 為 `reasoning`；`Reasoning` 物件再包含 `effort`、`summary`、`mode` 與 `context`。這個 nested shape 不等同於 K3 Chat Completions 的 top-level `reasoning_effort`。來源：[`dto/openai_request.go:842-872`](../../dto/openai_request.go#L842-L872)、[`dto/openai_request.go:967-972`](../../dto/openai_request.go#L967-L972)。

### Reasoning suffix parser

`setting/reasoning/suffix.go` 內有多組不同用途的 suffix。廣義 `EffortSuffixes` 包含 `-max`，但是 OpenAI parser 實際使用的 `OpenAIEffortSuffixes` 只有 `-high`、`-minimal`、`-low`、`-medium`、`-none` 與 `-xhigh`。因此只看到全域 `-max` 常數，不能代表 OpenAI Chat 或 Responses 已支援 `kimi-k3-max`。來源：[`setting/reasoning/suffix.go:10-16`](../../setting/reasoning/suffix.go#L10-L16)、[`setting/reasoning/suffix.go:99-119`](../../setting/reasoning/suffix.go#L99-L119)。

固定基底的 `max` 在 `ParseOpenAIReasoningModelSuffix` 特殊處理中只屬於 GPT-5.6 grammar；另外 DeepSeek V4 也有獨立的 `max` 行為。兩者都沒有涵蓋 Kimi model family。來源：[`setting/reasoning/suffix.go:107-119`](../../setting/reasoning/suffix.go#L107-L119)、[`setting/reasoning/suffix.go:193-213`](../../setting/reasoning/suffix.go#L193-L213)。

### Model list

固定基底的 Moonshot `ModelList` 只有 `kimi-k2.5`、數個已棄用的 `kimi-k2-*` model，沒有 `kimi-k3` 或 `kimi-k3-max`。

Moonshot adaptor 的 `GetModelList` 直接回傳上述清單，而 controller 初始化公開 model 集合時也會加入 `moonshot.ModelList`，所以固定基底的內建 model discovery 不會列出 K3。來源：[`relay/channel/moonshot/adaptor.go:129-134`](../../relay/channel/moonshot/adaptor.go#L129-L134)、[`controller/model.go:33-65`](../../controller/model.go#L33-L65)。

## 規格與固定基底對照

| 項目 | Kimi 官方契約 | new-api 固定基底狀態 |
| --- | --- | --- |
| 正式 model ID | `kimi-k3` | Moonshot 內建 model list 尚未列出 |
| reasoning effort | request 頂層 `reasoning_effort: "max"` | Chat DTO 可以表示並轉送此欄位 |
| `kimi-k3-max` | 沒有定義為官方 model ID | Moonshot 與 OpenAI Chat adaptor 都不會解析；它只能被視為 new-api 本地便利 suffix |
| K3 歷史 assistant message | 多輪與工具呼叫必須完整原樣回傳，包含 `reasoning_content` 與 `tool_calls` | Message DTO 可以表示這兩個欄位 |

## 本分支實作結果

Moonshot Chat Completions adaptor 現在提供兩個 K3 model 形式：普通 `kimi-k3` 在 client 沒有顯式提供 effort 時，依 2026-07-19 live API 相容性驗證補上 top-level `reasoning_effort: "none"`；本地便利名稱 `kimi-k3-max` 則轉成 `model: "kimi-k3"` 加上 top-level `reasoning_effort: "max"`。`none` 仍不是官方文件列出的 K3 effort，這裡屬於 new-api 的 Moonshot provider-specific 行為。兩種形式都同步維持 request model、`RelayInfo.UpstreamModelName` 與 `RelayInfo.ReasoningEffort`；已取消 `kimi-k3-none` alias。來源：[`relay/channel/moonshot/adaptor.go`](../../relay/channel/moonshot/adaptor.go)、[Kimi 模型參數參考](https://platform.kimi.com/docs/api/models-overview#reasoning_effort)。

顯式傳入 top-level `reasoning_effort` 的 request 仍會保留原本 payload，因此官方未來新增 effort 後可以先使用 `model: "kimi-k3"` 加上新值；只有新 suffix alias 需要在 Moonshot adaptor 與 model list 明確加入。`kimi-k3-high` 等目前未支援的 suffix 仍保持原樣，不會被提前轉換；OpenAI channel 與其他 provider 的 suffix parser 都沒有修改。相關 public adaptor regression coverage 位於 [`relay/channel/moonshot/kimi_k3_reasoning_effort_test.go`](../../relay/channel/moonshot/kimi_k3_reasoning_effort_test.go)。

Moonshot adaptor 另加入本地便利名稱 `kimi-k2.6-thinking`，轉成官方 `model: "kimi-k2.6"` 與 `thinking: {"type":"enabled"}`；即使共用 model mapping 已先把 upstream model 改成 `kimi-k2.6`，保留的 `OriginModelName` 仍會辨識這個 thinking alias。普通 `kimi-k2.6` 在 client 沒有顯式提供 `thinking` 時則補上 `thinking: {"type":"disabled"}`；client 顯式提供的合法 `thinking` 物件仍會保留。Thinking alias 對應的官方固定 temperature 是 `1.0`，普通 disabled 模式的固定值是 `0.6`，不能套用全模式一律 `1.0` 的規則。官方 K2.6 thinking 欄位契約與 temperature 差異來源：[思考模式](https://platform.kimi.com/docs/guide/use-kimi-k2-thinking-model)、[Kimi K2.6 參數變動說明](https://platform.kimi.com/docs/guide/kimi-k2-6-quickstart#参数变动说明)、[模型參數參考](https://platform.kimi.com/docs/api/models-overview#temperature)。Regression coverage 位於 [`relay/channel/moonshot/adaptor_test.go`](../../relay/channel/moonshot/adaptor_test.go) 與 [`relay/channel/moonshot/kimi_k26_thinking_test.go`](../../relay/channel/moonshot/kimi_k26_thinking_test.go)。

對官方 `kimi-k3` 與 `kimi-k2.6`，Moonshot adaptor 現在會省略 client 傳入的 `temperature`、`top_p`、`top_k` 與 `n`；顯式 penalty 則保持 non-nil 並正規化為官方固定值 `0`。這保留 K3 固定 `temperature=1.0` 的契約，也讓普通 K2.6 依最終 `thinking.type: "disabled"` 由 Moonshot 自動採用 `temperature=0.6`，不會在 adaptor 階段過早寫死錯誤模式的值。Regression coverage 位於 [`relay/channel/moonshot/adaptor_test.go`](../../relay/channel/moonshot/adaptor_test.go)、[`relay/channel/moonshot/kimi_k3_reasoning_effort_test.go`](../../relay/channel/moonshot/kimi_k3_reasoning_effort_test.go) 與 [`relay/channel/moonshot/kimi_k26_thinking_test.go`](../../relay/channel/moonshot/kimi_k26_thinking_test.go)。

既有 production channel 可以保留 parameter override；它在 adaptor conversion 與 JSON marshal 之後執行，因此無條件的 `thinking: {"type":"disabled"}` 管理員規則仍具有最後優先權。需要在同一 channel 讓普通 `kimi-k2.6` 關閉思考、`kimi-k2.6-thinking` 保持思考時，既有 disabled operation 必須增加 `original_model` 條件：

```json
{
  "path": "thinking",
  "mode": "set",
  "value": {
    "type": "disabled"
  },
  "conditions": [
    {
      "path": "original_model",
      "mode": "full",
      "value": "kimi-k2.6"
    }
  ]
}
```

`original_model` 保留 client 指定的名稱，`upstream_model` 則是轉換後的 provider model；這個條件只會命中普通 `kimi-k2.6`，不會蓋掉 `kimi-k2.6-thinking` 產生的 enabled payload。執行順序來源：[`relay/compatible_handler.go`](../../relay/compatible_handler.go)、[`relay/common/override.go`](../../relay/common/override.go)。

Moonshot model discovery 現在列出官方 `kimi-k3`、`kimi-k2.6`，以及 new-api 本地便利名稱 `kimi-k3-max`、`kimi-k2.6-thinking`；`kimi-k3-none` 已移除。官方 model list 沒有定義這些帶 suffix 名稱；它們只屬於 new-api request convenience。來源：[`relay/channel/moonshot/constants.go`](../../relay/channel/moonshot/constants.go)、[Kimi 模型列表](https://platform.kimi.com/docs/models)。
