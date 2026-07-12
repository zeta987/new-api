package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGPT56ReasoningSuffixesUseBaseModelRatio(t *testing.T) {
	InitRatioSettings()

	tests := map[string]float64{
		"gpt-5.6-luna-max":           0.5,
		"gpt-5.6-luna-pro-max":       0.5,
		"gpt-5.6-terra-standard-low": 1.25,
		"gpt-5.6-sol-stanard-xhigh":  2.5,
	}
	for model, wantRatio := range tests {
		t.Run(model, func(t *testing.T) {
			ratio, ok, matchedModel := GetModelRatio(model)

			require.True(t, ok)
			require.Equal(t, wantRatio, ratio)
			require.Equal(t, FormatMatchingModelName(model), matchedModel)
		})
	}
}
