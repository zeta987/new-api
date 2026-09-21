package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
)

func TestFormatMatchingModelNameDoesNotStripBase(t *testing.T) {
	assert.Equal(t, "qwen3-max@thinking:on", FormatMatchingModelName("qwen3-max@thinking:on"))
	assert.Equal(t, "claude-3-7-sonnet-thinking", FormatMatchingModelName("claude-3-7-sonnet-thinking"))
	assert.Equal(t, "gemini-2.5-flash-thinking-*", FormatMatchingModelName("gemini-2.5-flash-thinking-8192"))
	assert.Equal(t, "gpt-4-gizmo-*", FormatMatchingModelName("gpt-4-gizmo-abc"))
}

func TestRoutingMatchModelNameStripsThenWildcards(t *testing.T) {
	assert.Equal(t, "qwen3-max", RoutingMatchModelName("qwen3-max@thinking:on@temperature:0.2"))
	assert.Equal(t, "claude-3-7-sonnet", RoutingMatchModelName("claude-3-7-sonnet-thinking"))
	assert.Equal(t, "gemini-2.5-flash-thinking-*", RoutingMatchModelName("gemini-2.5-flash-thinking-8192"))
	assert.Equal(t, "gpt-5.1-codex-max", RoutingMatchModelName("gpt-5.1-codex-max"))

	geminiSettings := model_setting.GetGeminiSettings()
	old := geminiSettings.ThinkingAdapterEnabled
	geminiSettings.ThinkingAdapterEnabled = true
	t.Cleanup(func() { geminiSettings.ThinkingAdapterEnabled = old })
	assert.Equal(t, "gemini-2.5-flash", RoutingMatchModelName("gemini-2.5-flash-thinking-8192"))
}

func TestModelPricingCandidatesIncludeEffortSuffixBase(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "claude-fable-5-high",
			want: []string{"claude-fable-5-high", "claude-fable-5-*", "claude-fable-*", "claude-*", "claude-fable-5"},
		},
		{
			name: "claude-fable-5-1-max",
			want: []string{"claude-fable-5-1-max", "claude-fable-5-1-*", "claude-fable-5-*", "claude-fable-*", "claude-*", "claude-fable-5-1"},
		},
		{
			name: "gemini-3.8-flash-high",
			want: []string{"gemini-3.8-flash-high", "gemini-3.8-flash-*", "gemini-3.8-*", "gemini-*", "gemini-3.8-flash"},
		},
		{
			name: "deepseek-v4-pro-high",
			want: []string{"deepseek-v4-pro-high", "deepseek-v4-pro-*", "deepseek-v4-*", "deepseek-*", "deepseek-v4-pro"},
		},
		{
			name: "deepseek-flash-max",
			want: []string{"deepseek-flash-max", "deepseek-flash-*", "deepseek-*", "deepseek-flash"},
		},
		{
			name: "kimi-k3-high",
			want: []string{"kimi-k3-high", "kimi-k3-*", "kimi-*", "kimi-k3"},
		},
		{
			name: "grok-4.7-xhigh",
			want: []string{"grok-4.7-xhigh", "grok-4.7-*", "grok-*", "grok-4.7"},
		},
		// No effort suffix to strip: the candidate list is unchanged.
		{
			name: "kimi-k3-medium",
			want: []string{"kimi-k3-medium", "kimi-k3-*", "kimi-*"},
		},
		{
			name: "qwen3-max",
			want: []string{"qwen3-max", "qwen3-*"},
		},
		{
			name: "glm-5.2-max-extra",
			want: []string{"glm-5.2-max-extra", "glm-5.2-max-*", "glm-5.2-*", "glm-*"},
		},
		{
			name: "kimi-k2-thinking",
			want: []string{"kimi-k2-thinking", "kimi-k2-*", "kimi-*"},
		},
		// An @ modifier name is priced by the canonical ladder. Leaking its base
		// in here would make the base row answer for the canonical candidate.
		{
			name: "gemini-2.5-flash@thinking:on",
			want: []string{"gemini-2.5-flash@thinking:on", "gemini-2.5-*", "gemini-*"},
		},
		{
			name: "claude-fable-5@effort:high@thinking:on",
			want: []string{"claude-fable-5@effort:high@thinking:on", "claude-fable-*", "claude-*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ModelPricingCandidates(tt.name))
		})
	}
}

func TestRoutingMatchModelNamePreservesExemptAtName(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	original := append([]string(nil), settings.ThinkingModelBlacklist...)
	t.Cleanup(func() { settings.ThinkingModelBlacklist = original })
	settings.ThinkingModelBlacklist = append(original, "re:.*@sha256:.*")

	assert.Equal(t, "opaque@sha256:deadbeef", RoutingMatchModelName("opaque@sha256:deadbeef"))
	assert.Equal(t, "kimi-k2-thinking", RoutingMatchModelName("kimi-k2-thinking"))
}
