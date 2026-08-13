package deepseek

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDeepSeekModelListIncludesV4LowSuffixes(t *testing.T) {
	models := (&Adaptor{}).GetModelList()

	assert.Contains(t, models, "deepseek-v4-flash-low")
	assert.Contains(t, models, "deepseek-v4-pro-low")
}

func TestApplyDeepSeekV4LowSuffixAcrossRelayFormats(t *testing.T) {
	t.Run("OpenAI chat completions", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-pro-low"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low")

		err := applyDeepSeekV4OpenAIThinkingSuffix(info, request)

		require.NoError(t, err)
		assert.Equal(t, "deepseek-v4-pro", request.Model)
		assert.Equal(t, "enabled", gjson.GetBytes(request.THINKING, "type").String())
		assert.Equal(t, "low", request.ReasoningEffort)
		assert.Equal(t, "deepseek-v4-pro", info.UpstreamModelName)
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("Anthropic messages", func(t *testing.T) {
		request := &dto.ClaudeRequest{Model: "deepseek-v4-pro-low"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low")

		err := applyDeepSeekV4ClaudeThinkingSuffix(info, request)

		require.NoError(t, err)
		assert.Equal(t, "deepseek-v4-pro", request.Model)
		require.NotNil(t, request.Thinking)
		assert.Equal(t, "enabled", request.Thinking.Type)
		assert.Equal(t, "low", gjson.GetBytes(request.OutputConfig, "effort").String())
		assert.Equal(t, "deepseek-v4-pro", info.UpstreamModelName)
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("Responses", func(t *testing.T) {
		request := &dto.OpenAIResponsesRequest{Model: "deepseek-v4-pro-low"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low")

		applyDeepSeekV4ResponsesThinkingSuffix(info, request)

		assert.Equal(t, "deepseek-v4-pro", request.Model)
		require.NotNil(t, request.Reasoning)
		assert.Equal(t, "low", request.Reasoning.Effort)
		assert.Equal(t, "deepseek-v4-pro", info.UpstreamModelName)
		assert.Equal(t, "low", info.ReasoningEffort)
	})
}

func deepSeekLowTestRelayInfo(model string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: model,
		},
	}
}
