package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatMatchingModelNameNormalizesExactGLM53Aliases(t *testing.T) {
	for _, model := range []string{"glm-5.3-low", "glm-5.3-high", "glm-5.3-max"} {
		assert.Equal(t, "glm-5.3", FormatMatchingModelName(model))
	}
	for _, model := range []string{"glm-5.3", "glm-5.3-none", "glm-5.3-max-extra", "glm-5.1-high"} {
		assert.Equal(t, model, FormatMatchingModelName(model))
	}
}

func TestGLM53AliasPricingPrefersExactThenBaseFallback(t *testing.T) {
	originalPriceJSON := ModelPrice2JSONString()
	originalRatioJSON := ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
		require.NoError(t, UpdateModelRatioByJSONString(originalRatioJSON))
	})

	require.NoError(t, UpdateModelPriceByJSONString(`{"glm-5.3":0.3,"glm-5.3-max":0.9}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{"glm-5.3":3,"glm-5.3-max":9}`))

	price, ok := GetModelPrice("glm-5.3-max", false)
	require.True(t, ok)
	assert.Equal(t, 0.9, price)
	ratio, ok, normalized := GetModelRatio("glm-5.3-max")
	require.True(t, ok)
	assert.Equal(t, 9.0, ratio)
	assert.Equal(t, "glm-5.3", normalized)

	price, ok = GetModelPrice("glm-5.3-low", false)
	require.True(t, ok)
	assert.Equal(t, 0.3, price)
	ratio, ok, normalized = GetModelRatio("glm-5.3-low")
	require.True(t, ok)
	assert.Equal(t, 3.0, ratio)
	assert.Equal(t, "glm-5.3", normalized)
}
