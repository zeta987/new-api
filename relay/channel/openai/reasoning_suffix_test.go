package openai

import (
	"slices"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestAppliesGPT56ReasoningSuffix(t *testing.T) {
	tests := []struct {
		name          string
		request       dto.OpenAIResponsesRequest
		wantModel     string
		wantMode      string
		wantEffort    string
		wantSummary   string
		wantReasoning bool
	}{
		{
			name:          "max keeps mode omitted",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.6-luna-max"},
			wantModel:     "gpt-5.6-luna",
			wantEffort:    "max",
			wantReasoning: true,
		},
		{
			name: "body mode works without a suffix",
			request: dto.OpenAIResponsesRequest{
				Model:     "gpt-5.6-luna",
				Reasoning: &dto.Reasoning{Mode: []byte(`"pro"`), Summary: "auto"},
			},
			wantModel:     "gpt-5.6-luna",
			wantMode:      "pro",
			wantSummary:   "auto",
			wantReasoning: true,
		},
		{
			name:          "pro max sets both fields",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.6-luna-pro-max"},
			wantModel:     "gpt-5.6-luna",
			wantMode:      "pro",
			wantEffort:    "max",
			wantReasoning: true,
		},
		{
			name: "effort suffix preserves body mode",
			request: dto.OpenAIResponsesRequest{
				Model:     "gpt-5.6-luna-high",
				Reasoning: &dto.Reasoning{Mode: []byte(`"pro"`), Effort: "low", Summary: "auto"},
			},
			wantModel:     "gpt-5.6-luna",
			wantMode:      "pro",
			wantEffort:    "high",
			wantSummary:   "auto",
			wantReasoning: true,
		},
		{
			name: "explicit standard suffix overrides body mode",
			request: dto.OpenAIResponsesRequest{
				Model:     "gpt-5.6-luna-standard-medium",
				Reasoning: &dto.Reasoning{Mode: []byte(`"pro"`), Effort: "low"},
			},
			wantModel:     "gpt-5.6-luna",
			wantMode:      "standard",
			wantEffort:    "medium",
			wantReasoning: true,
		},
		{
			name:          "invalid suffix is untouched",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.6-luna-pro-ultra"},
			wantModel:     "gpt-5.6-luna-pro-ultra",
			wantReasoning: false,
		},
		{
			name:          "minimal is not a GPT-5.6 effort",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.6-luna-minimal"},
			wantModel:     "gpt-5.6-luna-minimal",
			wantReasoning: false,
		},
		{
			name:          "codex max model remains intact",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.1-codex-max"},
			wantModel:     "gpt-5.1-codex-max",
			wantReasoning: false,
		},
		{
			name:          "unrelated pro model remains intact",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.4-pro"},
			wantModel:     "gpt-5.4-pro",
			wantReasoning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tt.request.Model},
			}
			converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, tt.request)
			require.NoError(t, err)

			got, ok := converted.(dto.OpenAIResponsesRequest)
			require.True(t, ok)
			assert.Equal(t, tt.wantModel, got.Model)
			assert.Equal(t, tt.wantModel, info.UpstreamModelName)
			if !tt.wantReasoning {
				assert.Nil(t, got.Reasoning)
				return
			}
			require.NotNil(t, got.Reasoning)
			if tt.wantMode == "" {
				assert.Nil(t, got.Reasoning.Mode)
			} else {
				require.NotNil(t, got.Reasoning.Mode)
				assert.Equal(t, `"`+tt.wantMode+`"`, string(got.Reasoning.Mode))
			}
			assert.Equal(t, tt.wantEffort, got.Reasoning.Effort)
			assert.Equal(t, tt.wantSummary, got.Reasoning.Summary)
			assert.Equal(t, tt.wantEffort, info.ReasoningEffort)

			encoded, err := common.Marshal(got)
			require.NoError(t, err)
			if tt.wantMode == "" {
				assert.NotContains(t, string(encoded), `"mode"`)
			} else {
				assert.Contains(t, string(encoded), `"mode":"`+tt.wantMode+`"`)
			}
		})
	}
}

func TestConvertOpenAIResponsesRequestUsesOriginalModelSuffixAfterMapping(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.6-luna-pro-max",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.6-terra"},
	}
	request := dto.OpenAIResponsesRequest{Model: "gpt-5.6-terra"}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.NotNil(t, got.Reasoning)

	assert.Equal(t, "gpt-5.6-terra", got.Model)
	assert.Equal(t, "gpt-5.6-terra", info.UpstreamModelName)
	require.NotNil(t, got.Reasoning.Mode)
	assert.Equal(t, `"pro"`, string(got.Reasoning.Mode))
	assert.Equal(t, "max", got.Reasoning.Effort)
}

func TestModelListIncludesGPT56Models(t *testing.T) {
	for _, model := range []string{"gpt-5.6-luna", "gpt-5.6-terra", "gpt-5.6-sol"} {
		assert.Truef(t, slices.Contains(ModelList, model), "ModelList is missing %s", model)
	}
}
