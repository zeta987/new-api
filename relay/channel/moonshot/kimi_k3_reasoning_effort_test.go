package moonshot_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/moonshot"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKimiK3ReasoningEffortConversion(t *testing.T) {
	t.Run("moonshot passes explicit top-level effort", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{
			Model:           "kimi-k3",
			ReasoningEffort: "max",
		}
		info := &relaycommon.RelayInfo{
			OriginModelName: "kimi-k3",
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeMoonshot,
				UpstreamModelName: "kimi-k3",
			},
		}

		converted, err := (&moonshot.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		got, payload := requireConvertedOpenAIRequest(t, converted, err)

		assert.Equal(t, "kimi-k3", got.Model)
		assert.Equal(t, "max", got.ReasoningEffort)
		assert.Equal(t, "kimi-k3", info.UpstreamModelName)
		assert.Empty(t, info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"kimi-k3","reasoning_effort":"max"}`, payload)
	})

	t.Run("moonshot converts max suffix to official payload", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "kimi-k3-max"}
		info := &relaycommon.RelayInfo{
			OriginModelName: "kimi-k3-max",
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeMoonshot,
				UpstreamModelName: "kimi-k3-max",
			},
		}

		converted, err := (&moonshot.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		got, payload := requireConvertedOpenAIRequest(t, converted, err)

		assert.Equal(t, "kimi-k3", got.Model)
		assert.Equal(t, "max", got.ReasoningEffort)
		assert.Equal(t, "kimi-k3", info.UpstreamModelName)
		assert.Equal(t, "max", info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"kimi-k3","reasoning_effort":"max"}`, payload)
	})

	t.Run("openai leaves kimi suffix unchanged", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "kimi-k3-max"}
		info := &relaycommon.RelayInfo{
			OriginModelName: "kimi-k3-max",
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeOpenAI,
				UpstreamModelName: "kimi-k3-max",
			},
		}

		converted, err := (&openai.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		got, payload := requireConvertedOpenAIRequest(t, converted, err)

		assert.Equal(t, "kimi-k3-max", got.Model)
		assert.Empty(t, got.ReasoningEffort)
		assert.Equal(t, "kimi-k3-max", info.UpstreamModelName)
		assert.Empty(t, info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"kimi-k3-max"}`, payload)
	})

	t.Run("moonshot preserves unsupported effort suffix", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "kimi-k3-high"}
		info := &relaycommon.RelayInfo{
			OriginModelName: "kimi-k3-high",
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeMoonshot,
				UpstreamModelName: "kimi-k3-high",
			},
		}

		converted, err := (&moonshot.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		got, payload := requireConvertedOpenAIRequest(t, converted, err)

		assert.Equal(t, "kimi-k3-high", got.Model)
		assert.Empty(t, got.ReasoningEffort)
		assert.Equal(t, "kimi-k3-high", info.UpstreamModelName)
		assert.Empty(t, info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"kimi-k3-high"}`, payload)
	})
}

func requireConvertedOpenAIRequest(t *testing.T, converted any, conversionErr error) (*dto.GeneralOpenAIRequest, string) {
	t.Helper()
	require.NoError(t, conversionErr)
	request, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	payload, err := common.Marshal(request)
	require.NoError(t, err)
	return request, string(payload)
}

func TestMoonshotModelListIncludesKimiK3(t *testing.T) {
	assert.Contains(t, (&moonshot.Adaptor{}).GetModelList(), "kimi-k3")
}
