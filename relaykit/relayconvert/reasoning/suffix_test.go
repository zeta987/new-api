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
