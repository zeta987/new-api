package reasoning

import (
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestModelFamilyDiscoveryKeepsExplicitAndOpaqueNames(t *testing.T) {
	names := []string{"gpt-6-astra-pro-high", "gpt-5.1-codex-max", "gpt-6-future", "custom-high"}
	assert.Equal(t, names, ExpandOpenAIReasoningModels(names))
	settings := model_setting.GetGlobalSettings()
	original := append([]string(nil), settings.ThinkingModelBlacklist...)
	t.Cleanup(func() { settings.ThinkingModelBlacklist = original })
	settings.ThinkingModelBlacklist = append(original, "gpt-6-astra")
	assert.Equal(t, []string{"gpt-6-astra"}, ExpandOpenAIReasoningModels([]string{"gpt-6-astra", "gpt-6-astra-*"}))
}

func TestExpandEffortSuffixFamilies(t *testing.T) {
	tests := []struct {
		base string
		want []string
	}{
		{base: "claude-fable-5", want: []string{"claude-fable-5", "claude-fable-5-low", "claude-fable-5-medium", "claude-fable-5-high", "claude-fable-5-xhigh", "claude-fable-5-max"}},
		{base: "claude-opus-5", want: []string{"claude-opus-5", "claude-opus-5-low", "claude-opus-5-medium", "claude-opus-5-high", "claude-opus-5-xhigh", "claude-opus-5-max"}},
		{base: "gemini-3.8-flash", want: []string{"gemini-3.8-flash", "gemini-3.8-flash-low", "gemini-3.8-flash-medium", "gemini-3.8-flash-high"}},
		{base: "glm-5.3-flash", want: []string{"glm-5.3-flash", "glm-5.3-flash-low", "glm-5.3-flash-high", "glm-5.3-flash-max"}},
		{base: "deepseek-v4-pro", want: []string{"deepseek-v4-pro", "deepseek-v4-pro-none", "deepseek-v4-pro-low", "deepseek-v4-pro-high", "deepseek-v4-pro-max"}},
		{base: "kimi-k3", want: []string{"kimi-k3", "kimi-k3-none", "kimi-k3-low", "kimi-k3-high", "kimi-k3-max"}},
		{base: "grok-4.0", want: []string{"grok-4.0", "grok-4.0-low", "grok-4.0-medium", "grok-4.0-high"}},
		{base: "grok-4.5", want: []string{"grok-4.5", "grok-4.5-low", "grok-4.5-medium", "grok-4.5-high"}},
		{base: "grok-4.6", want: []string{"grok-4.6", "grok-4.6-low", "grok-4.6-medium", "grok-4.6-high", "grok-4.6-xhigh"}},
		{base: "grok-4.7", want: []string{"grok-4.7", "grok-4.7-low", "grok-4.7-medium", "grok-4.7-high", "grok-4.7-xhigh"}},
		// Names outside the published vocabulary expand to themselves.
		{base: "glm-5.2", want: []string{"glm-5.2"}},
		{base: "glm-5.2-max-extra", want: []string{"glm-5.2-max-extra"}},
		{base: "kimi-k2-thinking", want: []string{"kimi-k2-thinking"}},
		{base: "kimi-k2.5", want: []string{"kimi-k2.5"}},
		{base: "deepseek-chat", want: []string{"deepseek-chat"}},
		{base: "grok-4.7-preview", want: []string{"grok-4.7-preview"}},
		{base: "grok-4.7-latest", want: []string{"grok-4.7-latest"}},
		{base: "grok-4.7-20260922", want: []string{"grok-4.7-20260922"}},
		{base: "grok-4.7-search", want: []string{"grok-4.7-search"}},
		{base: "grok-4.7-reasoning", want: []string{"grok-4.7-reasoning"}},
		{base: "grok-4.7-multi-agent", want: []string{"grok-4.7-multi-agent"}},
		// The Qwen path is unchanged.
		{base: "qwen3.8-max", want: []string{"qwen3.8-max", "qwen3.8-max-none", "qwen3.8-max-low", "qwen3.8-max-medium", "qwen3.8-max-xhigh"}},
		// An explicit @ modifier names one configuration, not a family.
		{base: "kimi-k3@thinking:on", want: []string{"kimi-k3@thinking:on"}},
	}

	for _, tt := range tests {
		t.Run(tt.base, func(t *testing.T) {
			assert.Equal(t, tt.want, ExpandOpenAIReasoningModels([]string{tt.base}))
		})
	}
}

func TestExpandEffortSuffixFamiliesRespectsBlacklistAndDeduplicates(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	original := append([]string(nil), settings.ThinkingModelBlacklist...)
	t.Cleanup(func() { settings.ThinkingModelBlacklist = original })
	settings.ThinkingModelBlacklist = append(original, "deepseek-v4-pro")

	assert.Equal(t, []string{"deepseek-v4-pro"}, ExpandOpenAIReasoningModels([]string{"deepseek-v4-pro"}))

	// Registering the base and every variant by hand yields the same set as
	// registering the base alone, with no duplicates.
	manual := []string{"kimi-k3", "kimi-k3-none", "kimi-k3-low", "kimi-k3-high", "kimi-k3-max"}
	assert.Equal(t, manual, ExpandOpenAIReasoningModels(manual))
	assert.Equal(t, manual, ExpandOpenAIReasoningModels([]string{"kimi-k3"}))
}

// Every expanded variant folds back onto the base it came from. The reverse is
// deliberately not asserted: a base has many variants.
//
// GLM is absent on purpose. Its collapse lives in FormatMatchingModelName, not
// in the suffix normalizer, so EffortSuffixBaseModelName("glm-5.3-low") is ""
// while RoutingMatchModelName still resolves it to glm-5.3. The routing-level
// invariant that covers every family, GLM included, is asserted in
// controller/model_family_test.go where ModelMatchCandidates is importable.
func TestExpandedVariantsFoldBackToTheirBase(t *testing.T) {
	for _, base := range []string{
		"claude-fable-5", "claude-fable-5-1", "claude-opus-5", "claude-sonnet-5",
		"gemini-3.5-flash", "gemini-3.8-flash", "gemini-3.1-pro-preview",
		"deepseek-flash", "deepseek-v4-pro", "kimi-k3", "grok-4.0", "grok-4.5", "grok-4.6", "grok-4.7",
	} {
		t.Run(base, func(t *testing.T) {
			variants := ExpandOpenAIReasoningModels([]string{base})
			require.Greater(t, len(variants), 1)
			for _, variant := range variants[1:] {
				assert.Equal(t, base, EffortSuffixBaseModelName(variant), variant)
				assert.Equal(t, base, BaseModelName(variant), variant)
			}
		})
	}
}
