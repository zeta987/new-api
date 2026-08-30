package zhipu_4v_test

import (
	"testing"

	"github.com/QuantumNous/new-api/relay/channel/zhipu_4v"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGLM53FlashChatAliasesWriteTopLevelReasoningEffort(t *testing.T) {
	for _, effort := range []string{"low", "high", "max"} {
		t.Run(effort, func(t *testing.T) {
			alias := "glm-5.3-flash-" + effort
			request := &dto.GeneralOpenAIRequest{Model: alias, ReasoningEffort: "medium"}
			info := &relaycommon.RelayInfo{
				OriginModelName: alias,
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: alias},
			}

			converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
			require.NoError(t, err)
			got := requireZhipuRequest(t, converted)

			assert.Equal(t, "glm-5.3-flash", got.Model)
			assert.Equal(t, effort, got.ReasoningEffort)
			assert.Equal(t, "glm-5.3-flash", info.UpstreamModelName)
			assert.Equal(t, effort, info.ReasoningEffort)
		})
	}
}

func TestGLM53FlashResponsesAliasWritesReasoningEffort(t *testing.T) {
	alias := "glm-5.3-flash-high"
	request := dto.OpenAIResponsesRequest{
		Model:     alias,
		Reasoning: &dto.Reasoning{Effort: "low"},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: alias,
		RelayMode:       relayconstant.RelayModeResponses,
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: alias},
	}

	converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "glm-5.3-flash", got.Model)
	require.NotNil(t, got.Reasoning)
	assert.Equal(t, "high", got.Reasoning.Effort)
	assert.Equal(t, "glm-5.3-flash", info.UpstreamModelName)
	assert.Equal(t, "high", info.ReasoningEffort)
}

func TestGLM53FlashResponsesRecoversAliasAfterBaseMapping(t *testing.T) {
	request := dto.OpenAIResponsesRequest{Model: "glm-5.3-flash"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "glm-5.3-flash-max",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "glm-5.3-flash"},
	}

	converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "glm-5.3-flash", got.Model)
	require.NotNil(t, got.Reasoning)
	assert.Equal(t, "max", got.Reasoning.Effort)
	assert.Equal(t, "max", info.ReasoningEffort)
}

func TestZhipuV4ResponsesURLStaysOnAPIv1(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://open.bigmodel.cn",
		},
	}

	requestURL, err := (&zhipu_4v.Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	assert.Equal(t, "https://open.bigmodel.cn/api/v1/responses", requestURL)
}
