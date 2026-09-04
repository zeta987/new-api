package ratio_setting

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAstraPricingUsesBaseForReasoningAliases(t *testing.T) {
	InitRatioSettings()
	for _, model := range []string{"gpt-6-astra", "gpt-6-astra-pro-max", "gpt-6-astra-standard-high"} {
		ratio, found, _ := GetModelRatio(model)
		require.True(t, found)
		assert.Equal(t, 5.0, ratio)
		assert.Equal(t, 5.0, GetCompletionRatio(model))
		cache, found := GetCacheRatio(model)
		require.True(t, found)
		assert.Equal(t, 0.1, cache)
		create, found := GetCreateCacheRatio(model)
		require.True(t, found)
		assert.Equal(t, 1.25, create)
	}
}
