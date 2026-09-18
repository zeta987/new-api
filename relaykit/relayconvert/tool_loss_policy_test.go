package relayconvert

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiMixedNativeAndFunctionToolsPreservePartsSignatureAndID(t *testing.T) {
	t.Parallel()

	var part dto.GeminiPart
	require.NoError(t, kitutil.Unmarshal([]byte(`{
		"toolCall":{"name":"googleSearch","args":{"query":"weather"}},
		"toolResponse":{"name":"googleSearch","response":{"result":"sunny"}},
		"executableCode":{"id":"code-1","language":"PYTHON","code":"print(1)"},
		"codeExecutionResult":{"id":"result-1","outcome":"OUTCOME_OK","output":"1"}
	}`), &part))
	partJSON, err := kitutil.Marshal(part)
	require.NoError(t, err)
	var partWire map[string]any
	require.NoError(t, kitutil.Unmarshal(partJSON, &partWire))
	assert.Equal(t, map[string]any{"name": "googleSearch", "args": map[string]any{"query": "weather"}}, partWire["toolCall"])
	assert.Equal(t, map[string]any{"name": "googleSearch", "response": map[string]any{"result": "sunny"}}, partWire["toolResponse"])
	executableCode, ok := partWire["executableCode"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "code-1", executableCode["id"])
	codeExecutionResult, ok := partWire["codeExecutionResult"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "result-1", codeExecutionResult["id"])

	var geminiResponse dto.GeminiChatResponse
	require.NoError(t, kitutil.Unmarshal([]byte(`{
		"candidates":[{"index":0,"content":{"role":"model","parts":[
			{"toolCall":{"name":"googleSearch","args":{"query":"weather"}}},
			{"toolResponse":{"name":"googleSearch","response":{"result":"sunny"}}},
			{"functionCall":{"id":" call_1 ","name":"lookup","args":{"city":"Taipei"}},"thoughtSignature":"native-signature"},
			{"text":"done"}
		]}}]
	}`), &geminiResponse))

	nonStream := ResponseGeminiChat2OpenAI("chatcmpl-1", 1, &geminiResponse)
	require.Len(t, nonStream.Choices, 1)
	assert.Equal(t, "done", nonStream.Choices[0].Message.StringContent())
	nonStreamCalls := nonStream.Choices[0].Message.ParseToolCalls()
	require.Len(t, nonStreamCalls, 1)
	assert.Equal(t, " call_1 ", nonStreamCalls[0].ID)
	assertToolCallThoughtSignature(t, nonStreamCalls[0], "native-signature")

	stream, _ := StreamResponseGeminiChat2OpenAI(&geminiResponse)
	require.Len(t, stream.Choices, 1)
	assert.Equal(t, "done", stream.Choices[0].Delta.GetContentString())
	require.Len(t, stream.Choices[0].Delta.ToolCalls, 1)
	assert.Equal(t, " call_1 ", stream.Choices[0].Delta.ToolCalls[0].ID)
	assertToolCallThoughtSignature(t, stream.Choices[0].Delta.ToolCalls[0], "native-signature")
}

func TestOpenAIChatToGeminiPrefersClientThoughtSignatureOverBypass(t *testing.T) {
	t.Parallel()

	var request dto.GeneralOpenAIRequest
	require.NoError(t, kitutil.Unmarshal([]byte(`{
		"model":"gemini-3-pro",
		"messages":[
			{"role":"user","content":"look it up"},
			{"role":"assistant","content":"","tool_calls":[
				{"id":"call_1","type":"function","function":{"name":"first","arguments":"{}"}},
				{"id":"call_2","type":"function","function":{"name":"second","arguments":"{}"},"extra_content":{"google":{"thought_signature":"client-signature"}}}
			]}
		]
	}`), &request))
	info := &convmeta.Values{Options: &convmeta.Options{Gemini: convmeta.GeminiOptions{FunctionCallThoughtSignatureEnabled: true}}}

	converted, err := OpenAIChatRequestToGeminiGenerateContent(context.Background(), request, info)
	require.NoError(t, err)
	require.Len(t, converted.Contents, 2)
	require.Len(t, converted.Contents[1].Parts, 2)
	assert.Empty(t, converted.Contents[1].Parts[0].ThoughtSignature)
	var thoughtSignature string
	require.NoError(t, kitutil.Unmarshal(converted.Contents[1].Parts[1].ThoughtSignature, &thoughtSignature))
	assert.Equal(t, "client-signature", thoughtSignature)
	assert.Equal(t, "call_1", converted.Contents[1].Parts[0].FunctionCall.ID)
	assert.Equal(t, "call_2", converted.Contents[1].Parts[1].FunctionCall.ID)
}

func TestGeminiPartialFunctionCallPreservesEarlySignatureAndID(t *testing.T) {
	t.Parallel()

	state, err := NewResponseStreamState(types.RelayFormatGemini, types.RelayFormatOpenAI, ResponseStreamOptions{
		ID:      "chatcmpl-1",
		Created: 1,
	})
	require.NoError(t, err)
	willContinue := true
	firstResults, err := ConvertStreamResponseChunk(nil, nil, state, &dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{
			Content: dto.GeminiChatContent{Parts: []dto.GeminiPart{{
				FunctionCall: &dto.FunctionCall{
					ID:           " call_1 ",
					FunctionName: "lookup",
					WillContinue: &willContinue,
				},
				ThoughtSignature: []byte(`"early-signature"`),
			}}},
		}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, firstResults)

	secondResults, err := ConvertStreamResponseChunk(nil, nil, state, &dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{
			Content: dto.GeminiChatContent{Parts: []dto.GeminiPart{{
				FunctionCall: &dto.FunctionCall{
					ID:           " call_1 ",
					FunctionName: "lookup",
					Arguments:    map[string]any{"city": "Taipei"},
				},
			}}},
		}},
	})
	require.NoError(t, err)
	require.Len(t, secondResults, 1)
	stream, ok := secondResults[0].Value.(*dto.ChatCompletionsStreamResponse)
	require.True(t, ok)
	require.Len(t, stream.Choices, 1)
	require.Len(t, stream.Choices[0].Delta.ToolCalls, 1)
	assert.Equal(t, " call_1 ", stream.Choices[0].Delta.ToolCalls[0].ID)
	assertToolCallThoughtSignature(t, stream.Choices[0].Delta.ToolCalls[0], "early-signature")
}

func TestClearToolCallsClearsThoughtSignatureMetadata(t *testing.T) {
	t.Parallel()

	var response dto.ChatCompletionsStreamResponse
	require.NoError(t, kitutil.Unmarshal([]byte(`{
		"choices":[{"delta":{"tool_calls":[{
			"index":0,
			"id":"call_1",
			"type":"function",
			"function":{"name":"lookup","arguments":"{}"},
			"extra_content":{"google":{"thought_signature":"chunk-signature"}}
		}]}}]
	}`), &response))
	require.Len(t, response.Choices, 1)
	require.Len(t, response.Choices[0].Delta.ToolCalls, 1)
	before, err := kitutil.Marshal(response.Choices[0].Delta.ToolCalls[0])
	require.NoError(t, err)
	assert.Contains(t, string(before), `"thought_signature":"chunk-signature"`)

	response.ClearToolCalls()
	after, err := kitutil.Marshal(response.Choices[0].Delta.ToolCalls[0])
	require.NoError(t, err)
	assert.NotContains(t, string(after), "extra_content")
}

func assertToolCallThoughtSignature(t *testing.T, call any, want string) {
	t.Helper()

	data, err := kitutil.Marshal(call)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, kitutil.Unmarshal(data, &wire))
	extraContent, ok := wire["extra_content"].(map[string]any)
	require.True(t, ok)
	google, ok := extraContent["google"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, want, google["thought_signature"])
}

func TestConvertRequestDefaultPolicyAllowsGeminiCodeExecution(t *testing.T) {
	t.Parallel()

	tools, err := kitutil.Marshal([]map[string]any{{"codeExecution": map[string]any{}}})
	require.NoError(t, err)
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Role: "user", Parts: []dto.GeminiPart{{Text: "run this"}}},
		},
		Tools: tools,
	}

	result, err := ConvertRequest(nil, nil, types.RelayFormatOpenAI, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.IsType(t, &dto.GeneralOpenAIRequest{}, result.Value)
	assert.True(t, hasConversionDiagnosticCode(result.Diagnostics, "unsupported_hosted_tool"))
}

func TestConvertResponseStrictPolicyStillSucceedsOnContinuationLoss(t *testing.T) {
	t.Parallel()

	text := "hello"
	resp := &dto.ClaudeResponse{
		Id:         "msg_1",
		Type:       "message",
		Role:       "assistant",
		Model:      "claude-test",
		StopReason: "pause_turn",
		Content: []dto.ClaudeMediaMessage{
			{Type: "redacted_thinking", Data: "secret"},
			{Type: "text", Text: &text},
		},
	}
	info := &convmeta.Values{
		Options: &convmeta.Options{ToolLossPolicy: types.ConversionLossPolicyStrict},
	}

	result, err := ConvertResponse(nil, info, types.RelayFormatOpenAI, resp)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.IsType(t, &dto.OpenAITextResponse{}, result.Value)
	assert.True(t, hasConversionDiagnosticCode(result.Diagnostics, "continuation_state_lost"))
}

func TestConvertRequestSafePolicyReturnsConversionLossError(t *testing.T) {
	t.Parallel()

	tools, err := kitutil.Marshal([]map[string]any{{"codeExecution": map[string]any{}}})
	require.NoError(t, err)
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Role: "user", Parts: []dto.GeminiPart{{Text: "run this"}}},
		},
		Tools: tools,
	}
	info := &convmeta.Values{
		Options: &convmeta.Options{ToolLossPolicy: types.ConversionLossPolicySafe},
	}

	result, err := ConvertRequest(nil, info, types.RelayFormatOpenAI, req)
	require.Error(t, err)
	var loss *types.ConversionLossError
	require.ErrorAs(t, err, &loss)
	require.NotEmpty(t, loss.Diagnostics)
	require.NotNil(t, result)
	assert.True(t, hasConversionDiagnosticCode(loss.Diagnostics, "unsupported_hosted_tool"))
	assert.True(t, hasConversionDiagnosticCode(result.Diagnostics, "unsupported_hosted_tool"))
}

func hasConversionDiagnosticCode(diagnostics []types.ConversionDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
