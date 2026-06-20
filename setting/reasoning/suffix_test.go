package reasoning

import (
	"testing"

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
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			require.Equal(t, tt.want, IsClaudePost46AdaptiveThinkingModel(tt.model))
		})
	}
}
