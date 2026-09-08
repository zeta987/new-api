package helper

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenRouterGLMEffortUsesBaseMappingAndChannelOverride(t *testing.T) {
	tests := []struct {
		alias  string
		effort string
	}{
		{alias: "glm-5.3-flash-none", effort: "none"},
		{alias: "glm-5.3-flash-minimal", effort: "minimal"},
		{alias: "glm-5.3-flash-low", effort: "low"},
		{alias: "glm-5.3-flash-medium", effort: "medium"},
		{alias: "glm-5.3-flash-high", effort: "high"},
		{alias: "glm-5.3-flash-xhigh", effort: "xhigh"},
		{alias: "glm-5.3-flash-max", effort: "max"},
	}

	for _, testCase := range tests {
		t.Run(testCase.effort, func(t *testing.T) {
			c, _ := gin.CreateTestContext(nil)
			c.Set("model_mapping", `{"glm-5.3-flash":"z-ai/glm-5.3-flash"}`)
			request := &dto.GeneralOpenAIRequest{
				Model:     testCase.alias,
				Reasoning: []byte(`{"enabled":false,"max_tokens":0}`),
			}
			info := &relaycommon.RelayInfo{
				OriginModelName: testCase.alias,
				RelayFormat:     types.RelayFormatOpenAI,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       constant.ChannelTypeOpenRouter,
					UpstreamModelName: testCase.alias,
					ParamOverride:     openRouterGLMEffortOverride(t),
				},
			}

			require.NoError(t, ModelMappedHelper(c, info, request))
			payload, err := common.Marshal(request)
			require.NoError(t, err)
			payload, err = relaycommon.ApplyParamOverrideWithRelayInfo(payload, info)
			require.NoError(t, err)

			assert.Equal(t, "z-ai/glm-5.3-flash", gjson.GetBytes(payload, "model").String())
			assert.Equal(t, testCase.effort, gjson.GetBytes(payload, "reasoning.effort").String())
			assert.False(t, gjson.GetBytes(payload, "reasoning.enabled").Exists())
			assert.False(t, gjson.GetBytes(payload, "reasoning.max_tokens").Exists())
			assert.Equal(t, testCase.effort, info.ReasoningEffort)
			assert.Equal(t, testCase.alias, info.OriginModelName)
		})
	}
}

func openRouterGLMEffortOverride(t *testing.T) map[string]interface{} {
	t.Helper()
	const config = `{
  "operations": [
    {"path":"reasoning","mode":"set","value":{"effort":"none"},"conditions":[{"path":"original_model","mode":"prefix","value":"glm-"},{"path":"original_model","mode":"suffix","value":"-none"}],"logic":"AND"},
    {"path":"reasoning","mode":"set","value":{"effort":"minimal"},"conditions":[{"path":"original_model","mode":"prefix","value":"glm-"},{"path":"original_model","mode":"suffix","value":"-minimal"}],"logic":"AND"},
    {"path":"reasoning","mode":"set","value":{"effort":"low"},"conditions":[{"path":"original_model","mode":"prefix","value":"glm-"},{"path":"original_model","mode":"suffix","value":"-low"}],"logic":"AND"},
    {"path":"reasoning","mode":"set","value":{"effort":"medium"},"conditions":[{"path":"original_model","mode":"prefix","value":"glm-"},{"path":"original_model","mode":"suffix","value":"-medium"}],"logic":"AND"},
    {"path":"reasoning","mode":"set","value":{"effort":"high"},"conditions":[{"path":"original_model","mode":"prefix","value":"glm-"},{"path":"original_model","mode":"suffix","value":"-high"}],"logic":"AND"},
    {"path":"reasoning","mode":"set","value":{"effort":"xhigh"},"conditions":[{"path":"original_model","mode":"prefix","value":"glm-"},{"path":"original_model","mode":"suffix","value":"-xhigh"}],"logic":"AND"},
    {"path":"reasoning","mode":"set","value":{"effort":"max"},"conditions":[{"path":"original_model","mode":"prefix","value":"glm-"},{"path":"original_model","mode":"suffix","value":"-max"}],"logic":"AND"}
  ]
}`

	var override map[string]interface{}
	require.NoError(t, common.Unmarshal([]byte(config), &override))
	return override
}
