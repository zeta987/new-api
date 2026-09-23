package reasoning

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffortSuffixModelNames(t *testing.T) {
	tests := []struct {
		base string
		want []string
	}{
		{
			base: "claude-fable-5",
			want: []string{"claude-fable-5", "claude-fable-5-low", "claude-fable-5-medium", "claude-fable-5-high", "claude-fable-5-xhigh", "claude-fable-5-max"},
		},
		{
			base: "claude-fable-5-1",
			want: []string{"claude-fable-5-1", "claude-fable-5-1-low", "claude-fable-5-1-medium", "claude-fable-5-1-high", "claude-fable-5-1-xhigh", "claude-fable-5-1-max"},
		},
		{
			base: "claude-opus-5",
			want: []string{"claude-opus-5", "claude-opus-5-low", "claude-opus-5-medium", "claude-opus-5-high", "claude-opus-5-xhigh", "claude-opus-5-max"},
		},
		{
			base: "claude-opus-5-5",
			want: []string{"claude-opus-5-5", "claude-opus-5-5-low", "claude-opus-5-5-medium", "claude-opus-5-5-high", "claude-opus-5-5-xhigh", "claude-opus-5-5-max"},
		},
		{
			base: "claude-sonnet-5-5",
			want: []string{"claude-sonnet-5-5", "claude-sonnet-5-5-low", "claude-sonnet-5-5-medium", "claude-sonnet-5-5-high", "claude-sonnet-5-5-xhigh", "claude-sonnet-5-5-max"},
		},
		{
			base: "claude-sonnet-5",
			want: []string{"claude-sonnet-5", "claude-sonnet-5-low", "claude-sonnet-5-medium", "claude-sonnet-5-high", "claude-sonnet-5-xhigh", "claude-sonnet-5-max"},
		},
		{
			base: "gemini-3.1-flash-lite",
			want: []string{"gemini-3.1-flash-lite", "gemini-3.1-flash-lite-minimal", "gemini-3.1-flash-lite-low", "gemini-3.1-flash-lite-medium", "gemini-3.1-flash-lite-high"},
		},
		{
			base: "gemini-3.5-flash-lite",
			want: []string{"gemini-3.5-flash-lite", "gemini-3.5-flash-lite-minimal", "gemini-3.5-flash-lite-low", "gemini-3.5-flash-lite-medium", "gemini-3.5-flash-lite-high"},
		},
		{
			base: "gemini-3.5-flash",
			want: []string{"gemini-3.5-flash", "gemini-3.5-flash-minimal", "gemini-3.5-flash-low", "gemini-3.5-flash-medium", "gemini-3.5-flash-high"},
		},
		{
			base: "gemini-3.6-flash",
			want: []string{"gemini-3.6-flash", "gemini-3.6-flash-minimal", "gemini-3.6-flash-low", "gemini-3.6-flash-medium", "gemini-3.6-flash-high"},
		},
		{
			base: "gemini-3.7-flash",
			want: []string{"gemini-3.7-flash", "gemini-3.7-flash-low", "gemini-3.7-flash-medium", "gemini-3.7-flash-high"},
		},
		{
			base: "gemini-3.8-flash",
			want: []string{"gemini-3.8-flash", "gemini-3.8-flash-low", "gemini-3.8-flash-medium", "gemini-3.8-flash-high"},
		},
		{
			base: "gemini-3.1-pro-preview",
			want: []string{"gemini-3.1-pro-preview", "gemini-3.1-pro-preview-low", "gemini-3.1-pro-preview-medium", "gemini-3.1-pro-preview-high"},
		},
		{
			base: "glm-5.3",
			want: []string{"glm-5.3", "glm-5.3-low", "glm-5.3-high", "glm-5.3-max"},
		},
		{
			base: "glm-5.3-flash",
			want: []string{"glm-5.3-flash", "glm-5.3-flash-low", "glm-5.3-flash-high", "glm-5.3-flash-max"},
		},
		{
			base: "glm-5.3-flashx",
			want: []string{"glm-5.3-flashx", "glm-5.3-flashx-low", "glm-5.3-flashx-high", "glm-5.3-flashx-max"},
		},
		{
			base: "deepseek-flash",
			want: []string{"deepseek-flash", "deepseek-flash-none", "deepseek-flash-low", "deepseek-flash-high", "deepseek-flash-max"},
		},
		{
			base: "deepseek-v4-pro",
			want: []string{"deepseek-v4-pro", "deepseek-v4-pro-none", "deepseek-v4-pro-low", "deepseek-v4-pro-high", "deepseek-v4-pro-max"},
		},
		{
			base: "kimi-k3",
			want: []string{"kimi-k3", "kimi-k3-none", "kimi-k3-low", "kimi-k3-high", "kimi-k3-max"},
		},
		{
			base: "grok-4.0",
			want: []string{"grok-4.0", "grok-4.0-low", "grok-4.0-medium", "grok-4.0-high"},
		},
		{
			base: "grok-4.5",
			want: []string{"grok-4.5", "grok-4.5-low", "grok-4.5-medium", "grok-4.5-high"},
		},
		{
			base: "grok-4.6",
			want: []string{"grok-4.6", "grok-4.6-low", "grok-4.6-medium", "grok-4.6-high", "grok-4.6-xhigh"},
		},
		{
			base: "grok-4.7",
			want: []string{"grok-4.7", "grok-4.7-low", "grok-4.7-medium", "grok-4.7-high", "grok-4.7-xhigh"},
		},
		{
			base: "grok-4.9",
			want: []string{"grok-4.9", "grok-4.9-low", "grok-4.9-medium", "grok-4.9-high", "grok-4.9-xhigh"},
		},
		// Namespace prefixes are carried onto every variant.
		{
			base: "vendor/kimi-k3",
			want: []string{"vendor/kimi-k3", "vendor/kimi-k3-none", "vendor/kimi-k3-low", "vendor/kimi-k3-high", "vendor/kimi-k3-max"},
		},
		{
			base: "x-ai/grok-4.7",
			want: []string{"x-ai/grok-4.7", "x-ai/grok-4.7-low", "x-ai/grok-4.7-medium", "x-ai/grok-4.7-high", "x-ai/grok-4.7-xhigh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.base, func(t *testing.T) {
			assert.Equal(t, tt.want, EffortSuffixModelNames(tt.base))
		})
	}
}

func TestEffortSuffixModelNamesRejectsUnknownBases(t *testing.T) {
	for _, name := range []string{
		"",
		// GLM 5.2 is deliberately not published.
		"glm-5.2",
		"glm-5.2-max-extra",
		"kimi-k2-thinking",
		"kimi-k2.5",
		"deepseek-chat",
		"grok-4.7-preview",
		"grok-4.7-latest",
		"grok-4.7-20260922",
		"grok-4.7-search",
		"grok-4.7-reasoning",
		"grok-4.7-multi-agent",
		"claude-fable-5-high",
		"claude-sonnet-5-5-preview",
		"claude-sonnet-5-5-20260923",
		"claude-sonnet-5-20260630",
		"vendor/claude-sonnet-5-20260630",
		"gemini-2.5-flash",
		"qwen3.8-max",
		"claude-fable-5@thinking:on",
	} {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, EffortSuffixModelNames(name))
		})
	}
}

func TestEffortSuffixVocabularyOmissions(t *testing.T) {
	// The capability tables would admit these, the published vocabulary
	// does not. See the comment on effortSuffixVocabulary.
	for _, base := range []string{"gemini-3.7-flash", "gemini-3.8-flash"} {
		assert.NotContains(t, EffortSuffixModelNames(base), base+"-minimal")
	}
	for _, base := range []string{"claude-opus-5", "claude-opus-5-5", "claude-sonnet-5", "claude-sonnet-5-5", "claude-fable-5", "claude-fable-5-1"} {
		assert.NotContains(t, EffortSuffixModelNames(base), base+"-none")
	}
}

// Sanity check in one direction only: every level the vocabulary publishes must
// be renderable. The reverse does not hold, and must not be asserted.
func TestEffortSuffixVocabularyIsRenderable(t *testing.T) {
	for base, levels := range effortSuffixVocabulary {
		for _, level := range levels {
			t.Run(base+"-"+level, func(t *testing.T) {
				effort, err := ParseEffort(level)
				require.NoError(t, err)

				switch {
				case strings.HasPrefix(base, "claude-"):
					capabilities := claudeCapabilitiesFor(base)
					assert.True(t, capabilities.supportsEffort, "effort unsupported")
					if effort == EffortXHigh {
						assert.True(t, capabilities.supportsXHigh, "xhigh unsupported")
					}
					if effort == EffortMax {
						assert.True(t, capabilities.supportsMax, "max unsupported")
					}
				case strings.HasPrefix(base, "gemini-"):
					_, err := geminiLevelForEffort(base, effort)
					assert.NoError(t, err)
				}
			})
		}
	}
}

func TestClaudeOpus55DefaultAndDisabledThinking(t *testing.T) {
	defaultIntent := ResolveClaudeDefault("claude-opus-5-5", Intent{})
	assert.Equal(t, ModeAdaptive, defaultIntent.Mode)
	assert.Equal(t, EffortMedium, defaultIntent.Effort)
	assert.Equal(t, EffortHigh, ResolveClaudeDefault("claude-opus-5", Intent{}).Effort)

	rendered, err := RenderClaude("claude-opus-5-5", Intent{Mode: ModeDisabled}, nil, 0.8)
	require.NoError(t, err)
	require.NotNil(t, rendered.Thinking)
	assert.Equal(t, "adaptive", rendered.Thinking.Type)
	assert.Equal(t, EffortLow, rendered.OutputEffort)
	assert.Equal(t, EffortLow, rendered.EffectiveEffort)
	assert.False(t, claudeCapabilitiesFor("claude-opus-5-5").supportsManual)
}

func TestParseGrokReasoningEffortSuffix(t *testing.T) {
	tests := []struct {
		model  string
		base   string
		effort string
		ok     bool
	}{
		{model: "grok-4.6-xhigh", base: "grok-4.6", effort: "xhigh", ok: true},
		{model: "grok-4.6-low", base: "grok-4.6", effort: "low", ok: true},
		{model: "grok-4.0-low", base: "grok-4.0", effort: "low", ok: true},
		{model: "grok-4.4-high", base: "grok-4.4", effort: "high", ok: true},
		{model: "grok-4.5-high", base: "grok-4.5", effort: "high", ok: true},
		{model: "grok-4.7-xhigh", base: "grok-4.7", effort: "xhigh", ok: true},
		{model: "x-ai/grok-4.7-xhigh", base: "x-ai/grok-4.7", effort: "xhigh", ok: true},
		{model: "grok-5.0-xhigh", base: "grok-5.0", effort: "xhigh", ok: true},
		// 4.5 and earlier do not accept xhigh.
		{model: "grok-4.5-xhigh", base: "grok-4.5-xhigh"},
		{model: "grok-4.6", base: "grok-4.6"},
		{model: "grok-4.6-ultra", base: "grok-4.6-ultra"},
		{model: "grok-4.6-20260801-xhigh", base: "grok-4.6-20260801-xhigh"},
		{model: "grok-4.7-preview-high", base: "grok-4.7-preview-high"},
		{model: "grok-4.7-latest-xhigh", base: "grok-4.7-latest-xhigh"},
		{model: "grok-4.7-search-xhigh", base: "grok-4.7-search-xhigh"},
		{model: "grok-4.7-reasoning-high", base: "grok-4.7-reasoning-high"},
		{model: "grok-4.7-multi-agent-high", base: "grok-4.7-multi-agent-high"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			base, effort, ok := ParseGrokReasoningEffortSuffix(tt.model)
			assert.Equal(t, tt.base, base)
			assert.Equal(t, tt.effort, effort)
			assert.Equal(t, tt.ok, ok)
		})
	}

	assert.True(t, IsStandardGrokModel("grok-4.6"))
	assert.True(t, IsStandardGrokModel("grok-4.7"))
	assert.True(t, IsStandardGrokModel("x-ai/grok-4.7"))
	assert.False(t, IsStandardGrokModel("grok-4.6-xhigh"))
	assert.False(t, IsStandardGrokModel("grok-4.7-preview"))
	assert.False(t, IsStandardGrokModel("grok-3-mini"))
}
