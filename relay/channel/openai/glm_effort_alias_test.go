package openai_test

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The OpenAI-compatible adaptor must apply a GLM reasoning effort whether the
// alias survives as the upstream model name or arrives already recovered by
// channel model redirection.
func TestConvertOpenAIRequestAppliesGLMReasoningEffortAlias(t *testing.T) {
	tests := []struct {
		name             string
		channelType      int
		upstreamModel    string
		recordedEffort   string
		bodyEffort       string
		wantModel        string
		wantBodyEffort   string
		wantReasoningRaw string
	}{
		{
			name:           "openai strips the alias and sets the effort",
			channelType:    constant.ChannelTypeOpenAI,
			upstreamModel:  "glm-5.3-high",
			wantModel:      "glm-5.3",
			wantBodyEffort: "high",
		},
		{
			name:           "suffix wins over a body effort",
			channelType:    constant.ChannelTypeOpenAI,
			upstreamModel:  "glm-5.2-max",
			bodyEffort:     "high",
			wantModel:      "glm-5.2",
			wantBodyEffort: "max",
		},
		{
			name:           "recorded effort survives a redirected model",
			channelType:    constant.ChannelTypeOpenAI,
			upstreamModel:  "glm-5.3",
			recordedEffort: "low",
			wantModel:      "glm-5.3",
			wantBodyEffort: "low",
		},
		{
			name:             "openrouter rewrites the effort into its reasoning object",
			channelType:      constant.ChannelTypeOpenRouter,
			upstreamModel:    "z-ai/glm-5.3",
			recordedEffort:   "high",
			wantModel:        "z-ai/glm-5.3",
			wantReasoningRaw: `{"enabled":true,"effort":"high"}`,
		},
		{
			name:             "openrouter disables reasoning for the none alias",
			channelType:      constant.ChannelTypeOpenRouter,
			upstreamModel:    "glm-5.2-none",
			wantModel:        "glm-5.2",
			wantReasoningRaw: `{"enabled":false}`,
		},
		{
			name:           "azure keeps the alias untouched",
			channelType:    constant.ChannelTypeAzure,
			upstreamModel:  "glm-5.3-high",
			wantModel:      "glm-5.3-high",
			wantBodyEffort: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				OriginModelName:            testCase.upstreamModel,
				ModelSuffixReasoningEffort: testCase.recordedEffort,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       testCase.channelType,
					UpstreamModelName: testCase.upstreamModel,
				},
			}
			request := &dto.GeneralOpenAIRequest{
				Model:           testCase.upstreamModel,
				Messages:        []dto.Message{{Role: "user", Content: "hi"}},
				ReasoningEffort: testCase.bodyEffort,
			}

			converted, err := (&openai.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
			require.NoError(t, err)
			got, ok := converted.(*dto.GeneralOpenAIRequest)
			require.True(t, ok)

			assert.Equal(t, testCase.wantModel, got.Model)
			assert.Equal(t, testCase.wantModel, info.UpstreamModelName)
			assert.Equal(t, testCase.wantBodyEffort, got.ReasoningEffort)
			if testCase.wantReasoningRaw == "" {
				assert.Empty(t, got.Reasoning)
				return
			}
			assert.JSONEq(t, testCase.wantReasoningRaw, string(got.Reasoning))
		})
	}
}

// A client-sent "none" effort keeps its existing OpenRouter behavior, so the
// GLM disable branch never changes unrelated models.
func TestConvertOpenAIRequestKeepsClientNoneEffortOnOpenRouter(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "anthropic/claude-sonnet-4.5",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenRouter,
			UpstreamModelName: "anthropic/claude-sonnet-4.5",
		},
	}
	request := &dto.GeneralOpenAIRequest{
		Model:           "anthropic/claude-sonnet-4.5",
		Messages:        []dto.Message{{Role: "user", Content: "hi"}},
		ReasoningEffort: "none",
	}

	converted, err := (&openai.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)

	assert.Equal(t, "anthropic/claude-sonnet-4.5", got.Model)
	assert.Empty(t, got.ReasoningEffort)
	assert.Empty(t, got.Reasoning)
}
