package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatMatchingModelNameNormalizesGenericGLMAliases(t *testing.T) {
	for _, model := range []string{"glm-5.3-flash-low", "glm-5.3-flash-high", "glm-5.3-flash-max"} {
		assert.Equal(t, "glm-5.3-flash", FormatMatchingModelName(model))
	}
	for _, model := range []string{"glm-5.3-flash", "glm-5.3-flash-fast", "glm-5.3-flash-max-extra"} {
		assert.Equal(t, model, FormatMatchingModelName(model))
	}
}

func TestGenericGLMAliasPricingPrefersExactThenBaseFallback(t *testing.T) {
	originalPriceJSON := ModelPrice2JSONString()
	originalRatioJSON := ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
		require.NoError(t, UpdateModelRatioByJSONString(originalRatioJSON))
	})

	require.NoError(t, UpdateModelPriceByJSONString(`{"glm-5.3-flash":0.3,"glm-5.3-flash-max":0.9}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{"glm-5.3-flash":3,"glm-5.3-flash-max":9}`))

	price, ok := GetModelPrice("glm-5.3-flash-max", false)
	require.True(t, ok)
	assert.Equal(t, 0.9, price)
	ratio, ok, normalized := GetModelRatio("glm-5.3-flash-max")
	require.True(t, ok)
	assert.Equal(t, 9.0, ratio)
	assert.Equal(t, "glm-5.3-flash", normalized)

	price, ok = GetModelPrice("glm-5.3-flash-low", false)
	require.True(t, ok)
	assert.Equal(t, 0.3, price)
	ratio, ok, normalized = GetModelRatio("glm-5.3-flash-low")
	require.True(t, ok)
	assert.Equal(t, 3.0, ratio)
	assert.Equal(t, "glm-5.3-flash", normalized)
}
