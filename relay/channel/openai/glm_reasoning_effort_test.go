package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatGLMAliasWritesTopLevelReasoningEffort(t *testing.T) {
	alias := "glm-5.3-flash-low"
	request := &dto.GeneralOpenAIRequest{Model: alias, ReasoningEffort: "high"}
	info := openAIGLMRelayInfo(alias, alias)

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)

	assert.Equal(t, "glm-5.3-flash", got.Model)
	assert.Equal(t, "low", got.ReasoningEffort)
	assert.Equal(t, "glm-5.3-flash", info.UpstreamModelName)
	assert.Equal(t, "low", info.ReasoningEffort)
}

func TestOpenAIChatGLMAliasKeepsMappedProviderModel(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "provider/glm-flash"}
	info := openAIGLMRelayInfo("glm-5.3-flash-max", "provider/glm-flash")
	info.IsModelMapped = true

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)

	assert.Equal(t, "provider/glm-flash", got.Model)
	assert.Equal(t, "max", got.ReasoningEffort)
	assert.Equal(t, "provider/glm-flash", info.UpstreamModelName)
	assert.Equal(t, "max", info.ReasoningEffort)
}

func TestOpenAIChatBareGLMPreservesExplicitReasoningEffort(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.3-flash", ReasoningEffort: "high"}
	info := openAIGLMRelayInfo("glm-5.3-flash", "glm-5.3-flash")

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)

	assert.Equal(t, "glm-5.3-flash", got.Model)
	assert.Equal(t, "high", got.ReasoningEffort)
	assert.Equal(t, "high", info.ReasoningEffort)
}

func TestOpenAIChatGLMConversionLeavesOtherModelsAndAzureUnchanged(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		model       string
	}{
		{name: "non GLM OpenAI model", channelType: constant.ChannelTypeOpenAI, model: "custom-model"},
		{name: "Azure GLM alias", channelType: constant.ChannelTypeAzure, model: "glm-5.3-flash-high"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := &dto.GeneralOpenAIRequest{Model: testCase.model, ReasoningEffort: "medium"}
			info := openAIGLMRelayInfo(testCase.model, testCase.model)
			info.ChannelType = testCase.channelType

			converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
			require.NoError(t, err)
			got, ok := converted.(*dto.GeneralOpenAIRequest)
			require.True(t, ok)

			assert.Equal(t, testCase.model, got.Model)
			assert.Equal(t, "medium", got.ReasoningEffort)
			assert.Empty(t, info.ReasoningEffort)
		})
	}
}

func TestOpenAIResponsesDoesNotConvertGLMAlias(t *testing.T) {
	alias := "glm-5.3-flash-high"
	request := dto.OpenAIResponsesRequest{Model: alias}
	info := openAIGLMRelayInfo(alias, alias)
	info.RelayMode = relayconstant.RelayModeResponses
	info.RelayFormat = types.RelayFormatOpenAIResponses

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, alias, got.Model)
	assert.Nil(t, got.Reasoning)
	assert.Empty(t, info.ReasoningEffort)
}

func TestOpenAIChatGLMUsesConfiguredV1ChatCompletionsPath(t *testing.T) {
	info := openAIGLMRelayInfo("glm-5.3-flash-high", "glm-5.3-flash")
	info.ChannelBaseUrl = "https://relay.example"
	info.RequestURLPath = "/v1/chat/completions"

	requestURL, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	assert.Equal(t, "https://relay.example/v1/chat/completions", requestURL)
}

func openAIGLMRelayInfo(originModel, upstreamModel string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: originModel,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		RelayFormat:     types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: upstreamModel,
		},
	}
}
