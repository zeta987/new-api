package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
)

func TestResolveChatRequestHandlingForcesGLMAliasesThroughChatAdaptor(t *testing.T) {
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
		{name: "glm alias bypasses responses bridge", model: "glm-5.3-flash-high", responsesBridge: true},
		{name: "glm alias bypasses global pass through", model: "glm-5.3-flash-high", globalPass: true},
		{name: "glm alias bypasses channel pass through", model: "glm-5.3-flash-high", channelPass: true},
		{name: "ordinary model uses responses bridge", model: "gpt-4.1", responsesBridge: true, wantResponses: true},
		{name: "ordinary model uses global pass through", model: "gpt-4.1", globalPass: true, wantRawPassThrough: true},
		{name: "bare glm keeps configured pass through", model: "glm-5.3-flash", channelPass: true, wantRawPassThrough: true},
		{name: "qwen alias bypasses responses bridge", model: "qwen3.8-max-low", channelType: constant.ChannelTypeOpenAI, responsesBridge: true},
		{name: "qwen alias with modifier bypasses responses bridge", model: "qwen3.8-max-low@effort:xhigh", channelType: constant.ChannelTypeOpenAI, responsesBridge: true},
		{name: "mapped qwen alias intent bypasses responses bridge", model: "customer-model", upstreamModel: "qwen3.8-max", reasoningState: &dto.ReasoningConversionState{Effort: "low"}, channelType: constant.ChannelTypeOpenAI, responsesBridge: true},
		{name: "qwen alias keeps global pass through", model: "qwen3.8-max-low", channelType: constant.ChannelTypeOpenAI, globalPass: true, wantRawPassThrough: true},
		{name: "qwen alias keeps channel pass through", model: "qwen3.8-flash-xhigh", channelType: constant.ChannelTypeOpenAI, channelPass: true, wantRawPassThrough: true},
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
