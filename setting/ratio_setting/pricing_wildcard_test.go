package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPricingWildcardMatchesGPT56Variants(t *testing.T) {
	originalPriceJSON := ModelPrice2JSONString()
	originalRatioJSON := ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
		require.NoError(t, UpdateModelRatioByJSONString(originalRatioJSON))
	})

	require.NoError(t, UpdateModelPriceByJSONString(`{"gpt-5.6-*":0.125}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{"gpt-5.6-*":3.25}`))
	for _, model := range []string{
		"gpt-5.6-luna",
		"gpt-5.6-luna-pro",
		"gpt-5.6-luna-pro-max",
		"gpt-5.6-terra-standard-high",
		"gpt-5.6-sol-max",
	} {
		t.Run(model, func(t *testing.T) {
			price, hasPrice := GetModelPrice(model, false)
			require.True(t, hasPrice)
			require.Equal(t, 0.125, price)

			ratio, hasRatio, _ := GetModelRatio(model)
			require.True(t, hasRatio)
			require.Equal(t, 3.25, ratio)
		})
	}
	_, hasPrice := GetModelPrice("gpt-5.60-luna", false)
	require.False(t, hasPrice)
}

func TestPricingWildcardUsesSpecificityOrder(t *testing.T) {
	require.Equal(t, []string{
		"gpt-5.6-luna-pro-max",
		"gpt-5.6-luna-pro-*",
		"gpt-5.6-luna-*",
		"gpt-5.6-*",
		"gpt-*",
		"gpt-5.6-luna",
	}, ModelPricingCandidates("gpt-5.6-luna-pro-max"))
	require.Equal(t, []string{"gpt-5.6-*"}, ModelPricingCandidates("gpt-5.6-*"))
	require.Empty(t, ModelPricingCandidates("gpt-5.*-luna"))

	originalPriceJSON := ModelPrice2JSONString()
	originalRatioJSON := ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
		require.NoError(t, UpdateModelRatioByJSONString(originalRatioJSON))
	})
	require.NoError(t, UpdateModelPriceByJSONString(`{
		"gpt-5.6-*":0.125,
		"gpt-5.6-luna-*":0.25,
		"gpt-5.6-luna-pro-*":0.375,
		"gpt-5.6-luna-pro-max":0.75,
		"gpt-5.6-luna":0.5
	}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{
		"gpt-5.6-*":1.25,
		"gpt-5.6-luna-*":2.5,
		"gpt-5.6-luna-pro-*":3.75,
		"gpt-5.6-luna-pro-max":7.5,
		"gpt-5.6-luna":5
	}`))

	for _, test := range []struct {
		model     string
		wantPrice float64
		wantRatio float64
	}{
		{model: "gpt-5.6-luna-pro-max", wantPrice: 0.75, wantRatio: 7.5},
		{model: "gpt-5.6-luna-pro-high", wantPrice: 0.375, wantRatio: 3.75},
		{model: "gpt-5.6-luna-standard-high", wantPrice: 0.25, wantRatio: 2.5},
		{model: "gpt-5.6-terra-high", wantPrice: 0.125, wantRatio: 1.25},
		{model: "gpt-5.6-luna", wantPrice: 0.5, wantRatio: 5},
	} {
		t.Run(test.model, func(t *testing.T) {
			price, hasPrice := GetModelPrice(test.model, false)
			require.True(t, hasPrice)
			require.Equal(t, test.wantPrice, price)
			ratio, hasRatio, _ := GetModelRatio(test.model)
			require.True(t, hasRatio)
			require.Equal(t, test.wantRatio, ratio)
		})
	}
}

func TestPricingWildcardPreservesLiteralAndCompactKeys(t *testing.T) {
	originalPriceJSON := ModelPrice2JSONString()
	originalRatioJSON := ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
		require.NoError(t, UpdateModelRatioByJSONString(originalRatioJSON))
	})
	require.NoError(t, UpdateModelPriceByJSONString(`{
		"*-openai-compact":0.75,
		"gpt-5.6-*":0.125
	}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{
		"*-openai-compact":7.5,
		"gpt-4-gizmo-*":4.25,
		"gpt-5.6-*":1.25
	}`))

	price, hasPrice := GetModelPrice("gpt-5.6-*", false)
	require.True(t, hasPrice)
	require.Equal(t, 0.125, price)
	ratio, hasRatio, _ := GetModelRatio("gpt-4-gizmo-*")
	require.True(t, hasRatio)
	require.Equal(t, 4.25, ratio)

	price, hasPrice = GetModelPrice("gpt-5.6-luna-openai-compact", false)
	require.True(t, hasPrice)
	require.Equal(t, 0.75, price)
	ratio, hasRatio, _ = GetModelRatio("gpt-5.6-luna-openai-compact")
	require.True(t, hasRatio)
	require.Equal(t, 7.5, ratio)
}

func TestPricingWildcardAppliesToRelatedRatios(t *testing.T) {
	originalCompletionJSON := CompletionRatio2JSONString()
	originalCacheJSON := CacheRatio2JSONString()
	originalCreateCacheJSON := CreateCacheRatio2JSONString()
	originalImageJSON := ImageRatio2JSONString()
	originalAudioJSON := AudioRatio2JSONString()
	originalAudioCompletionJSON := AudioCompletionRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateCompletionRatioByJSONString(originalCompletionJSON))
		require.NoError(t, UpdateCacheRatioByJSONString(originalCacheJSON))
		require.NoError(t, UpdateCreateCacheRatioByJSONString(originalCreateCacheJSON))
		require.NoError(t, UpdateImageRatioByJSONString(originalImageJSON))
		require.NoError(t, UpdateAudioRatioByJSONString(originalAudioJSON))
		require.NoError(t, UpdateAudioCompletionRatioByJSONString(originalAudioCompletionJSON))
	})

	require.NoError(t, UpdateCompletionRatioByJSONString(`{"gpt-5.6-*":6.5}`))
	require.NoError(t, UpdateCacheRatioByJSONString(`{"gpt-5.6-*":0.2}`))
	require.NoError(t, UpdateCreateCacheRatioByJSONString(`{"gpt-5.6-*":1.3}`))
	require.NoError(t, UpdateImageRatioByJSONString(`{"gpt-5.6-*":2.2}`))
	require.NoError(t, UpdateAudioRatioByJSONString(`{"gpt-5.6-*":3.3}`))
	require.NoError(t, UpdateAudioCompletionRatioByJSONString(`{"gpt-5.6-*":4.4}`))

	model := "gpt-5.6-luna-pro-max"
	require.Equal(t, 6.5, GetCompletionRatio(model))
	require.Equal(t, CompletionRatioInfo{Ratio: 6.5, Locked: false}, GetCompletionRatioInfo(model))
	cacheRatio, ok := GetCacheRatio(model)
	require.True(t, ok)
	require.Equal(t, 0.2, cacheRatio)
	createCacheRatio, ok := GetCreateCacheRatio(model)
	require.True(t, ok)
	require.Equal(t, 1.3, createCacheRatio)
	imageRatio, ok := GetImageRatio(model)
	require.True(t, ok)
	require.Equal(t, 2.2, imageRatio)
	require.Equal(t, 3.3, GetAudioRatio(model))
	require.True(t, ContainsAudioRatio(model))
	require.Equal(t, 4.4, GetAudioCompletionRatio(model))
	require.True(t, ContainsAudioCompletionRatio(model))
}

func TestPricingWildcardZeroPriceIsConfigured(t *testing.T) {
	originalPriceJSON := ModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
	})
	require.NoError(t, UpdateModelPriceByJSONString(`{"gpt-5.6-*":0}`))
	price, ok := GetModelPrice("gpt-5.6-luna-pro-max", false)
	require.True(t, ok)
	require.Zero(t, price)
}
