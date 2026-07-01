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
