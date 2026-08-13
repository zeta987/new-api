package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatMatchingModelNameNormalizesExactGLM52Aliases(t *testing.T) {
	for _, model := range []string{"glm-5.2-none", "glm-5.2-high", "glm-5.2-max"} {
		assert.Equal(t, "glm-5.2", FormatMatchingModelName(model))
	}
	for _, model := range []string{"glm-5.2", "glm-5.2-low", "glm-5.2-max-extra", "glm-5.1-high"} {
		assert.Equal(t, model, FormatMatchingModelName(model))
	}
}

func TestGLM52AliasPricingPrefersExactThenBaseFallback(t *testing.T) {
	originalPriceJSON := ModelPrice2JSONString()
	originalRatioJSON := ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
		require.NoError(t, UpdateModelRatioByJSONString(originalRatioJSON))
	})

	require.NoError(t, UpdateModelPriceByJSONString(`{"glm-5.2":0.2,"glm-5.2-high":0.8}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{"glm-5.2":2,"glm-5.2-high":8}`))

	price, ok := GetModelPrice("glm-5.2-high", false)
	require.True(t, ok)
	assert.Equal(t, 0.8, price)
	ratio, ok, normalized := GetModelRatio("glm-5.2-high")
	require.True(t, ok)
	assert.Equal(t, 8.0, ratio)
	assert.Equal(t, "glm-5.2", normalized)

	price, ok = GetModelPrice("glm-5.2-max", false)
	require.True(t, ok)
	assert.Equal(t, 0.2, price)
	ratio, ok, normalized = GetModelRatio("glm-5.2-max")
	require.True(t, ok)
	assert.Equal(t, 2.0, ratio)
	assert.Equal(t, "glm-5.2", normalized)
}
