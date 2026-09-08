package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	openaichannel "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveChatRequestHandlingRespectsProtocolPolicy(t *testing.T) {
	tests := []struct {
		name               string
		model              string
		globalPass         bool
		channelPass        bool
		responsesBridge    bool
		channelType        int
		upstreamModel      string
		reasoningState     *dto.ReasoningConversionState
		wantResponses      bool
		wantRawPassThrough bool
	}{
		{name: "glm alias bypasses responses bridge", model: "glm-5.3-flash-high", channelType: constant.ChannelTypeOpenAI, responsesBridge: true},
		{name: "glm alias bypasses global pass through", model: "glm-5.3-flash-high", channelType: constant.ChannelTypeZhipu_v4, globalPass: true},
		{name: "glm alias bypasses channel pass through", model: "glm-5.3-flash-high", channelType: constant.ChannelTypeOpenAI, channelPass: true},
		{name: "openrouter glm alias uses responses bridge", model: "glm-5.3-flash-high", channelType: constant.ChannelTypeOpenRouter, responsesBridge: true, wantResponses: true},
		{name: "openrouter glm alias keeps chat without bridge", model: "glm-5.3-low", channelType: constant.ChannelTypeOpenRouter},
		{name: "openrouter glm alias respects global pass through", model: "glm-5.3-max", channelType: constant.ChannelTypeOpenRouter, responsesBridge: true, globalPass: true, wantRawPassThrough: true},
		{name: "openrouter glm alias respects channel pass through", model: "glm-5.3-flash-low", channelType: constant.ChannelTypeOpenRouter, responsesBridge: true, channelPass: true, wantRawPassThrough: true},
		{name: "ordinary model uses responses bridge", model: "gpt-4.1", responsesBridge: true, wantResponses: true},
		{name: "ordinary model uses global pass through", model: "gpt-4.1", globalPass: true, wantRawPassThrough: true},
		{name: "bare glm keeps configured pass through", model: "glm-5.3-flash", channelPass: true, wantRawPassThrough: true},
		{name: "qwen max alias uses responses bridge", model: "qwen3.8-max-low", channelType: constant.ChannelTypeOpenAI, responsesBridge: true, wantResponses: true},
		{name: "qwen flash alias uses responses bridge", model: "qwen3.8-flash-low", channelType: constant.ChannelTypeOpenAI, responsesBridge: true, wantResponses: true},
		{name: "qwen alias stays on chat without responses bridge", model: "qwen3.8-max-low", channelType: constant.ChannelTypeOpenAI},
		{name: "qwen alias with modifier uses responses bridge", model: "qwen3.8-max-low@effort:xhigh", channelType: constant.ChannelTypeOpenAI, responsesBridge: true, wantResponses: true},
		{name: "mapped qwen alias intent uses responses bridge", model: "customer-model", upstreamModel: "qwen3.8-max", reasoningState: &dto.ReasoningConversionState{Effort: "low"}, channelType: constant.ChannelTypeOpenAI, responsesBridge: true, wantResponses: true},
		{name: "qwen alias keeps global pass through", model: "qwen3.8-max-low", channelType: constant.ChannelTypeOpenAI, globalPass: true, responsesBridge: true, wantRawPassThrough: true},
		{name: "qwen alias keeps channel pass through", model: "qwen3.8-flash-xhigh", channelType: constant.ChannelTypeOpenAI, channelPass: true, responsesBridge: true, wantRawPassThrough: true},
		{name: "other channel does not force qwen chat", model: "qwen3.8-max-low", channelType: constant.ChannelTypeAzure, responsesBridge: true, wantResponses: true},
		{name: "bare qwen uses responses bridge", model: "qwen3.8-max", responsesBridge: true, wantResponses: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			upstreamModel := testCase.upstreamModel
			if upstreamModel == "" {
				upstreamModel = testCase.model
			}
			info := &relaycommon.RelayInfo{
				OriginModelName:     testCase.model,
				RelayMode:           relayconstant.RelayModeChatCompletions,
				ReasoningConversion: testCase.reasoningState,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       testCase.channelType,
					UpstreamModelName: upstreamModel,
					ChannelSetting:    dto.ChannelSettings{PassThroughBodyEnabled: testCase.channelPass},
				},
			}

			useResponses, useRawPassThrough := resolveChatRequestHandling(info, testCase.globalPass, testCase.responsesBridge)

			assert.Equal(t, testCase.wantResponses, useResponses)
			assert.Equal(t, testCase.wantRawPassThrough, useRawPassThrough)
		})
	}
}

func TestQwenAliasesRelayResponsesEffortAndOverrideTools(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled: true, AllChannels: true, ModelPatterns: []string{`^qwen3.8-.*$`},
	}
	for _, base := range []string{"qwen3.8-max", "qwen3.8-flash"} {
		for _, effort := range []string{"none", "low", "medium", "xhigh"} {
			t.Run(base+"-"+effort, func(t *testing.T) {
				captured, server := captureResponsesUpstream(t)

				alias := base + "-" + effort
				request := &dto.GeneralOpenAIRequest{
					Model: alias, Messages: []dto.Message{{Role: "user", Content: "請搜尋"}},
				}
				info := &relaycommon.RelayInfo{
					Request: request, OriginModelName: alias,
					RelayMode: relayconstant.RelayModeChatCompletions, RelayFormat: types.RelayFormatOpenAI,
					RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI},
					ChannelMeta: &relaycommon.ChannelMeta{
						ChannelType: constant.ChannelTypeOpenAI, UpstreamModelName: alias,
						ChannelBaseUrl: server.URL + "/compatible-mode", ApiKey: "test-key",
					},
				}
				require.NoError(t, common.UnmarshalJsonStr(`{"operations":[{"mode":"set","path":"tools","value":[{"type":"web_search"},{"type":"web_extractor"},{"type":"code_interpreter"}],"conditions":[{"path":"messages.-1.content","mode":"contains","value":"搜尋"}],"logic":"OR"}]}`, &info.ParamOverride))
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
				c.Request.Header.Set("Content-Type", "application/json")
				require.NoError(t, helper.ApplyReasoningModelSuffix(c, info, request))
				bridgeEnabled := service.ShouldChatCompletionsUseResponsesPolicy(policy, 1, info.ChannelType, info.OriginModelName)
				useResponses, useRaw := resolveChatRequestHandling(info, false, bridgeEnabled)
				require.True(t, useResponses)
				require.False(t, useRaw)
				adaptor := &openaichannel.Adaptor{}
				adaptor.Init(info)
				usage, apiErr := textRequestViaResponses(c, info, adaptor, request)
				require.Nil(t, apiErr)
				require.NotNil(t, usage)
				assert.Equal(t, 5, usage.TotalTokens)
				select {
				case upstream := <-captured:
					assert.Equal(t, "/compatible-mode/v1/responses", upstream.path)
					var body map[string]any
					require.NoError(t, common.Unmarshal(upstream.body, &body))
					assert.Equal(t, base, body["model"])
					assert.NotContains(t, body, "messages")
					assert.NotContains(t, body, "reasoning_effort")
					reasoning, ok := body["reasoning"].(map[string]any)
					require.True(t, ok)
					assert.Equal(t, effort, reasoning["effort"])
					assert.Equal(t, []any{map[string]any{"type": "web_search"}, map[string]any{"type": "web_extractor"}, map[string]any{"type": "code_interpreter"}}, body["tools"])
				default:
					t.Fatal("upstream did not receive a Responses request")
				}
			})
		}
	}
}

func TestOpenRouterAliasesRelayMappedResponsesTools(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled: true, ChannelTypes: []int{constant.ChannelTypeOpenRouter},
		ModelPatterns: []string{`^(qwen|glm)-?.*$`},
	}
	for _, testCase := range []struct {
		alias    string
		upstream string
		effort   string
	}{
		{"qwen3.8-max-low", "qwen/qwen3.8-max-0902", "low"},
		{"qwen3.8-flash-none", "qwen/qwen3.8-flash", "none"},
		{"glm-5.3-flash-max", "z-ai/glm-5.3-flash", "max"},
	} {
		t.Run(testCase.alias, func(t *testing.T) {
			captured, server := captureResponsesUpstream(t)
			request := &dto.GeneralOpenAIRequest{
				Model: testCase.alias, Messages: []dto.Message{{Role: "user", Content: "請計算"}},
			}
			info := &relaycommon.RelayInfo{
				Request: request, OriginModelName: testCase.alias,
				RelayMode: relayconstant.RelayModeChatCompletions, RelayFormat: types.RelayFormatOpenAI,
				RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI},
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType: constant.ChannelTypeOpenRouter, UpstreamModelName: testCase.alias,
					ChannelBaseUrl: server.URL, ApiKey: "test-key",
					ParamOverride: map[string]interface{}{
						"operations": []interface{}{
							map[string]interface{}{"mode": "set", "path": "reasoning", "value": map[string]interface{}{"effort": testCase.effort}},
							map[string]interface{}{"mode": "set", "path": "tools", "value": []interface{}{
								map[string]interface{}{"type": "openrouter:shell", "custom": map[string]interface{}{"type": "openrouter:shell", "parameters": map[string]interface{}{"engine": "openrouter"}}},
							}},
						},
					},
				},
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			c.Request.Header.Set("Content-Type", "application/json")
			mapping, err := common.Marshal(map[string]string{testCase.alias: testCase.upstream})
			require.NoError(t, err)
			c.Set("model_mapping", string(mapping))
			require.NoError(t, helper.ModelMappedHelper(c, info, request))
			require.NoError(t, helper.ApplyReasoningModelSuffix(c, info, request))
			bridge := service.ShouldChatCompletionsUseResponsesPolicy(policy, 1, info.ChannelType, info.OriginModelName)
			useResponses, useRaw := resolveChatRequestHandling(info, false, bridge)
			require.True(t, useResponses)
			require.False(t, useRaw)
			adaptor := &openaichannel.Adaptor{}
			adaptor.Init(info)
			usage, apiErr := textRequestViaResponses(c, info, adaptor, request)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			select {
			case upstream := <-captured:
				assert.Equal(t, "/v1/responses", upstream.path)
				var body map[string]interface{}
				require.NoError(t, common.Unmarshal(upstream.body, &body))
				assert.Equal(t, testCase.upstream, body["model"])
				assert.NotContains(t, body, "messages")
				reasoning, ok := body["reasoning"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, testCase.effort, reasoning["effort"])
				assert.Equal(t, []interface{}{map[string]interface{}{"type": "openrouter:shell", "parameters": map[string]interface{}{"engine": "openrouter"}}}, body["tools"])
			default:
				t.Fatal("upstream did not receive a Responses request")
			}
		})
	}
}

type capturedResponsesRequest struct {
	path string
	body []byte
}

func captureResponsesUpstream(t *testing.T) (<-chan capturedResponsesRequest, *httptest.Server) {
	t.Helper()
	captured := make(chan capturedResponsesRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		captured <- capturedResponsesRequest{path: r.URL.Path, body: body}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_audit","object":"response","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`))
	}))
	t.Cleanup(server.Close)
	return captured, server
}
