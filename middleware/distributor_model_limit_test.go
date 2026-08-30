package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenAllowsGPT56ReasoningModelCandidates(t *testing.T) {
	tests := []struct {
		name         string
		modelLimits  map[string]bool
		requestModel string
		want         bool
	}{
		{
			name:         "exact model",
			modelLimits:  map[string]bool{"gpt-5.6-luna-pro-max": true},
			requestModel: "gpt-5.6-luna-pro-max",
			want:         true,
		},
		{
			name:         "reasoning wildcard",
			modelLimits:  map[string]bool{"gpt-5.6-luna-*": true},
			requestModel: "gpt-5.6-luna-pro-max",
			want:         true,
		},
		{
			name:         "normalized base",
			modelLimits:  map[string]bool{"gpt-5.6-luna": true},
			requestModel: "gpt-5.6-luna-pro-max",
			want:         true,
		},
		{
			name:         "invalid suffix cannot use wildcard",
			modelLimits:  map[string]bool{"gpt-5.6-luna-*": true},
			requestModel: "gpt-5.6-luna-pro-ultra",
			want:         false,
		},
		{
			name:         "literal wildcard is configuration only",
			modelLimits:  map[string]bool{"gpt-5.6-luna-*": true},
			requestModel: "gpt-5.6-luna-*",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tokenAllowsModel(tt.modelLimits, tt.requestModel))
		})
	}
}
