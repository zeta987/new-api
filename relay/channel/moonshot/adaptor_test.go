package moonshot

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

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
