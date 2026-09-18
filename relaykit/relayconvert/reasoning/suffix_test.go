package reasoning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDeepSeekV4ThinkingSuffixSupportsLow(t *testing.T) {
	for _, model := range []string{"deepseek-v4-flash-low", "deepseek-v4-pro-low"} {
		t.Run(model, func(t *testing.T) {
			baseModel, thinkingType, effort, ok := ParseDeepSeekV4ThinkingSuffix(model)

			require.True(t, ok)
			assert.Equal(t, model[:len(model)-len("-low")], baseModel)
			assert.Equal(t, "enabled", thinkingType)
			assert.Equal(t, "low", effort)
		})
	}
}

func TestParseGLMReasoningEffortSuffix(t *testing.T) {
	tests := []struct {
		model  string
		base   string
		effort string
		ok     bool
	}{
		{model: "glm-5.2-none", base: "glm-5.2", effort: "none", ok: true},
		{model: "glm-5.2-minimal", base: "glm-5.2", effort: "minimal", ok: true},
		{model: "glm-5.2-low", base: "glm-5.2", effort: "low", ok: true},
		{model: "glm-5.2-medium", base: "glm-5.2", effort: "medium", ok: true},
		{model: "glm-5.2-high", base: "glm-5.2", effort: "high", ok: true},
		{model: "glm-5.2-xhigh", base: "glm-5.2", effort: "xhigh", ok: true},
		{model: "glm-5.2-max", base: "glm-5.2", effort: "max", ok: true},
		{model: "glm-5.3-high", base: "glm-5.3", effort: "high", ok: true},
		{model: "glm-5.3-flash-low", base: "glm-5.3-flash", effort: "low", ok: true},
		{model: "glm-5.3-flash-high", base: "glm-5.3-flash", effort: "high", ok: true},
		{model: "glm-5.3-flash-max", base: "glm-5.3-flash", effort: "max", ok: true},
		{model: "glm-future-model-xhigh", base: "glm-future-model", effort: "xhigh", ok: true},
		{model: "glm-5.3-flash", base: "glm-5.3-flash"},
		{model: "glm-5.3-flash-fast", base: "glm-5.3-flash-fast"},
		{model: "glm-5.3-flash-max-extra", base: "glm-5.3-flash-max-extra"},
		{model: "glm-low", base: "glm-low"},
		{model: "glm--low", base: "glm--low"},
		{model: "GLM-5.3-high", base: "GLM-5.3-high"},
		{model: "custom-glm-5.3-high", base: "custom-glm-5.3-high"},
	}

	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			base, effort, ok := ParseGLMReasoningEffortSuffix(test.model)
			assert.Equal(t, test.base, base)
			assert.Equal(t, test.effort, effort)
			assert.Equal(t, test.ok, ok)
		})
	}
}

func TestIsGLMReasoningEffortModel(t *testing.T) {
	for _, model := range []string{"glm-5", "glm-5.1", "glm-5.2", "glm-5.3-flash", "glm-future-model", "glm-5.3-flash-fast"} {
		assert.True(t, IsGLMReasoningEffortModel(model))
	}
	for _, model := range []string{"glm-5.2-max", "glm-5.3-flash-low", "glm-low", "glm--low", "GLM-5.3", "custom-glm-5.3"} {
		assert.False(t, IsGLMReasoningEffortModel(model))
	}
}

func TestGLMBasePredicateAndSuffixParserAreMutuallyExclusive(t *testing.T) {
	for _, model := range []string{"glm-5.3-flash", "glm-5.3-flash-low", "glm-future-model-xhigh", "glm-low", "glm--low"} {
		_, _, parsed := ParseGLMReasoningEffortSuffix(model)
		assert.False(t, parsed && IsGLMReasoningEffortModel(model), model)
	}
}

func TestParseGeminiModelSuffixNoThinkingDisablesReasoning(t *testing.T) {
	t.Parallel()

	base, intent, found, err := ParseGeminiModelSuffix("gemini-2.5-flash-nothinking", true)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "gemini-2.5-flash", base)
	assert.Equal(t, ModeDisabled, intent.Mode)
	assert.Equal(t, EffortNone, intent.Effort)
	assert.Equal(t, SourceSuffix, intent.Source)
}

func TestParseKnownProviderModelSuffix(t *testing.T) {
	t.Parallel()

	preserveQwenMax := func(name string) bool { return name == "qwen-max" || name == "vendor/qwen-max" }

	tests := []struct {
		name               string
		model              string
		allowThinkingAlias bool
		wantBase           string
		wantFound          bool
		wantMode           Mode
		wantEffort         Effort
		wantBudget         *int
		wantErr            bool
	}{
		{
			name:               "claude thinking alias",
			model:              "claude-3-7-sonnet-thinking",
			allowThinkingAlias: true,
			wantBase:           "claude-3-7-sonnet",
			wantFound:          true,
			wantMode:           ModeEnabled,
		},
		{
			name:               "claude nothinking alias",
			model:              "claude-3-7-sonnet-nothinking",
			allowThinkingAlias: true,
			wantBase:           "claude-3-7-sonnet",
			wantFound:          true,
			wantMode:           ModeDisabled,
			wantEffort:         EffortNone,
		},
		{
			name:               "claude thinking budget",
			model:              "claude-3-7-sonnet-thinking-8192",
			allowThinkingAlias: true,
			wantBase:           "claude-3-7-sonnet",
			wantFound:          true,
			wantBudget:         intPtr(8192),
		},
		{
			name:               "claude effort tail",
			model:              "claude-opus-4-8-high",
			allowThinkingAlias: true,
			wantBase:           "claude-opus-4-8",
			wantFound:          true,
			wantMode:           ModeEnabled,
			wantEffort:         EffortHigh,
		},
		{
			name:               "gemini thinking alias",
			model:              "gemini-2.5-flash-thinking",
			allowThinkingAlias: true,
			wantBase:           "gemini-2.5-flash",
			wantFound:          true,
			wantMode:           ModeEnabled,
		},
		{
			name:               "malformed thinking budget",
			model:              "claude-3-7-sonnet-thinking-abc",
			allowThinkingAlias: true,
			wantErr:            true,
		},
		{
			name:      "unknown openai-compatible name is untouched",
			model:     "gpt-4o-mini",
			wantBase:  "gpt-4o-mini",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, intent, found, err := ParseKnownProviderModelSuffix(tt.model, tt.allowThinkingAlias)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantBase, base)
			assert.Equal(t, tt.wantMode, intent.Mode)
			assert.Equal(t, tt.wantEffort, intent.Effort)
			if tt.wantBudget != nil {
				require.NotNil(t, intent.BudgetTokens)
				assert.Equal(t, *tt.wantBudget, *intent.BudgetTokens)
			} else {
				assert.Nil(t, intent.BudgetTokens)
			}
		})
	}

	t.Run("openai effort tail", func(t *testing.T) {
		t.Parallel()
		effort, base := ParseOpenAIReasoningEffortFromModelSuffix("gpt-5.6-sol-high", nil)
		assert.Equal(t, "high", effort)
		assert.Equal(t, "gpt-5.6-sol", base)
	})

	t.Run("preserve effort tail on real model id", func(t *testing.T) {
		t.Parallel()
		effort, base := ParseOpenAIReasoningEffortFromModelSuffix("qwen-max", preserveQwenMax)
		assert.Empty(t, effort)
		assert.Equal(t, "qwen-max", base)
	})

	t.Run("preserve effort tail with vendor prefix", func(t *testing.T) {
		t.Parallel()
		effort, base := ParseOpenAIReasoningEffortFromModelSuffix("vendor/qwen-max", preserveQwenMax)
		assert.Empty(t, effort)
		assert.Equal(t, "vendor/qwen-max", base)
	})

	t.Run("preserve gpt-5.1-codex-max with callback", func(t *testing.T) {
		t.Parallel()
		preserve := func(name string) bool { return name == "gpt-5.1-codex-max" }
		effort, base := ParseOpenAIReasoningEffortFromModelSuffix("gpt-5.1-codex-max", preserve)
		assert.Empty(t, effort)
		assert.Equal(t, "gpt-5.1-codex-max", base)
	})

	t.Run("splits gpt-5.1-codex-max without callback", func(t *testing.T) {
		t.Parallel()
		effort, base := ParseOpenAIReasoningEffortFromModelSuffix("gpt-5.1-codex-max", nil)
		assert.Equal(t, "max", effort)
		assert.Equal(t, "gpt-5.1-codex", base)
	})
}

func TestParseThinkingModifier(t *testing.T) {
	t.Parallel()

	on, ok := ParseThinkingModifier("on")
	require.True(t, ok)
	assert.Equal(t, ModeEnabled, on.Mode)

	adaptive, ok := ParseThinkingModifier("Adaptive")
	require.True(t, ok)
	assert.Equal(t, ModeAdaptive, adaptive.Mode)

	off, ok := ParseThinkingModifier("off")
	require.True(t, ok)
	assert.Equal(t, ModeDisabled, off.Mode)
	assert.Equal(t, EffortNone, off.Effort)

	zero, ok := ParseThinkingModifier("0")
	require.True(t, ok)
	assert.Equal(t, ModeDisabled, zero.Mode)

	budget, ok := ParseThinkingModifier("8192")
	require.True(t, ok)
	require.NotNil(t, budget.BudgetTokens)
	assert.Equal(t, 8192, *budget.BudgetTokens)
	assert.Equal(t, ModeEnabled, budget.Mode)

	dynamic, ok := ParseThinkingModifier("-1")
	require.True(t, ok)
	require.NotNil(t, dynamic.BudgetTokens)
	assert.Equal(t, -1, *dynamic.BudgetTokens)

	_, ok = ParseThinkingModifier("-2")
	assert.False(t, ok)
	_, ok = ParseThinkingModifier("enabled")
	assert.False(t, ok)
}

func intPtr(v int) *int {
	return &v
}
