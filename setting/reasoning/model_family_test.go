package reasoning

import (
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
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
