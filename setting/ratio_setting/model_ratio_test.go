package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelRatioIncludesOpus48SeriesMatchingOpus47(t *testing.T) {
	defaultRatios := GetDefaultModelRatioMap()

	for _, suffix := range []string{"", "-low", "-medium", "-high", "-xhigh", "-max"} {
		opus47 := "claude-opus-4-7" + suffix
		opus48 := "claude-opus-4-8" + suffix

		require.Equal(t, defaultRatios[opus47], defaultRatios[opus48])
	}
}

func TestDefaultModelRatioIncludesFable5SeriesMatchingOpus48(t *testing.T) {
	defaultRatios := GetDefaultModelRatioMap()

	for _, suffix := range []string{"", "-low", "-medium", "-high", "-xhigh", "-max"} {
		opus48 := "claude-opus-4-8" + suffix
		fable5 := "claude-fable-5" + suffix

		require.Equal(t, defaultRatios[opus48], defaultRatios[fable5])
	}
}

func TestDefaultModelRatioIncludesSonnet5SeriesMatchingSonnet4(t *testing.T) {
	defaultRatios := GetDefaultModelRatioMap()

	for _, suffix := range []string{"", "-thinking", "-low", "-medium", "-high", "-xhigh", "-max"} {
		require.Equal(t, defaultRatios["claude-sonnet-4-20250514"], defaultRatios["claude-sonnet-5"+suffix])
	}
}

func TestSonnet5RatiosMatchThreeDollarInputFifteenDollarOutput(t *testing.T) {
	InitRatioSettings()

	for _, model := range []string{
		"claude-sonnet-5",
		"claude-sonnet-5-low",
		"claude-sonnet-5-medium",
		"claude-sonnet-5-high",
		"claude-sonnet-5-xhigh",
		"claude-sonnet-5-max",
		"claude-sonnet-5-20260630",
		"claude-sonnet-5-20260630-xhigh",
	} {
		modelRatio, ok, _ := GetModelRatio(model)
		require.True(t, ok)
		require.Equal(t, 1.5, modelRatio)
		require.Equal(t, 5.0, GetCompletionRatio(model))
	}
}

func TestDefaultCacheRatiosIncludeOpus48SeriesMatchingOpus47(t *testing.T) {
	for _, suffix := range []string{"", "-thinking", "-low", "-medium", "-high", "-xhigh", "-max"} {
		opus47 := "claude-opus-4-7" + suffix
		opus48 := "claude-opus-4-8" + suffix

		require.Equal(t, defaultCacheRatio[opus47], defaultCacheRatio[opus48])
		require.Equal(t, defaultCreateCacheRatio[opus47], defaultCreateCacheRatio[opus48])
	}
}

func TestDefaultCacheRatiosIncludeSonnet5SeriesMatchingSonnet4(t *testing.T) {
	for _, suffix := range []string{"", "-thinking", "-low", "-medium", "-high", "-xhigh", "-max"} {
		require.Equal(t, defaultCacheRatio["claude-sonnet-4-20250514"], defaultCacheRatio["claude-sonnet-5"+suffix])
		require.Equal(t, defaultCreateCacheRatio["claude-sonnet-4-20250514"], defaultCreateCacheRatio["claude-sonnet-5"+suffix])
	}
}

func TestDefaultCacheRatiosIncludeFable5SeriesMatchingOpus48(t *testing.T) {
	for _, suffix := range []string{"", "-low", "-medium", "-high", "-xhigh", "-max"} {
		opus48 := "claude-opus-4-8" + suffix
		fable5 := "claude-fable-5" + suffix

		require.Equal(t, defaultCacheRatio[opus48], defaultCacheRatio[fable5])
		require.Equal(t, defaultCreateCacheRatio[opus48], defaultCreateCacheRatio[fable5])
	}
}

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

func TestGPT56PricingWildcardMatchesAllVariants(t *testing.T) {
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

func TestModelPricingCandidatesUseSpecificityOrder(t *testing.T) {
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
}

func TestPricingWildcardLiteralUsesExactLookupOnly(t *testing.T) {
	originalPriceJSON := ModelPrice2JSONString()
	originalRatioJSON := ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
		require.NoError(t, UpdateModelRatioByJSONString(originalRatioJSON))
	})

	require.NoError(t, UpdateModelPriceByJSONString(`{"gpt-5.6-*":0.125}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{"gpt-4-gizmo-*":4.25,"gpt-5.6-*":3.25}`))

	price, hasPrice := GetModelPrice("gpt-5.6-*", false)
	require.True(t, hasPrice)
	require.Equal(t, 0.125, price)

	ratio, hasRatio, _ := GetModelRatio("gpt-4-gizmo-*")
	require.True(t, hasRatio)
	require.Equal(t, 4.25, ratio)
}

func TestCompactPricingPrecedesPrefixWildcard(t *testing.T) {
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
		"gpt-5.6-*":1.25
	}`))

	model := "gpt-5.6-luna-openai-compact"
	price, hasPrice := GetModelPrice(model, false)
	require.True(t, hasPrice)
	require.Equal(t, 0.75, price)

	ratio, hasRatio, _ := GetModelRatio(model)
	require.True(t, hasRatio)
	require.Equal(t, 7.5, ratio)
}

func TestGPT56PricingUsesMostSpecificEntry(t *testing.T) {
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

	tests := []struct {
		model     string
		wantPrice float64
		wantRatio float64
	}{
		{model: "gpt-5.6-luna-pro-max", wantPrice: 0.75, wantRatio: 7.5},
		{model: "gpt-5.6-luna-pro-high", wantPrice: 0.375, wantRatio: 3.75},
		{model: "gpt-5.6-luna-standard-high", wantPrice: 0.25, wantRatio: 2.5},
		{model: "gpt-5.6-terra-high", wantPrice: 0.125, wantRatio: 1.25},
		{model: "gpt-5.6-luna", wantPrice: 0.5, wantRatio: 5},
	}
	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			price, hasPrice := GetModelPrice(test.model, false)
			require.True(t, hasPrice)
			require.Equal(t, test.wantPrice, price)

			ratio, hasRatio, matchedModel := GetModelRatio(test.model)
			require.True(t, hasRatio)
			require.Equal(t, test.wantRatio, ratio)
			require.Equal(t, FormatMatchingModelName(test.model), matchedModel)
		})
	}
}

func TestGPT56PricingWildcardAppliesToRelatedRatios(t *testing.T) {
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
	cacheRatio, hasCacheRatio := GetCacheRatio(model)
	require.True(t, hasCacheRatio)
	require.Equal(t, 0.2, cacheRatio)
	createCacheRatio, hasCreateCacheRatio := GetCreateCacheRatio(model)
	require.True(t, hasCreateCacheRatio)
	require.Equal(t, 1.3, createCacheRatio)
	imageRatio, hasImageRatio := GetImageRatio(model)
	require.True(t, hasImageRatio)
	require.Equal(t, 2.2, imageRatio)
	require.Equal(t, 3.3, GetAudioRatio(model))
	require.True(t, ContainsAudioRatio(model))
	require.Equal(t, 4.4, GetAudioCompletionRatio(model))
	require.True(t, ContainsAudioCompletionRatio(model))
}

func TestGPT56ZeroWildcardPriceIsConfigured(t *testing.T) {
	originalPriceJSON := ModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPriceJSON))
	})
	require.NoError(t, UpdateModelPriceByJSONString(`{"gpt-5.6-*":0}`))

	price, ok := GetModelPrice("gpt-5.6-luna-pro-max", false)
	require.True(t, ok)
	require.Zero(t, price)
}
