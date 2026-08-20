package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newModelMappingContext(t *testing.T, modelMapping string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if modelMapping != "" {
		c.Set("model_mapping", modelMapping)
	}
	return c
}

// A GLM effort alias must resolve the channel redirect chain from its base
// model, because channel selection and pricing already treat it as that model.
func TestModelMappedHelperResolvesGLMEffortAlias(t *testing.T) {
	tests := []struct {
		name         string
		channelType  int
		originModel  string
		modelMapping string
		wantUpstream string
		wantEffort   string
		wantMapped   bool
	}{
		{
			name:         "openrouter redirect resolves from base model",
			channelType:  constant.ChannelTypeOpenRouter,
			originModel:  "glm-5.3-high",
			modelMapping: `{"glm-5.3":"z-ai/glm-5.3"}`,
			wantUpstream: "z-ai/glm-5.3",
			wantEffort:   "high",
			wantMapped:   true,
		},
		{
			name:         "openai without redirect strips the alias",
			channelType:  constant.ChannelTypeOpenAI,
			originModel:  "glm-5.2-none",
			wantUpstream: "glm-5.2",
			wantEffort:   "none",
			wantMapped:   true,
		},
		{
			name:         "explicit alias redirect stays authoritative",
			channelType:  constant.ChannelTypeOpenRouter,
			originModel:  "glm-5.3-max",
			modelMapping: `{"glm-5.3-max":"z-ai/glm-5.3-exp","glm-5.3":"z-ai/glm-5.3"}`,
			wantUpstream: "z-ai/glm-5.3-exp",
			wantEffort:   "max",
			wantMapped:   true,
		},
		{
			name:         "zhipu v4 keeps resolving the alias in its own adaptor",
			channelType:  constant.ChannelTypeZhipu_v4,
			originModel:  "glm-5.3-high",
			wantUpstream: "glm-5.3-high",
			wantEffort:   "",
			wantMapped:   false,
		},
		{
			name:         "channel type outside the allowlist is untouched",
			channelType:  constant.ChannelTypeAnthropic,
			originModel:  "glm-5.3-high",
			wantUpstream: "glm-5.3-high",
			wantEffort:   "",
			wantMapped:   false,
		},
		{
			name:         "unvalidated alias is untouched",
			channelType:  constant.ChannelTypeOpenAI,
			originModel:  "glm-5.3-none",
			wantUpstream: "glm-5.3-none",
			wantEffort:   "",
			wantMapped:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := newModelMappingContext(t, testCase.modelMapping)
			info := &relaycommon.RelayInfo{
				OriginModelName: testCase.originModel,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       testCase.channelType,
					UpstreamModelName: testCase.originModel,
				},
			}
			request := &dto.GeneralOpenAIRequest{Model: testCase.originModel}

			require.NoError(t, ModelMappedHelper(c, info, request))

			assert.Equal(t, testCase.wantUpstream, info.UpstreamModelName)
			assert.Equal(t, testCase.wantUpstream, request.Model)
			assert.Equal(t, testCase.wantEffort, info.ModelSuffixReasoningEffort)
			assert.Equal(t, testCase.wantEffort, info.ReasoningEffort)
			assert.Equal(t, testCase.wantMapped, info.IsModelMapped)
		})
	}
}

// Redirect chain, self-redirect, and cycle behavior must survive the alias
// normalization added ahead of the chain.
func TestModelMappedHelperRedirectChainBehavior(t *testing.T) {
	tests := []struct {
		name         string
		originModel  string
		modelMapping string
		wantUpstream string
		wantMapped   bool
		wantErr      string
	}{
		{
			name:         "chained redirect uses the tail model",
			originModel:  "a",
			modelMapping: `{"a":"b","b":"c"}`,
			wantUpstream: "c",
			wantMapped:   true,
		},
		{
			name:         "self redirect on the requested model changes nothing",
			originModel:  "a",
			modelMapping: `{"a":"a"}`,
			wantUpstream: "a",
			wantMapped:   false,
		},
		{
			name:         "self redirect at the chain tail still maps",
			originModel:  "a",
			modelMapping: `{"a":"b","b":"b"}`,
			wantUpstream: "b",
			wantMapped:   true,
		},
		{
			name:         "cycle is rejected",
			originModel:  "a",
			modelMapping: `{"a":"b","b":"a"}`,
			wantErr:      "model_mapping_contains_cycle",
		},
		{
			name:         "invalid mapping json is rejected",
			originModel:  "a",
			modelMapping: `not-json`,
			wantErr:      "unmarshal_model_mapping_failed",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			c := newModelMappingContext(t, testCase.modelMapping)
			info := &relaycommon.RelayInfo{
				OriginModelName: testCase.originModel,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       constant.ChannelTypeOpenAI,
					UpstreamModelName: testCase.originModel,
				},
			}
			request := &dto.GeneralOpenAIRequest{Model: testCase.originModel}

			err := ModelMappedHelper(c, info, request)
			if testCase.wantErr != "" {
				require.EqualError(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.wantUpstream, info.UpstreamModelName)
			assert.Equal(t, testCase.wantUpstream, request.Model)
			assert.Equal(t, testCase.wantMapped, info.IsModelMapped)
		})
	}
}
