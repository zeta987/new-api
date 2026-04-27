package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChatCompletionsRequestToResponsesRequestPreservesBuiltInToolShape(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-5.1",
		"messages": [
			{"role": "user", "content": "search the docs"}
		],
		"tools": [
			{
				"type": "web_search_preview",
				"filters": {
					"allowed_domains": ["example.com"]
				}
			},
			{
				"type": "code_interpreter",
				"container": {
					"type": "auto"
				}
			}
		]
	}`)

	var req dto.GeneralOpenAIRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	out, err := ChatCompletionsRequestToResponsesRequest(&req)
	require.NoError(t, err)

	tools := gjson.ParseBytes(out.Tools)
	require.Equal(t, "web_search_preview", tools.Get("0.type").String())
	require.False(t, tools.Get("0.function").Exists())
	require.Equal(t, "example.com", tools.Get("0.filters.allowed_domains.0").String())
	require.Equal(t, "code_interpreter", tools.Get("1.type").String())
	require.False(t, tools.Get("1.function").Exists())
	require.Equal(t, "auto", tools.Get("1.container.type").String())
}

func TestResponsesResponseToChatCompletionsResponsePreservesReasoningSummary(t *testing.T) {
	raw := []byte(`{
		"id": "resp_123",
		"created_at": 1710000000,
		"model": "gpt-5.1",
		"output": [
			{
				"type": "reasoning",
				"id": "rs_1",
				"summary": [
					{"type": "summary_text", "text": "first thought"},
					{"type": "other", "text": "ignored"},
					{"type": "summary_text", "text": "second thought"}
				]
			},
			{
				"type": "message",
				"id": "msg_1",
				"role": "assistant",
				"content": [
					{"type": "output_text", "text": "final answer"}
				]
			}
		]
	}`)

	var resp dto.OpenAIResponsesResponse
	require.NoError(t, common.Unmarshal(raw, &resp))

	out, _, err := ResponsesResponseToChatCompletionsResponse(&resp, "chatcmpl_123")
	require.NoError(t, err)
	require.Len(t, out.Choices, 1)
	require.Equal(t, "final answer", out.Choices[0].Message.Content)
	require.Equal(t, "first thought\n\nsecond thought", out.Choices[0].Message.ReasoningContent)
}
