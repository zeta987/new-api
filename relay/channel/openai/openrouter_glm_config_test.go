package openai_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	openaiadaptor "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenRouterGLMEffortPipelineUsesOnlyMappingAndOverride(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	c.Set("model_mapping", `{"glm-5.3-flash":"z-ai/glm-5.3-flash"}`)
	request := &dto.GeneralOpenAIRequest{
		Model:     "glm-5.3-flash-high",
		Reasoning: []byte(`{"enabled":false,"max_tokens":0}`),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "glm-5.3-flash-high",
		RelayFormat:     types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			UpstreamModelName: "glm-5.3-flash-high",
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"path":  "reasoning",
						"mode":  "set",
						"value": map[string]interface{}{"effort": "high"},
						"conditions": []interface{}{
							map[string]interface{}{"path": "original_model", "mode": "prefix", "value": "glm-"},
							map[string]interface{}{"path": "original_model", "mode": "suffix", "value": "-high"},
						},
						"logic": "AND",
					},
				},
			},
		},
	}

	require.NoError(t, helper.ModelMappedHelper(c, info, request))
	converted, err := (&openaiadaptor.Adaptor{}).ConvertOpenAIRequest(c, info, request)
	require.NoError(t, err)
	payload, err := common.Marshal(converted)
	require.NoError(t, err)
	payload, err = relaycommon.ApplyParamOverrideWithRelayInfo(payload, info)
	require.NoError(t, err)

	assert.Equal(t, "z-ai/glm-5.3-flash", gjson.GetBytes(payload, "model").String())
	assert.Equal(t, "high", gjson.GetBytes(payload, "reasoning.effort").String())
	assert.False(t, gjson.GetBytes(payload, "reasoning.enabled").Exists())
	assert.False(t, gjson.GetBytes(payload, "reasoning.max_tokens").Exists())
	assert.Equal(t, "high", info.ReasoningEffort)
}
