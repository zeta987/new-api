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
	t.Run("moonshot defaults base model to none effort", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "kimi-k3"}
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
		assert.Equal(t, "none", got.ReasoningEffort)
		assert.Equal(t, "kimi-k3", info.UpstreamModelName)
		assert.Equal(t, "none", info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"kimi-k3","reasoning_effort":"none"}`, payload)
	})

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
		request := &dto.GeneralOpenAIRequest{
			Model:            "kimi-k3-max",
			Temperature:      common.GetPointer[float64](0.7),
			TopP:             common.GetPointer[float64](1),
			TopK:             common.GetPointer(40),
			N:                common.GetPointer(2),
			FrequencyPenalty: common.GetPointer[float64](0),
			PresencePenalty:  common.GetPointer[float64](0),
		}
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
		assert.Nil(t, got.Temperature)
		assert.Nil(t, got.TopP)
		assert.Nil(t, got.TopK)
		assert.Nil(t, got.N)
		if assert.NotNil(t, got.FrequencyPenalty) {
			assert.Zero(t, *got.FrequencyPenalty)
		}
		if assert.NotNil(t, got.PresencePenalty) {
			assert.Zero(t, *got.PresencePenalty)
		}
		assert.JSONEq(t, `{"model":"kimi-k3","reasoning_effort":"max","frequency_penalty":0,"presence_penalty":0}`, payload)
	})

	t.Run("moonshot no longer converts none suffix", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "kimi-k3-none"}
		info := &relaycommon.RelayInfo{
			OriginModelName: "kimi-k3-none",
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelType:       constant.ChannelTypeMoonshot,
				UpstreamModelName: "kimi-k3-none",
			},
		}

		converted, err := (&moonshot.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		got, payload := requireConvertedOpenAIRequest(t, converted, err)

		assert.Equal(t, "kimi-k3-none", got.Model)
		assert.Empty(t, got.ReasoningEffort)
		assert.Equal(t, "kimi-k3-none", info.UpstreamModelName)
		assert.Empty(t, info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"kimi-k3-none"}`, payload)
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

func TestMoonshotModelListIncludesKimiK3Variants(t *testing.T) {
	modelList := (&moonshot.Adaptor{}).GetModelList()
	for _, model := range []string{"kimi-k3", "kimi-k3-max"} {
		t.Run(model, func(t *testing.T) {
			assert.Contains(t, modelList, model)
		})
	}
	assert.NotContains(t, modelList, "kimi-k3-none")
}
