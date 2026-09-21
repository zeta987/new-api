package helper

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelMappedHelperFallsBackFromGLMAliasToBaseMapping(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("model_mapping", `{"glm-5.3-flash":"z-ai/glm-5.3-flash"}`)
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.3-flash-high"}
	info := glmMappingRelayInfo("glm-5.3-flash-high")

	err := ModelMappedHelper(c, info, request)

	require.NoError(t, err)
	assert.True(t, info.IsModelMapped)
	assert.Equal(t, "glm-5.3-flash-high", info.OriginModelName)
	assert.Equal(t, "z-ai/glm-5.3-flash", info.UpstreamModelName)
	assert.Equal(t, "z-ai/glm-5.3-flash", request.Model)
}

func TestModelMappedHelperPrefersExactGLMAliasMapping(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("model_mapping", `{"glm-5.3-flash":"z-ai/glm-5.3-flash","glm-5.3-flash-max":"vendor/special-max"}`)
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.3-flash-max"}
	info := glmMappingRelayInfo("glm-5.3-flash-max")

	err := ModelMappedHelper(c, info, request)

	require.NoError(t, err)
	assert.True(t, info.IsModelMapped)
	assert.Equal(t, "vendor/special-max", info.UpstreamModelName)
	assert.Equal(t, "vendor/special-max", request.Model)
}

func TestModelMappedHelperLeavesCompleteGLMAliasWithoutBaseMapping(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("model_mapping", `{"another-model":"provider/model"}`)
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.3-flash-low"}
	info := glmMappingRelayInfo("glm-5.3-flash-low")

	err := ModelMappedHelper(c, info, request)

	require.NoError(t, err)
	assert.False(t, info.IsModelMapped)
	assert.Equal(t, "glm-5.3-flash-low", info.UpstreamModelName)
	assert.Equal(t, "glm-5.3-flash-low", request.Model)
}

func TestModelMappedHelperDetectsCycleAfterGLMBaseFallback(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("model_mapping", `{"glm-5.3-flash":"mapped-a","mapped-a":"glm-5.3-flash"}`)
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.3-flash-high"}
	info := glmMappingRelayInfo("glm-5.3-flash-high")

	err := ModelMappedHelper(c, info, request)

	require.EqualError(t, err, "model_mapping_contains_cycle")
}

// An identity mapping on the base model is the shape that strips the effort
// suffix from the upstream name: the base fallback matches, then the second
// pass exits on mappedModel == currentModel with IsModelMapped still true.
func TestModelMappedHelperIdentityBaseMappingDropsEffortSuffix(t *testing.T) {
	tests := []struct {
		origin  string
		mapping string
		want    string
	}{
		{origin: "kimi-k3-high", mapping: `{"kimi-k3":"kimi-k3"}`, want: "kimi-k3"},
		{origin: "deepseek-v4-pro-high", mapping: `{"deepseek-v4-pro":"deepseek-v4-pro"}`, want: "deepseek-v4-pro"},
		{origin: "kimi-k3-high", mapping: `{"kimi-k3":"kimi-k3-0501"}`, want: "kimi-k3-0501"},
	}

	for _, tt := range tests {
		t.Run(tt.origin+" -> "+tt.want, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("model_mapping", tt.mapping)
			request := &dto.GeneralOpenAIRequest{Model: tt.origin}
			info := glmMappingRelayInfo(tt.origin)

			require.NoError(t, ModelMappedHelper(c, info, request))
			assert.True(t, info.IsModelMapped)
			assert.Equal(t, tt.origin, info.OriginModelName)
			assert.Equal(t, tt.want, info.UpstreamModelName)
			assert.Equal(t, tt.want, request.Model)
		})
	}
}

func glmMappingRelayInfo(modelName string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: modelName,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: modelName,
		},
	}
}
