package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacyPricingOptionUpdateInvalidatesPricingCache(t *testing.T) {
	originalPriceJSON := ratio_setting.ModelPrice2JSONString()
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(originalPriceJSON))
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
		InvalidatePricingCache()
	})

	updatePricingLock.Lock()
	pricingMap = []Pricing{{ModelName: "cached-model"}}
	lastGetPricingTime = time.Now()
	updatePricingLock.Unlock()

	require.NoError(t, updateOptionMap("ModelPrice", `{"gpt-5.6-*":0.125}`))
	updatePricingLock.RLock()
	defer updatePricingLock.RUnlock()
	assert.Nil(t, pricingMap)
	assert.True(t, lastGetPricingTime.IsZero())
}
