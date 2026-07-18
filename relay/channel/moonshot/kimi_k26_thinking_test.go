package moonshot_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/moonshot"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKimiK26ThinkingSuffixConversion(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:       "kimi-k2.6-thinking",
		Temperature: common.GetPointer[float64](0.7),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "kimi-k2.6-thinking",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeMoonshot,
			UpstreamModelName: "kimi-k2.6-thinking",
		},
	}

	converted, err := (&moonshot.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	got, payload := requireConvertedOpenAIRequest(t, converted, err)

	assert.Equal(t, "kimi-k2.6", got.Model)
	assert.JSONEq(t, `{"type":"enabled"}`, string(got.THINKING))
	require.NotNil(t, got.Temperature)
	assert.Equal(t, 1.0, *got.Temperature)
	assert.Equal(t, "kimi-k2.6", info.UpstreamModelName)
	assert.JSONEq(t, `{"model":"kimi-k2.6","temperature":1,"thinking":{"type":"enabled"}}`, payload)
}

func TestMoonshotModelListIncludesKimiK26ThinkingVariants(t *testing.T) {
	modelList := (&moonshot.Adaptor{}).GetModelList()

	assert.Contains(t, modelList, "kimi-k2.6")
	assert.Contains(t, modelList, "kimi-k2.6-thinking")
}
