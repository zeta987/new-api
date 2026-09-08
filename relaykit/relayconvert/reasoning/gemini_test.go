package reasoning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderGeminiProImageEnabledThoughts(t *testing.T) {
	visible, hidden := true, false
	for _, testCase := range []struct {
		name   string
		intent Intent
	}{
		{"enabled with default visibility", Intent{Mode: ModeEnabled}},
		{"enabled with visible thoughts", Intent{Mode: ModeEnabled, IncludeThoughts: &visible}},
		{"enabled with hidden thoughts", Intent{Mode: ModeEnabled, IncludeThoughts: &hidden}},
		{"visibility without mode", Intent{IncludeThoughts: &visible}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rendered, err := RenderGemini("gemini-3-pro-image-preview", testCase.intent, nil, 1)
			require.NoError(t, err)
			require.NotNil(t, rendered.Config)
			assert.Equal(t, testCase.intent.IncludeThoughts, rendered.Config.IncludeThoughts)
			assert.Empty(t, rendered.Config.ThinkingLevel)
			assert.Nil(t, rendered.Config.ThinkingBudget)
		})
	}
}

func TestRenderGeminiProImageRejectsStrengthControls(t *testing.T) {
	unlimited, zero, positive := -1, 0, 1024
	for _, testCase := range []struct {
		name   string
		intent Intent
	}{
		{"disabled", Intent{Mode: ModeDisabled}},
		{"adaptive", Intent{Mode: ModeAdaptive}},
		{"explicit effort", Intent{Mode: ModeEnabled, Effort: EffortLow}},
		{"unlimited budget", Intent{BudgetTokens: &unlimited}},
		{"zero budget", Intent{BudgetTokens: &zero}},
		{"positive budget", Intent{BudgetTokens: &positive}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := RenderGemini("gemini-3-pro-image-preview", testCase.intent, nil, 1)
			require.Error(t, err)
		})
	}
	_, err := RenderGemini("gemini-2.5-flash-image", Intent{Mode: ModeEnabled}, nil, 1)
	require.Error(t, err)
}
