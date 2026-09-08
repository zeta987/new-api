package ratio_setting

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReasoningFamilyPricingInheritsWithoutWildcardEntry(t *testing.T) {
	prices, ratios, completion, cache := ModelPrice2JSONString(), ModelRatio2JSONString(), CompletionRatio2JSONString(), CacheRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(prices))
		require.NoError(t, UpdateModelRatioByJSONString(ratios))
		require.NoError(t, UpdateCompletionRatioByJSONString(completion))
		require.NoError(t, UpdateCacheRatioByJSONString(cache))
	})
	require.NoError(t, UpdateModelPriceByJSONString(`{}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{"gpt-5.6-sol":2,"gpt-6-astra":5}`))
	require.NoError(t, UpdateCompletionRatioByJSONString(`{"gpt-5.6-sol":5,"gpt-6-astra":5}`))
	require.NoError(t, UpdateCacheRatioByJSONString(`{"gpt-5.6-sol":0.1,"gpt-6-astra":0.1}`))
	for _, tc := range []struct {
		model string
		ratio float64
	}{{"gpt-5.6-sol-*", 2}, {"gpt-5.6-sol-pro-max", 2}, {"gpt-6-astra-*", 5}, {"gpt-6-astra-pro-max", 5}} {
		actual, found, _ := GetModelRatio(tc.model)
		assert.True(t, found, tc.model)
		assert.Equal(t, tc.ratio, actual)
		assert.Equal(t, 5.0, GetCompletionRatio(tc.model))
		actual, found = GetCacheRatio(tc.model)
		assert.True(t, found)
		assert.Equal(t, 0.1, actual)
	}
	require.NoError(t, UpdateModelRatioByJSONString(`{"gpt-6-astra":5,"gpt-6-astra-*":7,"gpt-6-astra-pro-max":9}`))
	for _, tc := range []struct {
		model string
		ratio float64
	}{{"gpt-6-astra", 5}, {"gpt-6-astra-*", 7}, {"gpt-6-astra-high", 7}, {"gpt-6-astra-pro-max", 9}} {
		actual, found, _ := GetModelRatio(tc.model)
		assert.True(t, found)
		assert.Equal(t, tc.ratio, actual)
	}
	require.NoError(t, UpdateModelPriceByJSONString(`{"gpt-6-astra":0}`))
	price, found := GetModelPrice("gpt-6-astra-*", false)
	assert.True(t, found)
	assert.Zero(t, price)
}
