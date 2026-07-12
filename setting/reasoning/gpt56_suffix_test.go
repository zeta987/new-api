package reasoning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOpenAIReasoningModelSuffixGPT56(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantBase   string
		wantMode   string
		wantEffort string
	}{
		{
			name:       "max effort",
			model:      "gpt-5.6-luna-max",
			wantBase:   "gpt-5.6-luna",
			wantEffort: "max",
		},
		{
			name:       "pro max",
			model:      "gpt-5.6-luna-pro-max",
			wantBase:   "gpt-5.6-luna",
			wantMode:   "pro",
			wantEffort: "max",
		},
		{
			name:       "explicit standard",
			model:      "gpt-5.6-terra-standard-high",
			wantBase:   "gpt-5.6-terra",
			wantMode:   "standard",
			wantEffort: "high",
		},
		{
			name:       "standard compatibility alias",
			model:      "gpt-5.6-sol-stanard-xhigh",
			wantBase:   "gpt-5.6-sol",
			wantMode:   "standard",
			wantEffort: "xhigh",
		},
		{
			name:     "pro with default effort",
			model:    "gpt-5.6-luna-pro",
			wantBase: "gpt-5.6-luna",
			wantMode: "pro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, mode, effort, ok := ParseOpenAIReasoningModelSuffix(tt.model)

			require.True(t, ok)
			assert.Equal(t, tt.wantBase, base)
			assert.Equal(t, tt.wantMode, mode)
			assert.Equal(t, tt.wantEffort, effort)
		})
	}
}

func TestParseOpenAIReasoningModelSuffixGPT56Efforts(t *testing.T) {
	for _, model := range []string{"gpt-5.6-luna", "gpt-5.6-terra", "gpt-5.6-sol"} {
		for _, effort := range []string{"none", "low", "medium", "high", "xhigh", "max"} {
			t.Run(model+"-"+effort, func(t *testing.T) {
				base, mode, gotEffort, ok := ParseOpenAIReasoningModelSuffix(model + "-" + effort)

				require.True(t, ok)
				assert.Equal(t, model, base)
				assert.Empty(t, mode)
				assert.Equal(t, effort, gotEffort)
			})
		}
	}
}

func TestParseOpenAIReasoningModelSuffixRejectsInvalidGPT56Suffixes(t *testing.T) {
	for _, model := range []string{
		"gpt-5.6-luna-minimal",
		"gpt-5.6-luna-ultra",
		"gpt-5.6-luna-pro-ultra",
		"gpt-5.6-luna-high-pro",
		"gpt-5.6-luna-pro-max-extra",
	} {
		t.Run(model, func(t *testing.T) {
			base, mode, effort, ok := ParseOpenAIReasoningModelSuffix(model)

			assert.False(t, ok)
			assert.Equal(t, model, base)
			assert.Empty(t, mode)
			assert.Empty(t, effort)
		})
	}
}

func TestParseOpenAIReasoningEffortFromModelSuffixSupportsGPT56Max(t *testing.T) {
	effort, base := ParseOpenAIReasoningEffortFromModelSuffix("gpt-5.6-luna-max")
	assert.Equal(t, "max", effort)
	assert.Equal(t, "gpt-5.6-luna", base)

	effort, base = ParseOpenAIReasoningEffortFromModelSuffix("gpt-5.5-max")
	assert.Empty(t, effort)
	assert.Equal(t, "gpt-5.5-max", base)
}

func TestGPT56ReasoningWildcardModel(t *testing.T) {
	wildcard, ok := GPT56ReasoningWildcardModel("gpt-5.6-luna-pro-max")
	require.True(t, ok)
	assert.Equal(t, "gpt-5.6-luna-*", wildcard)

	wildcard, ok = GPT56ReasoningWildcardModel("gpt-5.6-luna-pro-ultra")
	assert.False(t, ok)
	assert.Empty(t, wildcard)

	assert.True(t, IsGPT56ReasoningWildcard("gpt-5.6-luna-*"))
	assert.False(t, IsGPT56ReasoningWildcard("gpt-5.6-luna-max"))
}
