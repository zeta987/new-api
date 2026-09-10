package deepseek

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDeepSeekModelListIncludesCurrentLowSuffixes(t *testing.T) {
	models := (&Adaptor{}).GetModelList()

	assert.Contains(t, models, "deepseek-flash-low")
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

func TestApplyDeepSeekV4HighSuffixAcrossRelayFormats(t *testing.T) {
	for _, base := range []string{"deepseek-flash", "deepseek-v4-pro"} {
		t.Run(base, func(t *testing.T) {
			alias := base + "-high"
			assert.Contains(t, (&Adaptor{}).GetModelList(), alias)

			info := deepSeekLowTestRelayInfo(alias)
			chat := &dto.GeneralOpenAIRequest{Model: alias, ReasoningEffort: "max"}
			require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(info, chat))
			assert.Equal(t, base, chat.Model)
			assert.JSONEq(t, `{"type":"enabled"}`, string(chat.THINKING))
			assert.Equal(t, "high", chat.ReasoningEffort)
			assert.Equal(t, base, info.UpstreamModelName)
			assert.Equal(t, "high", info.ReasoningEffort)

			info = deepSeekLowTestRelayInfo(alias)
			claude := &dto.ClaudeRequest{Model: alias, Thinking: &dto.Thinking{Type: "disabled"}}
			require.NoError(t, applyDeepSeekV4ClaudeThinkingSuffix(info, claude))
			assert.Equal(t, base, claude.Model)
			require.NotNil(t, claude.Thinking)
			assert.Equal(t, "enabled", claude.Thinking.Type)
			assert.JSONEq(t, `{"effort":"high"}`, string(claude.OutputConfig))
			assert.Equal(t, base, info.UpstreamModelName)
			assert.Equal(t, "high", info.ReasoningEffort)

			info = deepSeekLowTestRelayInfo(alias)
			responses := &dto.OpenAIResponsesRequest{Model: alias, Reasoning: &dto.Reasoning{Effort: "none"}}
			applyDeepSeekV4ResponsesThinkingSuffix(info, responses)
			assert.Equal(t, base, responses.Model)
			assert.Equal(t, "high", responses.Reasoning.Effort)
			assert.Equal(t, base, info.UpstreamModelName)
			assert.Equal(t, "high", info.ReasoningEffort)
		})
	}
}

func TestDeepSeekFlashEffortSuffixes(t *testing.T) {
	for _, tc := range []struct {
		model    string
		base     string
		thinking string
		effort   string
	}{
		{"deepseek-flash-none", "deepseek-flash", "disabled", ""},
		{"deepseek-flash-low", "deepseek-flash", "enabled", "low"},
		{"deepseek-flash-max", "deepseek-flash", "enabled", "max"},
		{"deepseek-flash", "deepseek-flash", "enabled", "high"},
		{"deepseek-flash-medium", "deepseek-flash-medium", "enabled", "high"},
	} {
		t.Run(tc.model, func(t *testing.T) {
			chat := &dto.GeneralOpenAIRequest{Model: tc.model, ReasoningEffort: "high", THINKING: []byte(`{"type":"enabled"}`)}
			info := deepSeekLowTestRelayInfo(tc.model)
			require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(info, chat))
			assert.Equal(t, tc.base, chat.Model)
			assert.Equal(t, tc.thinking, gjson.GetBytes(chat.THINKING, "type").String())
			assert.Equal(t, tc.effort, chat.ReasoningEffort)

			claude := &dto.ClaudeRequest{Model: tc.model, Thinking: &dto.Thinking{Type: "enabled"}, OutputConfig: []byte(`{"effort":"high"}`)}
			info = deepSeekLowTestRelayInfo(tc.model)
			require.NoError(t, applyDeepSeekV4ClaudeThinkingSuffix(info, claude))
			assert.Equal(t, tc.base, claude.Model)
			require.NotNil(t, claude.Thinking)
			assert.Equal(t, tc.thinking, claude.Thinking.Type)
			assert.Equal(t, tc.effort, gjson.GetBytes(claude.OutputConfig, "effort").String())

			responses := &dto.OpenAIResponsesRequest{Model: tc.model, Reasoning: &dto.Reasoning{Effort: "high"}}
			info = deepSeekLowTestRelayInfo(tc.model)
			applyDeepSeekV4ResponsesThinkingSuffix(info, responses)
			assert.Equal(t, tc.base, responses.Model)
			want := tc.effort
			if tc.thinking == "disabled" {
				want = "none"
			}
			assert.Equal(t, want, responses.Reasoning.Effort)
		})
	}
	chat := &dto.GeneralOpenAIRequest{Model: "deepseek-flash"}
	require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(deepSeekLowTestRelayInfo(chat.Model), chat))
	assert.Empty(t, chat.THINKING)
	assert.Empty(t, chat.ReasoningEffort)
}
