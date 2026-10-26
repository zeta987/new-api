package moonshot

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	channelconstant "github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKimiK31ProRequestContract(t *testing.T) {
	for _, model := range []string{"kimi-k3.1-pro", "kimi-k3.1-pro-highspeed"} {
		for _, effort := range []string{"", "low", "high", "max"} {
			t.Run(model+"/"+effort, func(t *testing.T) {
				name := model
				if effort != "" {
					name += "-" + effort
				}
				info := newKimiRelayInfo("", false)
				info.OriginModelName, info.UpstreamModelName = name, name
				request := &dto.GeneralOpenAIRequest{Model: name, Temperature: common.GetPointer(0.2), TopP: common.GetPointer(0.4), N: common.GetPointer(3),
					FrequencyPenalty: common.GetPointer(1.0), PresencePenalty: common.GetPointer(1.0), THINKING: []byte(`{"type":"disabled"}`)}
				converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
				require.NoError(t, err)
				wire, err := common.Marshal(converted)
				require.NoError(t, err)
				want := map[string]any{"model": model}
				if effort != "" {
					want["reasoning_effort"] = effort
				}
				expected, err := common.Marshal(want)
				require.NoError(t, err)
				assert.JSONEq(t, string(expected), string(wire))
				assert.Equal(t, model, info.UpstreamModelName)
			})
		}
		for _, effort := range []string{"none", "medium", "xhigh"} {
			t.Run(model+"/reject/"+effort, func(t *testing.T) {
				_, err := (&Adaptor{}).ConvertOpenAIRequest(nil, nil, &dto.GeneralOpenAIRequest{Model: model, ReasoningEffort: effort})
				require.Error(t, err)
			})
		}
		_, err := (&Adaptor{}).ConvertOpenAIRequest(nil, nil, &dto.GeneralOpenAIRequest{Model: model + "-none"})
		require.Error(t, err)
		assert.Contains(t, (&Adaptor{}).GetModelList(), model+"-max")
		assert.NotContains(t, (&Adaptor{}).GetModelList(), model+"-none")
	}
}

func TestKimiK31ResponsesEffortAlias(t *testing.T) {
	info := newKimiRelayInfo("", false)
	info.OriginModelName, info.UpstreamModelName = "kimi-k3.1-pro-highspeed-high", "kimi-k3.1-pro-highspeed-high"
	request := dto.OpenAIResponsesRequest{Model: info.UpstreamModelName, Temperature: common.GetPointer(0.2), Reasoning: &dto.Reasoning{Effort: "low"}}
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	wire, err := common.Marshal(converted)
	require.NoError(t, err)
	assert.Contains(t, string(wire), `"model":"kimi-k3.1-pro-highspeed"`)
	assert.Contains(t, string(wire), `"effort":"high"`)
	assert.NotContains(t, string(wire), `"temperature"`)
}

func TestKimiCacheUsageChatAndSSE(t *testing.T) {
	originalTimeout := channelconstant.StreamingTimeout
	channelconstant.StreamingTimeout = 30
	t.Cleanup(func() { channelconstant.StreamingTimeout = originalTimeout })
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			c, _ := newKimiTestContext(t, nil)
			info := newKimiRelayInfo("", stream)
			info.UpstreamModelName = "kimi-k3.1-pro"
			body := `{"id":"chatcmpl-test","object":"chat.completion","model":"kimi-k3.1-pro","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1000,"completion_tokens":100,"total_tokens":1100,"prompt_tokens_details":{"cached_tokens":200,"cache_write_tokens":100}}}`
			contentType := "application/json"
			if stream {
				body = `data: {"id":"chatcmpl-test","model":"kimi-k3.1-pro","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}` + "\n\ndata: " + `{"id":"chatcmpl-test","choices":[],"usage":{"prompt_tokens":1000,"completion_tokens":100,"total_tokens":1100,"prompt_tokens_details":{"cached_tokens":200,"cache_write_tokens":100}}}` + "\n\ndata: [DONE]\n\n"
				contentType = "text/event-stream"
			}
			response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
			response.Header.Set("Content-Type", contentType)
			response.Header.Set("Msh-Usage-Cache-Write-Tokens-5m", "50")
			response.Header.Set("Msh-Usage-Cache-Write-Tokens-1h", "50")
			result, apiErr := (&Adaptor{}).DoResponse(c, response, info)
			require.Nil(t, apiErr)
			usage, ok := result.(*dto.Usage)
			require.True(t, ok)
			assert.Equal(t, dto.BillingUsageSemanticOpenAI, usage.UsageSemantic)
			assert.Equal(t, 1000, usage.PromptTokens)
			assert.Equal(t, 50, usage.ClaudeCacheCreation5mTokens)
			assert.Equal(t, 50, usage.ClaudeCacheCreation1hTokens)
		})
	}
}

func TestKimiResponsesCacheUsageSnapshot(t *testing.T) {
	originalTimeout := channelconstant.StreamingTimeout
	channelconstant.StreamingTimeout = 30
	t.Cleanup(func() { channelconstant.StreamingTimeout = originalTimeout })
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			c, _ := newKimiTestContext(t, nil)
			info := newKimiRelayInfo("", stream)
			info.RelayMode, info.RelayFormat = relayconstant.RelayModeResponses, types.RelayFormatOpenAIResponses
			info.UpstreamModelName = "kimi-k3.1-pro"
			body := `{"id":"resp-test","object":"response","model":"kimi-k3.1-pro","status":"completed","output":[],"usage":{"input_tokens":1000,"output_tokens":100,"total_tokens":1100,"input_tokens_details":{"cached_tokens":200,"cache_write_tokens":100,"cache_write_tokens_details":{"ttl_12h_tokens":100}}}}`
			contentType := "application/json"
			if stream {
				body = "data: " + `{"type":"response.completed","response":` + body + "}\n\n"
				contentType = "text/event-stream"
			}
			response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
			response.Header.Set("Content-Type", contentType)
			response.Header.Set("Msh-Usage-Cache-Write-Tokens-5m", "50")
			response.Header.Set("Msh-Usage-Cache-Write-Tokens-1h", "50")
			result, apiErr := (&Adaptor{}).DoResponse(c, response, info)
			require.Nil(t, apiErr)
			usage, ok := result.(*dto.Usage)
			require.True(t, ok)
			snapshot := dto.NewOpenAIResponsesBillingUsage(usage)
			require.NotNil(t, snapshot)
			canonical, ok := snapshot.CanonicalUsage()
			require.True(t, ok)
			assert.Equal(t, 1000, canonical.PromptTokens)
			assert.Equal(t, 50, canonical.ClaudeCacheCreation5mTokens)
			assert.Equal(t, 50, canonical.ClaudeCacheCreation1hTokens)
			assert.Zero(t, canonical.PromptTokensDetails.CacheWrite12hTokens())
			params := service.BuildTieredTokenParams(canonical, false, map[string]bool{"cr": true, "cc": true, "cc1h": true})
			assert.Equal(t, float64(700), params.P)
			assert.Equal(t, float64(50), params.CC)
			assert.Equal(t, float64(50), params.CC1h)
			assert.Zero(t, params.CC12h)
		})
	}
}

func TestKimiCacheUsageUpdatesExistingBillingSnapshot(t *testing.T) {
	details := dto.InputTokenDetails{CachedTokens: 200, CacheWriteTokens: 100,
		CacheWriteTokensDetails: &dto.CacheWriteTokenDetails{TTL12hTokens: 100}}
	usage := &dto.Usage{PromptTokens: 1000, CompletionTokens: 100, InputTokens: 1000, OutputTokens: 100,
		PromptTokensDetails: details, InputTokensDetails: &details}
	usage.BillingUsage = dto.NewOpenAIResponsesBillingUsage(usage)
	headers := make(http.Header)
	headers.Set("Msh-Usage-Cache-Write-Tokens-5m", "50")
	headers.Set("Msh-Usage-Cache-Write-Tokens-1h", "50")
	applyKimiCacheUsage(usage, headers)
	canonical, ok := usage.BillingUsage.CanonicalUsage()
	require.True(t, ok)
	params := service.BuildTieredTokenParams(canonical, false, map[string]bool{"cr": true, "cc": true, "cc1h": true})
	assert.Equal(t, float64(700), params.P)
	assert.Equal(t, float64(50), params.CC)
	assert.Equal(t, float64(50), params.CC1h)
	assert.Zero(t, params.CC12h)
}

func TestKimiCacheUsageRejectsContradictoryHeaders(t *testing.T) {
	for _, pair := range [][2]string{{"50", "51"}, {"-1", "101"}, {"50", ""}, {"9223372036854775807", "1"}, {"0", "0"}} {
		usage := &dto.Usage{PromptTokens: 1000, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200, CacheWriteTokens: 100}}
		usage.BillingUsage = dto.NewOpenAIChatBillingUsage(usage)
		headers := make(http.Header)
		headers.Set("Msh-Usage-Cache-Write-Tokens-5m", pair[0])
		headers.Set("Msh-Usage-Cache-Write-Tokens-1h", pair[1])
		applyKimiCacheUsage(usage, headers)
		assert.Equal(t, 100, usage.PromptTokensDetails.CacheWriteTokens)
		assert.Zero(t, usage.ClaudeCacheCreation1hTokens)
		assert.Zero(t, usage.BillingUsage.OpenAIUsage.ClaudeCacheCreation1hTokens)
	}
}

func TestConvertOpenAIRequestKimiK26NormalizesFixedSamplingParameters(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:            "kimi-k2.6",
		Temperature:      common.GetPointer[float64](0.7),
		TopP:             common.GetPointer[float64](1),
		TopK:             common.GetPointer(40),
		N:                common.GetPointer(2),
		FrequencyPenalty: common.GetPointer[float64](0.5),
		PresencePenalty:  common.GetPointer[float64](0.5),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Nil(t, convertedRequest.Temperature)
	require.Nil(t, convertedRequest.TopP)
	require.Nil(t, convertedRequest.TopK)
	require.Nil(t, convertedRequest.N)
	require.NotNil(t, convertedRequest.FrequencyPenalty)
	require.Zero(t, *convertedRequest.FrequencyPenalty)
	require.NotNil(t, convertedRequest.PresencePenalty)
	require.Zero(t, *convertedRequest.PresencePenalty)
}

func TestConvertOpenAIRequestKimiK26DisabledOmitsFixedTemperature(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:       "kimi-k2.6",
		Temperature: common.GetPointer[float64](0.7),
		THINKING:    []byte(`{"type":"disabled"}`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Nil(t, convertedRequest.Temperature)
}

func TestConvertOpenAIRequestKimiK26DefaultsToDisabledThinking(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model: "kimi-k2.6",
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Nil(t, convertedRequest.Temperature)
	require.JSONEq(t, `{"type":"disabled"}`, string(convertedRequest.THINKING))
}

func TestConvertOpenAIRequestKimiK26PreservesExplicitThinking(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "kimi-k2.6",
		THINKING: []byte(`{"type":"enabled"}`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.6",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.JSONEq(t, `{"type":"enabled"}`, string(convertedRequest.THINKING))
}

func TestConvertOpenAIRequestOtherMoonshotModelKeepsTemperature(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:       "kimi-k2.5",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.5",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	convertedRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, convertedRequest.Temperature)
	require.Equal(t, 0.7, *convertedRequest.Temperature)
}

// Kimi serves OpenAI Responses natively (kimi-k3); the body is forwarded
// unchanged and other models are rejected by the upstream.
func TestResponsesRequestPassesThroughToV1Responses(t *testing.T) {
	const body = `{"model":"kimi-k3","input":"Latest Go release?","tools":[{"type":"web_search"}],"stream":false}`
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.moonshot.invalid", UpstreamModelName: "kimi-k3"},
	}
	url, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.moonshot.invalid/v1/responses", url)

	var request dto.OpenAIResponsesRequest
	require.NoError(t, common.UnmarshalJsonStr(body, &request))
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	wire, err := common.Marshal(converted)
	require.NoError(t, err)
	assert.JSONEq(t, body, string(wire))
}
