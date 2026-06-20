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

func TestDefaultCacheRatiosIncludeOpus48SeriesMatchingOpus47(t *testing.T) {
	for _, suffix := range []string{"", "-thinking", "-low", "-medium", "-high", "-xhigh", "-max"} {
		opus47 := "claude-opus-4-7" + suffix
		opus48 := "claude-opus-4-8" + suffix

		require.Equal(t, defaultCacheRatio[opus47], defaultCacheRatio[opus48])
		require.Equal(t, defaultCreateCacheRatio[opus47], defaultCreateCacheRatio[opus48])
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
