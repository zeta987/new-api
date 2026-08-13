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

func TestParseGLM52ReasoningEffortSuffix(t *testing.T) {
	tests := []struct {
		model  string
		base   string
		effort string
		ok     bool
	}{
		{model: "glm-5.2-none", base: "glm-5.2", effort: "none", ok: true},
		{model: "glm-5.2-high", base: "glm-5.2", effort: "high", ok: true},
		{model: "glm-5.2-max", base: "glm-5.2", effort: "max", ok: true},
		{model: "glm-5.2", base: "glm-5.2"},
		{model: "glm-5.2-low", base: "glm-5.2-low"},
		{model: "glm-5.2-xhigh", base: "glm-5.2-xhigh"},
		{model: "glm-5.2-max-extra", base: "glm-5.2-max-extra"},
		{model: "glm-5.1-high", base: "glm-5.1-high"},
		{model: "GLM-5.2-high", base: "GLM-5.2-high"},
	}

	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			base, effort, ok := ParseGLM52ReasoningEffortSuffix(test.model)
			assert.Equal(t, test.base, base)
			assert.Equal(t, test.effort, effort)
			assert.Equal(t, test.ok, ok)
		})
	}
}
