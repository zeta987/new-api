package model

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func pricingUsagePluginSource(version, usageSchema string) string {
	return pricingUsagePluginSourceFor("pricing-usage-probe", "pricing-usage-model", version, usageSchema)
}

func pricingUsagePluginSourceFor(key, model, version, usageSchema string) string {
	return fmt.Sprintf(`
export const meta = {
  apiVersion: 1, key: %q, name: "Pricing Usage Probe", version: %q, author: {name: "Test"},
  models: [%q], fetchMode: "per_task", usageSchema: %s
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {}; }
`, key, version, model, usageSchema)
}

func TestPricingCarriesTaskUsageSchemaAndRefreshesWithPluginGeneration(t *testing.T) {
	resetPricingEndpointTestTables(t)
	const pluginKey = "pricing-usage-probe"
	initialSource := pricingUsagePluginSource("1.0.0", `{
  seconds: {type: "number", unit: "second", description: "Estimated duration."}
}`)
	_, err := jsplugin.DefaultRegistry.Register(initialSource, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(pluginKey) })

	insertPricingEndpointChannel(t, 901, constant.ChannelTypeTaskPlugin, dto.ChannelOtherSettings{})
	insertPricingEndpointAbility(t, 901, "pricing-usage-model")
	insertPricingEndpointAbility(t, 901, "ordinary-model")

	initialPricing := pricingByModel(GetPricing())
	require.Contains(t, initialPricing, "pricing-usage-model")
	require.Contains(t, initialPricing, "ordinary-model")
	assert.Equal(t, "second", initialPricing["pricing-usage-model"].BillingUsageSchema["seconds"].Unit)
	assert.Equal(t, "Estimated duration.", initialPricing["pricing-usage-model"].BillingUsageSchema["seconds"].Description["en"])
	assert.Nil(t, initialPricing["ordinary-model"].BillingUsageSchema)

	updatedSource := pricingUsagePluginSource("1.1.0", `{
  seconds: {type: "number", unit: "second", description: "Measured duration."},
  clips: {type: "number", unit: "count", description: "Generated clip count."}
}`)
	_, err = jsplugin.DefaultRegistry.Register(updatedSource, jsplugin.Options{})
	require.NoError(t, err)
	lastGetPricingTime = time.Now().Add(-2 * time.Minute)

	refreshedPricing := pricingByModel(GetPricing())
	require.Len(t, refreshedPricing["pricing-usage-model"].BillingUsageSchema, 2)
	assert.Equal(t, "Measured duration.", refreshedPricing["pricing-usage-model"].BillingUsageSchema["seconds"].Description["en"])
	assert.Equal(t, "count", refreshedPricing["pricing-usage-model"].BillingUsageSchema["clips"].Unit)
}

func TestPricingAliasCarriesPluginUsageSchemaAndTailExpr(t *testing.T) {
	resetPricingEndpointTestTables(t)
	const pluginKey = "pricing-usage-probe"
	source := pricingUsagePluginSource("1.0.0", `{
  seconds: {type: "number", unit: "second", description: "Estimated duration."}
}`)
	_, err := jsplugin.DefaultRegistry.Register(source, jsplugin.Options{})
	require.NoError(t, err)
	t.Cleanup(func() { jsplugin.DefaultRegistry.Unregister(pluginKey) })

	mapping := `{"alias-model":"pricing-usage-model"}`
	channel := &Channel{
		Id:           910,
		Type:         constant.ChannelTypeTaskPlugin,
		Key:          "key-910",
		Status:       1,
		Name:         "channel-910",
		Models:       "alias-model,pricing-usage-model",
		ModelMapping: &mapping,
	}
	require.NoError(t, DB.Create(channel).Error)
	insertPricingEndpointAbility(t, 910, "alias-model")
	insertPricingEndpointAbility(t, 910, "pricing-usage-model")
	InitChannelCache()

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"pricing-usage-model":"tiered_expr","alias-own-expr":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"pricing-usage-model":"u(\"seconds\")","alias-own-expr":"u(\"seconds\") * 2"}`,
	}))
	InvalidatePricingCache()

	pricing := pricingByModel(GetPricing())
	require.Contains(t, pricing, "alias-model")
	require.Contains(t, pricing, "pricing-usage-model")
	assert.Equal(t, "second", pricing["alias-model"].BillingUsageSchema["seconds"].Unit)
	assert.Equal(t, "Estimated duration.", pricing["alias-model"].BillingUsageSchema["seconds"].Description["en"])
	assert.Equal(t, "tiered_expr", pricing["alias-model"].BillingMode)
	assert.Equal(t, `u("seconds")`, pricing["alias-model"].BillingExpr)
	assert.Equal(t, "tiered_expr", pricing["pricing-usage-model"].BillingMode)
	assert.Equal(t, `u("seconds")`, pricing["pricing-usage-model"].BillingExpr)

	ownMapping := `{"alias-own-expr":"pricing-usage-model"}`
	own := &Channel{
		Id:           911,
		Type:         constant.ChannelTypeTaskPlugin,
		Key:          "key-911",
		Status:       1,
		Name:         "channel-911",
		Models:       "alias-own-expr,pricing-usage-model",
		ModelMapping: &ownMapping,
	}
	require.NoError(t, DB.Create(own).Error)
	insertPricingEndpointAbility(t, 911, "alias-own-expr")
	InitChannelCache()
	InvalidatePricingCache()

	refreshed := pricingByModel(GetPricing())
	assert.Equal(t, `u("seconds") * 2`, refreshed["alias-own-expr"].BillingExpr)
	assert.Equal(t, "second", refreshed["alias-own-expr"].BillingUsageSchema["seconds"].Unit)
}

func TestInitChannelCachePublishesAliasViewBeforePricingInvalidation(t *testing.T) {
	for _, memoryCacheEnabled := range []bool{false, true} {
		memoryCacheEnabled := memoryCacheEnabled
		t.Run(fmt.Sprintf("memory_cache_enabled=%t", memoryCacheEnabled), func(t *testing.T) {
			resetPricingEndpointTestTables(t)
			common.MemoryCacheEnabled = memoryCacheEnabled

			const (
				aliasModel  = "pricing-generation-alias"
				firstKey    = "pricing-generation-first"
				firstModel  = "pricing-generation-first-model"
				secondKey   = "pricing-generation-second"
				secondModel = "pricing-generation-second-model"
			)
			_, err := jsplugin.DefaultRegistry.Register(pricingUsagePluginSourceFor(firstKey, firstModel, "1.0.0", `{
  seconds: {type: "number", unit: "second", description: "First generation."}
}`), jsplugin.Options{})
			require.NoError(t, err)
			_, err = jsplugin.DefaultRegistry.Register(pricingUsagePluginSourceFor(secondKey, secondModel, "1.0.0", `{
  clips: {type: "number", unit: "count", description: "Second generation."}
}`), jsplugin.Options{})
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, jsplugin.DefaultRegistry.Unregister(firstKey))
				require.NoError(t, jsplugin.DefaultRegistry.Unregister(secondKey))
			})

			mapping := fmt.Sprintf(`{%q:%q}`, aliasModel, firstModel)
			channel := &Channel{
				Id:           912,
				Type:         constant.ChannelTypeTaskPlugin,
				Key:          "key-912",
				Status:       common.ChannelStatusEnabled,
				Name:         "channel-912",
				Models:       fmt.Sprintf("%s,%s,%s", aliasModel, firstModel, secondModel),
				ModelMapping: &mapping,
			}
			require.NoError(t, DB.Create(channel).Error)
			insertPricingEndpointAbility(t, 912, aliasModel)

			saved := map[string]string{}
			require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
				saved[key] = value
				return nil
			}))
			t.Cleanup(func() {
				require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
			})
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
				"billing_setting.billing_mode": fmt.Sprintf(`{%q:"tiered_expr",%q:"tiered_expr"}`, firstModel, secondModel),
				"billing_setting.billing_expr": fmt.Sprintf(`{%q:"u(\"seconds\")",%q:"u(\"clips\")"}`, firstModel, secondModel),
			}))

			InitChannelCache()
			initial := pricingByModel(GetPricing())[aliasModel]
			require.Equal(t, `u("seconds")`, initial.BillingExpr)
			require.Contains(t, initial.BillingUsageSchema, "seconds")

			updatedMapping := fmt.Sprintf(`{%q:%q}`, aliasModel, secondModel)
			require.NoError(t, DB.Model(&Channel{}).Where("id = ?", 912).Update("model_mapping", updatedMapping).Error)

			const channelWaitTimeout = 5 * time.Second
			rebuildStarted := make(chan struct{})
			releaseRebuild := make(chan struct{})
			rebuildPauseResult := make(chan error, 1)
			var pauseRebuild sync.Once
			var releaseRebuildOnce sync.Once
			var cleanupSyncOnce sync.Once
			callbackName := fmt.Sprintf("test:pause-task-alias-rebuild:%t", memoryCacheEnabled)
			require.NoError(t, DB.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
				if _, isAliasViewQuery := tx.Statement.Dest.(*[]Channel); !isAliasViewQuery {
					return
				}
				pauseRebuild.Do(func() {
					close(rebuildStarted)
					timer := time.NewTimer(channelWaitTimeout)
					defer timer.Stop()
					select {
					case <-releaseRebuild:
						rebuildPauseResult <- nil
					case <-timer.C:
						rebuildPauseResult <- fmt.Errorf(
							"timed out after %s waiting to release the paused task alias rebuild",
							channelWaitTimeout,
						)
					}
				})
			}))

			cacheSyncDone := make(chan struct{})
			go func() {
				defer close(cacheSyncDone)
				InitChannelCache()
			}()
			cleanupSync := func() {
				cleanupSyncOnce.Do(func() {
					releaseRebuildOnce.Do(func() { close(releaseRebuild) })

					var cleanupErr error
					timer := time.NewTimer(channelWaitTimeout)
					select {
					case <-cacheSyncDone:
						timer.Stop()
					case <-timer.C:
						cleanupErr = fmt.Errorf(
							"timed out after %s waiting for InitChannelCache worker cleanup",
							channelWaitTimeout,
						)
					}
					select {
					case pauseErr := <-rebuildPauseResult:
						cleanupErr = errors.Join(cleanupErr, pauseErr)
					default:
					}
					cleanupErr = errors.Join(cleanupErr, DB.Callback().Query().Remove(callbackName))
					require.NoError(t, cleanupErr, "task alias cache sync cleanup must finish before shared fixture cleanup")
				})
			}
			t.Cleanup(cleanupSync)

			startTimer := time.NewTimer(channelWaitTimeout)
			select {
			case <-rebuildStarted:
				startTimer.Stop()
			case <-startTimer.C:
				require.FailNow(t, "timed out waiting for the task alias rebuild query to start", "timeout=%s", channelWaitTimeout)
			}
			GetPricing()
			// Exercise the same release-and-join cleanup used by early failure paths.
			cleanupSync()
			workerFinished := false
			select {
			case <-cacheSyncDone:
				workerFinished = true
			default:
			}
			require.True(t, workerFinished, "task alias cache sync cleanup must join its worker")

			updated := pricingByModel(GetPricing())[aliasModel]
			assert.Equal(t, `u("clips")`, updated.BillingExpr)
			assert.Contains(t, updated.BillingUsageSchema, "clips")
			assert.NotContains(t, updated.BillingUsageSchema, "seconds")
		})
	}
}

func pricingByModel(pricings []Pricing) map[string]Pricing {
	result := make(map[string]Pricing, len(pricings))
	for _, pricing := range pricings {
		result[pricing.ModelName] = pricing
	}
	return result
}
