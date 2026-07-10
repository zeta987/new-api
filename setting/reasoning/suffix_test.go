package reasoning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsClaudeAdaptiveThinkingModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "claude-3-5-sonnet-20241022", want: false},
		{model: "claude-3-7-sonnet-20250219", want: false},
		{model: "claude-sonnet-4-20250514", want: false},
		{model: "claude-sonnet-4-5-20250929", want: false},
		{model: "claude-opus-4-6", want: true},
		{model: "claude-sonnet-4-6-high", want: true},
		{model: "claude-opus-4-7-thinking", want: true},
		{model: "claude-opus-4-10-max", want: true},
		{model: "claude-fable-5", want: true},
		{model: "claude-fable-5-xhigh", want: true},
		{model: "claude-sonnet-5", want: true},
		{model: "claude-opus-5-max", want: true},
		{model: "claude-haiku-5-low", want: true},
		{model: "claude-mythos-5-medium", want: true},
		{model: "not-claude-fable-5", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			require.Equal(t, tt.want, IsClaudeAdaptiveThinkingModel(tt.model))
		})
	}
}

func TestIsClaudePost46AdaptiveThinkingModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "claude-opus-4-6", want: false},
		{model: "claude-sonnet-4-6-high", want: false},
		{model: "claude-opus-4-7", want: true},
		{model: "claude-opus-4-8-thinking", want: true},
		{model: "claude-opus-4-10-max", want: true},
		{model: "claude-fable-5", want: true},
		{model: "claude-fable-5-medium", want: true},
		{model: "claude-sonnet-5", want: true},
		{model: "claude-opus-5-max", want: true},
		{model: "claude-haiku-5-low", want: true},
		{model: "claude-mythos-5-medium", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			require.Equal(t, tt.want, IsClaudePost46AdaptiveThinkingModel(tt.model))
		})
	}
}

func TestIsClaudeEffortLevel(t *testing.T) {
	for _, effort := range []string{"low", "medium", "high", "xhigh", "max"} {
		t.Run(effort, func(t *testing.T) {
			require.True(t, IsClaudeEffortLevel(effort))
		})
	}

	require.False(t, IsClaudeEffortLevel("minimal"))
	require.False(t, IsClaudeEffortLevel(""))
}

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
