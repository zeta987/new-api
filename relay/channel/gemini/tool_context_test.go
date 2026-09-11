package gemini

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiToolCombinationPostOverride(t *testing.T) {
	for _, mode := range []string{"AUTO", "NONE", "ANY", "VALIDATED", ""} {
		t.Run(mode, func(t *testing.T) {
			_, _, info := geminiContextFixture(false)
			a := &Adaptor{}
			a.Init(info)
			body := `{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"generationConfig":{},"tools":[{"functionDeclarations":[{"name":"lookup"}]},{"codeExecution":{}}],"toolConfig":{"functionCallingConfig":{"mode":"` + mode + `","allowedFunctionNames":["lookup"]}},"futureField":{"keep":true}}`
			got, err := a.PreparePostOverrideRequest([]byte(body))
			require.NoError(t, err)
			var parsed struct {
				Tools       []json.RawMessage `json:"tools"`
				ToolConfig  dto.ToolConfig    `json:"toolConfig"`
				FutureField map[string]bool   `json:"futureField"`
			}
			require.NoError(t, common.Unmarshal(got, &parsed))
			require.NotNil(t, parsed.ToolConfig.IncludeServerSideToolInvocations)
			assert.True(t, *parsed.ToolConfig.IncludeServerSideToolInvocations)
			want := mode
			if want == "AUTO" || want == "" {
				want = "VALIDATED"
			}
			assert.Equal(t, dto.FunctionCallingConfigMode(want), parsed.ToolConfig.FunctionCallingConfig.Mode)
			assert.Equal(t, []string{"lookup"}, parsed.ToolConfig.FunctionCallingConfig.AllowedFunctionNames)
			assert.Len(t, parsed.Tools, 2)
			assert.True(t, parsed.FutureField["keep"])
		})
	}
}

func TestGeminiToolContextNonstreamRoundTrip(t *testing.T) {
	isolateGeminiContexts(t)
	c, recorder, info := geminiContextFixture(false)
	request := &dto.GeneralOpenAIRequest{Model: "gemini-3.8-flash", Messages: []dto.Message{{Role: "user", Content: "search and calculate"}}}
	a := &Adaptor{}
	a.Init(info)
	converted, err := a.ConvertOpenAIRequest(c, info, request)
	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	var withTools map[string]json.RawMessage
	require.NoError(t, common.Unmarshal(body, &withTools))
	withTools["tools"] = json.RawMessage(`[{"functionDeclarations":[{"name":"lookup"}]},{"googleSearch":{}},{"codeExecution":{}}]`)
	body, err = common.Marshal(withTools)
	require.NoError(t, err)
	_, err = a.PreparePostOverrideRequest(body)
	require.NoError(t, err)
	native := `{"role":"model","parts":[{"toolCall":{"toolType":"GOOGLE_SEARCH_WEB","id":"s1","args":{"queries":["test"]},"future":1},"thoughtSignature":"search-signature","futurePart":{"keep":true}},{"toolResponse":{"toolType":"GOOGLE_SEARCH_WEB","id":"s1","response":{"results":[1]}},"thoughtSignature":"result-signature"},{"executableCode":{"id":"e1","language":"PYTHON","code":"print(2)"},"thoughtSignature":"code-signature"},{"codeExecutionResult":{"id":"e1","outcome":"OUTCOME_OK","output":"2"}},{"functionCall":{"id":"f1","name":"lookup","args":{"b":2,"a":1}},"thoughtSignature":"function-signature"}]}`
	response := `{"candidates":[{"index":0,"content":` + native + `,"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":3,"totalTokenCount":5}}`
	_, apiErr := GeminiChatHandler(c, info, &http.Response{Body: io.NopCloser(strings.NewReader(response))})
	require.Nil(t, apiErr)
	var chat dto.OpenAITextResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &chat))
	require.Len(t, chat.Choices, 1)
	assistant := chat.Choices[0].Message
	// Cherry's custom provider path drops signatures and reserializes arguments.
	assistant.ToolCalls = json.RawMessage(`[{"id":"f1","type":"function","function":{"name":"lookup","arguments":"{\"a\":1,\"b\":2}"},"extra_content":{"google":{"thought_signature":"skip_thought_signature_validator"}}}]`)
	next := &dto.GeneralOpenAIRequest{Model: request.Model, Messages: append(append([]dto.Message{}, request.Messages...), assistant, dto.Message{Role: "tool", ToolCallId: "f1", Content: `{"ok":true}`})}
	c2, _, info2 := geminiContextFixture(false)
	a2 := &Adaptor{}
	a2.Init(info2)
	converted, err = a2.ConvertOpenAIRequest(c2, info2, next)
	require.NoError(t, err)
	body, err = common.Marshal(converted)
	require.NoError(t, err)
	got, err := a2.PreparePostOverrideRequest(body)
	require.NoError(t, err)
	var restored struct {
		Contents   []json.RawMessage `json:"contents"`
		ToolConfig dto.ToolConfig    `json:"toolConfig"`
	}
	require.NoError(t, common.Unmarshal(got, &restored))
	require.Len(t, restored.Contents, 3)
	assert.JSONEq(t, native, string(restored.Contents[1]))
	assert.Contains(t, string(restored.Contents[2]), `"id":"f1"`)
	require.NotNil(t, restored.ToolConfig.IncludeServerSideToolInvocations)
	assert.True(t, *restored.ToolConfig.IncludeServerSideToolInvocations)

	for _, change := range []string{"model", "prior_user", "system"} {
		t.Run("override_"+change, func(t *testing.T) {
			var overridden map[string]json.RawMessage
			require.NoError(t, common.Unmarshal(body, &overridden))
			var messages []json.RawMessage
			require.NoError(t, common.Unmarshal(overridden["contents"], &messages))
			switch change {
			case "model":
				messages[1] = json.RawMessage(`{"role":"model","parts":[{"text":"admin replacement"}]}`)
			case "prior_user":
				messages[0] = json.RawMessage(`{"role":"user","parts":[{"text":"admin replacement"}]}`)
			case "system":
				overridden["systemInstruction"] = json.RawMessage(`{"parts":[{"text":"admin replacement"}]}`)
			}
			overridden["contents"], err = common.Marshal(messages)
			require.NoError(t, err)
			changed, marshalErr := common.Marshal(overridden)
			require.NoError(t, marshalErr)
			got, prepareErr := a2.PreparePostOverrideRequest(changed)
			require.NoError(t, prepareErr)
			assert.JSONEq(t, string(changed), string(got))
			assert.NotContains(t, string(got), "search-signature")
		})
	}

	for _, scope := range []string{"user", "token", "channel", "model", "history", "arguments", "tool_id"} {
		t.Run("isolated_"+scope, func(t *testing.T) {
			isolatedRequest, copyErr := common.DeepCopy(next)
			require.NoError(t, copyErr)
			ctx, _, isolatedInfo := geminiContextFixture(false)
			switch scope {
			case "user":
				isolatedInfo.UserId++
			case "token":
				isolatedInfo.TokenId++
			case "channel":
				isolatedInfo.ChannelId++
			case "model":
				isolatedInfo.UpstreamModelName = "gemini-3.8-pro"
			case "history":
				isolatedRequest.Messages[0].SetStringContent("a different conversation")
			case "arguments":
				isolatedRequest.Messages[1].ToolCalls = json.RawMessage(`[{"id":"f1","type":"function","function":{"name":"lookup","arguments":"{\"a\":9,\"b\":2}"}}]`)
			case "tool_id":
				isolatedRequest.Messages[1].ToolCalls = json.RawMessage(`[{"id":"different","type":"function","function":{"name":"lookup","arguments":"{\"a\":1,\"b\":2}"}}]`)
			}
			isolatedAdaptor := &Adaptor{}
			isolatedAdaptor.Init(isolatedInfo)
			value, convertErr := isolatedAdaptor.ConvertOpenAIRequest(ctx, isolatedInfo, isolatedRequest)
			require.NoError(t, convertErr)
			encoded, marshalErr := common.Marshal(value)
			require.NoError(t, marshalErr)
			got, prepareErr := isolatedAdaptor.PreparePostOverrideRequest(encoded)
			require.NoError(t, prepareErr)
			assert.JSONEq(t, string(encoded), string(got))
			assert.NotContains(t, string(got), "search-signature")
		})
	}
}

func TestGeminiToolContextStreamingRoundTrip(t *testing.T) {
	for _, test := range []struct {
		name                            string
		functionCall, thinkingToContent bool
	}{
		{"native_only_result", false, false}, {"function_result_continuation", true, false}, {"thinking_to_content", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			isolateGeminiContexts(t)
			oldTimeout := constant.StreamingTimeout
			constant.StreamingTimeout = 30
			t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
			c, recorder, info := geminiContextFixture(true)
			info.ChannelSetting.ThinkingToContent = test.thinkingToContent
			info.ThinkingContentInfo.IsFirstThinkingContent = true
			request := &dto.GeneralOpenAIRequest{Model: "gemini-3.8-flash", Messages: []dto.Message{{Role: "system", Content: "test system"}, {Role: "user", Content: "search and calculate"}}}
			a, body := prepareGeminiContextRequest(t, c, info, request, true)
			_, err := a.PreparePostOverrideRequest(body)
			require.NoError(t, err)
			parts := []string{
				`{"toolCall":{"toolType":"GOOGLE_SEARCH_WEB","id":"search","args":{"queries":["test"]}},"thoughtSignature":"s1"}`,
				`{"toolResponse":{"toolType":"GOOGLE_SEARCH_WEB","id":"search","response":{"result":"found"}},"thoughtSignature":"s2","futurePart":{"preserve":true}}`,
				`{"executableCode":{"language":"PYTHON","code":"print(2)","id":"code"},"thoughtSignature":"s3"}`,
				`{"codeExecutionResult":{"outcome":"OUTCOME_OK","output":"2","id":"code"},"thoughtSignature":"s4"}`,
			}
			if test.thinkingToContent {
				parts = append([]string{`{"text":"Let me think.","thought":true,"thoughtSignature":"thinking-signature"}`}, parts...)
			}
			if test.functionCall {
				parts = append(parts, `{"functionCall":{"id":"function","name":"lookup","args":{"b":2,"a":1}},"thoughtSignature":"s5"}`)
			} else {
				parts = append(parts, `{"text":"Final answer.","thoughtSignature":"s5"}`)
			}
			var stream strings.Builder
			for i, part := range parts {
				finish := ""
				if i == len(parts)-1 {
					finish = `,"finishReason":"STOP"`
				}
				stream.WriteString(`data: {"candidates":[{"index":0,"content":{"role":"model","parts":[` + part + `]} ` + finish + `}]}` + "\n\n")
			}
			stream.WriteString("data: [DONE]\n\n")
			_, apiErr := GeminiChatStreamHandler(c, info, &http.Response{Body: io.NopCloser(strings.NewReader(stream.String()))})
			require.Nil(t, apiErr)
			assistant := readGeminiChatStream(t, recorder.Body.String())
			require.NotEmpty(t, assistant.StringContent())
			if test.thinkingToContent {
				assert.Contains(t, assistant.StringContent(), "<think>\nLet me think.\n</think>\n")
			}
			nextMessages := append(append([]dto.Message{}, request.Messages...), assistant)
			if test.functionCall {
				calls := assistant.ParseToolCalls()
				require.Len(t, calls, 1)
				assert.Equal(t, "function", calls[0].ID)
				assert.Equal(t, "lookup", calls[0].Function.Name)
				assert.JSONEq(t, `{"a":1,"b":2}`, calls[0].Function.Arguments)
				nextMessages = append(nextMessages, dto.Message{Role: "tool", ToolCallId: "function", Content: `{"ok":true}`})
			} else {
				nextMessages = append(nextMessages, dto.Message{Role: "user", Content: "hello"})
			}
			c2, _, info2 := geminiContextFixture(false)
			a2, nextBody := prepareGeminiContextRequest(t, c2, info2, &dto.GeneralOpenAIRequest{Model: request.Model, Messages: nextMessages}, false)
			got, err := a2.PreparePostOverrideRequest(nextBody)
			require.NoError(t, err)
			var restored struct {
				Contents   []json.RawMessage `json:"contents"`
				ToolConfig dto.ToolConfig    `json:"toolConfig"`
				Tools      []json.RawMessage `json:"tools"`
			}
			require.NoError(t, common.Unmarshal(got, &restored))
			require.Len(t, restored.Contents, 3)
			assert.JSONEq(t, `{"role":"model","parts":[`+strings.Join(parts, ",")+`]}`, string(restored.Contents[1]))
			require.NotNil(t, restored.ToolConfig.IncludeServerSideToolInvocations)
			assert.True(t, *restored.ToolConfig.IncludeServerSideToolInvocations)
			assert.Empty(t, restored.Tools, "restoring history must not re-enable new tools")
		})
	}
}

func TestGeminiToolContextBoundsAndExpiry(t *testing.T) {
	now := time.Unix(100, 0)
	one, two, three := sha256.Sum256([]byte("one")), sha256.Sum256([]byte("two")), sha256.Sum256([]byte("three"))
	cache := newGeminiContextCache(2, 9, time.Minute)
	input := []byte("1111")
	cache.put(one, input, now)
	input[0] = 'x'
	assert.Equal(t, "1111", string(cache.get(one, now)))
	copy := cache.get(one, now)
	copy[0] = 'x'
	assert.Equal(t, "1111", string(cache.get(one, now)))
	cache.put(two, []byte("2222"), now)
	assert.NotNil(t, cache.get(one, now))
	cache.put(three, []byte("3333"), now)
	assert.Nil(t, cache.get(two, now), "LRU entry is evicted")
	assert.Equal(t, "1111", string(cache.get(one, now.Add(50*time.Second))))
	assert.Equal(t, "1111", string(cache.get(one, now.Add(100*time.Second))), "reads extend idle TTL")
	assert.Nil(t, cache.get(one, now.Add(161*time.Second)))

	bytesOnly := newGeminiContextCache(10, 7, time.Minute)
	bytesOnly.put(one, []byte("1111"), now)
	bytesOnly.put(two, []byte("2222"), now)
	assert.Nil(t, bytesOnly.get(one, now), "byte limit applies even below entry capacity")
	bytesOnly.put(three, []byte("too large"), now)
	assert.Nil(t, bytesOnly.get(three, now))
	assert.Equal(t, "2222", string(bytesOnly.get(two, now)))
	bytesOnly.put(two, []byte("same projection but different native context"), now)
	assert.Equal(t, "2222", string(bytesOnly.get(two, now)), "oversized writes do not replace valid entries")
	bytesOnly.put(two, []byte("3333"), now)
	assert.Nil(t, bytesOnly.get(two, now), "ambiguous native contexts must never be guessed")
	bytesOnly.put(two, []byte("2222"), now)
	assert.Nil(t, bytesOnly.get(two, now), "ambiguity remains until expiry")

	state := &geminiToolContext{enabled: true, cache: cache, candidates: make(map[int]*geminiContextCandidate)}
	chunk := `{"candidates":[{"content":{"parts":[{"text":"` + strings.Repeat("x", geminiContextMaxBytes/2) + `"}]}}]}`
	state.capture([]byte(chunk))
	require.False(t, state.failed)
	state.capture([]byte(chunk))
	assert.True(t, state.failed, "stream limit applies during accumulation")
	assert.Nil(t, state.candidates, "oversized accumulation is released immediately")
	state.capture([]byte(`{"candidates":[{"content":{"parts":[{"text":"end"}]},"finishReason":"STOP"}]}`))
	assert.Nil(t, state.candidates, "later chunks cannot revive a truncated context")

	state = &geminiToolContext{enabled: true, cache: cache, candidates: make(map[int]*geminiContextCandidate)}
	state.capture([]byte(`{"candidates":[{"content":{"parts":[{"text":"partial","thoughtSignature":"private"}]}}]}`))
	state.project(&dto.ChatCompletionsStreamResponse{Choices: []dto.ChatCompletionsStreamResponseChoice{{Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: ptrGeminiContext("partial")}}}})
	state.save()
	key, ok := geminiHistoryHash(state.prefix, dto.Message{Role: "assistant", Content: "partial"})
	require.True(t, ok)
	assert.Nil(t, cache.get(key, time.Now()), "incomplete streams do not enter the cache")
}

func TestGeminiToolCombinationUnchangedRoutes(t *testing.T) {
	for _, item := range []struct{ model, tools string }{
		{"gemini-2.5-flash", `[{"functionDeclarations":[{"name":"lookup"}]},{"codeExecution":{}}]`},
		{"gemini-3.8-flash", `[{"functionDeclarations":[{"name":"lookup"}]}]`},
		{"gemini-3.8-flash", `[{"codeExecution":{}}]`},
		{"gemini-3.8-flash", `[]`},
	} {
		_, _, info := geminiContextFixture(false)
		info.UpstreamModelName = item.model
		a := &Adaptor{}
		a.Init(info)
		body := `{"contents":[],"tools":` + item.tools + `,"toolConfig":{"functionCallingConfig":{"mode":"AUTO"}}}`
		got, err := a.PreparePostOverrideRequest([]byte(body))
		require.NoError(t, err)
		assert.Equal(t, body, string(got))
	}
}

func prepareGeminiContextRequest(t *testing.T, c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, mixed bool) (*Adaptor, []byte) {
	t.Helper()
	a := &Adaptor{}
	a.Init(info)
	converted, err := a.ConvertOpenAIRequest(c, info, request)
	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	if mixed {
		var fields map[string]json.RawMessage
		require.NoError(t, common.Unmarshal(body, &fields))
		fields["tools"] = json.RawMessage(`[{"functionDeclarations":[{"name":"lookup"}]},{"googleSearch":{}},{"codeExecution":{}}]`)
		fields["toolConfig"] = json.RawMessage(`{"functionCallingConfig":{"mode":"AUTO"}}`)
		body, err = common.Marshal(fields)
		require.NoError(t, err)
	}
	return a, body
}

func readGeminiChatStream(t *testing.T, stream string) dto.Message {
	t.Helper()
	message := dto.Message{Role: "assistant"}
	var content strings.Builder
	calls := make(map[int]dto.ToolCallResponse)
	for line := range strings.SplitSeq(stream, "\n") {
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || data == "[DONE]" {
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(data, &chunk))
		for _, choice := range chunk.Choices {
			if choice.Index != 0 {
				continue
			}
			if choice.Delta.Content != nil {
				content.WriteString(*choice.Delta.Content)
			}
			for _, delta := range choice.Delta.ToolCalls {
				require.NotNil(t, delta.Index)
				call := calls[*delta.Index]
				if delta.ID != "" {
					call.ID = delta.ID
				}
				if delta.Type != nil {
					call.Type = delta.Type
				}
				call.Function.Name += delta.Function.Name
				call.Function.Arguments += delta.Function.Arguments
				calls[*delta.Index] = call
			}
		}
	}
	message.SetStringContent(content.String())
	var tools []dto.ToolCallResponse
	for i := range len(calls) {
		tools = append(tools, calls[i])
	}
	if len(tools) > 0 {
		message.SetToolCalls(tools)
	}
	return message
}

func isolateGeminiContexts(t *testing.T) {
	t.Helper()
	previous := geminiContexts
	geminiContexts = newGeminiContextCache(geminiContextCacheEntries, geminiContextCacheBytes, geminiContextTTL)
	t.Cleanup(func() { geminiContexts = previous })
}

func ptrGeminiContext[T any](value T) *T { return &value }

type geminiDeadlineRecorder struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (w *geminiDeadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	w.deadlines = append(w.deadlines, deadline)
	return nil
}

func TestGeminiContextWriterForwardsWriteDeadline(t *testing.T) {
	recorder := &geminiDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	c, _ := gin.CreateTestContext(recorder)
	c.Writer = &geminiContextWriter{ResponseWriter: c.Writer, state: &geminiToolContext{}}
	controller := http.NewResponseController(c.Writer)
	deadline := time.Unix(1234, 0)
	require.NoError(t, controller.SetWriteDeadline(deadline))
	require.NoError(t, controller.SetWriteDeadline(time.Time{}))
	assert.Equal(t, []time.Time{deadline, {}}, recorder.deadlines)
}

func TestGeminiToolContextRejectsStaleRetryScope(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, target := range []string{"gemini-2.5", "vertex"} {
			t.Run(fmt.Sprintf("%s_stream_%t", target, stream), func(t *testing.T) {
				isolateGeminiContexts(t)
				oldTimeout := constant.StreamingTimeout
				constant.StreamingTimeout = 30
				t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
				c, _, info := geminiContextFixture(stream)
				request := &dto.GeneralOpenAIRequest{Model: info.UpstreamModelName, Messages: []dto.Message{{Role: "user", Content: "initial attempt"}}}
				adaptor, body := prepareGeminiContextRequest(t, c, info, request, true)
				_, err := adaptor.PreparePostOverrideRequest(body)
				require.NoError(t, err)
				stale := adaptor.toolContext
				require.NotNil(t, stale)
				require.True(t, stale.enabled)
				// The controller reuses this gin.Context and mutates RelayInfo for
				// the next channel after the first upstream attempt has failed.
				info.ChannelId++
				if target == "gemini-2.5" {
					info.UpstreamModelName = "gemini-2.5-flash"
				} else {
					info.ChannelType = constant.ChannelTypeVertexAi
				}
				response := `{"candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"retry success","thoughtSignature":"retry-signature"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":2,"totalTokenCount":3}}`
				var apiErr *types.NewAPIError
				if stream {
					_, apiErr = GeminiChatStreamHandler(c, info, &http.Response{Body: io.NopCloser(strings.NewReader("data: " + response + "\n\ndata: [DONE]\n\n"))})
				} else {
					_, apiErr = GeminiChatHandler(c, info, &http.Response{Body: io.NopCloser(strings.NewReader(response))})
				}
				require.Nil(t, apiErr)
				key, ok := geminiHistoryHash(stale.prefix, dto.Message{Role: "assistant", Content: "retry success"})
				require.True(t, ok)
				assert.Nil(t, stale.cache.get(key, time.Now()), "retry responses must never enter the first attempt's scope")
				assert.Empty(t, stale.candidates, "shared handlers must not capture into stale attempt state")
			})
		}
	}
}

func geminiContextFixture(stream bool) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{UserId: 117, TokenId: 19, RelayFormat: types.RelayFormatOpenAI, RelayMode: relayconstant.RelayModeChatCompletions, OriginModelName: "gemini-3.8-flash", IsStream: stream, StartTime: time.Now(), ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 71, UpstreamModelName: "gemini-3.8-flash"}}
	return c, recorder, info
}
