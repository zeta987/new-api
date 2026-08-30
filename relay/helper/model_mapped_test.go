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

func glmMappingRelayInfo(modelName string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: modelName,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: modelName,
		},
	}
}
