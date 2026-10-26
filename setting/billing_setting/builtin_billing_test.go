package billing_setting_test

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGPT6AstraBuiltinBilling(t *testing.T) {
	settings := config.GlobalConfig.Get("billing_setting").(*billing_setting.BillingSetting)
	saved := *settings
	savedRatios, savedPrices := ratio_setting.ModelRatio2JSONString(), ratio_setting.ModelPrice2JSONString()
	savedOptions := common.OptionMap
	t.Cleanup(func() {
		*settings, common.OptionMap = saved, savedOptions
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(savedRatios))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(savedPrices))
	})
	common.OptionMap = map[string]string{"billing_setting.billing_mode": `{}`, "billing_setting.billing_expr": `{}`}
	require.NoError(t, config.GlobalConfig.LoadFromDB(common.OptionMap))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
	assert.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingMode("gpt-6-astra"))
	expression, ok := billing_setting.GetBillingExpr("gpt-6-astra")
	require.True(t, ok)
	for _, alias := range []string{"gpt-6-astra-low", "gpt-6-astra-pro-max", "gpt-6-astra-*"} {
		assert.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingMode(alias))
		actual, found := billing_setting.GetBillingExpr(alias)
		require.True(t, found)
		assert.Equal(t, expression, actual)
	}

	for _, tc := range []struct {
		name                                       string
		input, output, cached, written, written12h int
		request                                    string
		quota                                      int
	}{
		{"standard", 1000, 100, 0, 0, 0, `{}`, 7500},
		{"client flex cannot discount standard pricing", 1000, 100, 0, 0, 0, `{"service_tier":"flex"}`, 7500},
		{"cache at context boundary", 272000, 1000, 200000, 20000, 0, `{}`, 510000},
		{"whole request above boundary", 272001, 1000, 200000, 20000, 0, `{}`, 1007510},
		// The prompt caching guide example: 4000 of the 7000 writes are 12h.
		{"12h cache writes", 12000, 0, 4000, 7000, 4000, `{}`, 75750},
		{"12h cache writes above boundary", 300000, 0, 200000, 20000, 10000, `{}`, 1375000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usage := &dto.Usage{
				PromptTokens: tc.input, CompletionTokens: tc.output,
				PromptTokensDetails: dto.InputTokenDetails{CachedTokens: tc.cached, CacheWriteTokens: tc.written,
					CacheWriteTokensDetails: &dto.CacheWriteTokenDetails{TTL30mTokens: tc.written - tc.written12h, TTL12hTokens: tc.written12h}},
			}
			params := service.BuildTieredTokenParams(usage, false, billingexpr.UsedVars(expression))
			result, err := billingexpr.ComputeTieredQuotaWithRequest(&billingexpr.BillingSnapshot{
				ExprString: expression, GroupRatio: 1, QuotaPerUnit: 500000,
			}, params, billingexpr.RequestInput{Body: []byte(tc.request)})
			require.NoError(t, err)
			assert.Equal(t, tc.quota, result.ActualQuotaAfterGroup)
		})
	}

	t.Run("admin options expose defaults without persisting them", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		controller.GetOptions(ctx)
		var response struct {
			Success bool
			Data    []model.Option
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success)
		found := map[string]string{}
		for _, option := range response.Data {
			if _, ok := common.OptionMap[option.Key]; ok {
				assert.NotContains(t, found, option.Key)
				var values map[string]string
				require.NoError(t, common.UnmarshalJsonStr(option.Value, &values))
				found[option.Key] = values["gpt-6-astra"]
				assert.Equal(t, `{}`, common.OptionMap[option.Key])
			}
		}
		assert.Equal(t, map[string]string{"billing_setting.billing_mode": "tiered_expr", "billing_setting.billing_expr": expression}, found)
	})

	for _, tc := range []struct {
		name, mode, expr, ratios, prices, wantMode string
	}{
		{"custom expression overrides legacy price", "tiered_expr", "p * 7", `{"gpt-6-astra":8}`, `{}`, "tiered_expr"},
		{"explicit ratio mode", "ratio", "", `{}`, `{}`, "ratio"},
		{"existing free token price", "", "", `{"gpt-6-astra":0}`, `{}`, "ratio"},
		{"existing per-call price", "", "", `{}`, `{"gpt-6-astra":0.1}`, "ratio"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			*settings = billing_setting.BillingSetting{BillingMode: map[string]string{}, BillingExpr: map[string]string{}}
			if tc.mode != "" {
				settings.BillingMode["gpt-6-astra"] = tc.mode
			}
			if tc.expr != "" {
				settings.BillingExpr["gpt-6-astra"] = tc.expr
			}
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(tc.ratios))
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(tc.prices))
			assert.Equal(t, tc.wantMode, billing_setting.GetBillingMode("gpt-6-astra"))
			actual, ok := billing_setting.GetBillingExpr("gpt-6-astra")
			assert.Equal(t, tc.expr, actual)
			assert.Equal(t, tc.expr != "", ok)
		})
	}
	t.Run("explicit alias and wildcard prices override built-in", func(t *testing.T) {
		*settings = billing_setting.BillingSetting{BillingMode: map[string]string{}, BillingExpr: map[string]string{}}
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
		for _, priceKey := range []string{"gpt-6-astra-pro-max", "gpt-6-astra-*"} {
			prices, err := common.Marshal(map[string]float64{priceKey: 0})
			require.NoError(t, err)
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(string(prices)))
			assert.Equal(t, billing_setting.BillingModeRatio, billing_setting.GetBillingMode("gpt-6-astra-pro-max"))
			_, found := billing_setting.GetBillingExpr("gpt-6-astra-pro-max")
			assert.False(t, found)
		}
	})
}

func TestNewOfficialModelsBuiltinBilling(t *testing.T) {
	settings := config.GlobalConfig.Get("billing_setting").(*billing_setting.BillingSetting)
	saved := *settings
	savedRatios, savedPrices := ratio_setting.ModelRatio2JSONString(), ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		*settings = saved
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(savedRatios))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(savedPrices))
	})
	*settings = billing_setting.BillingSetting{BillingMode: map[string]string{}, BillingExpr: map[string]string{}}
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))

	for _, tc := range []struct {
		name         string
		model, alias string
		usage        dto.Usage
		claude       bool
		wantQuota    int
	}{
		{"sol cache rates", "gpt-6-sol", "gpt-6-sol-pro-max", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200, CacheWriteTokens: 100}}, false, 1345},
		{"sol context boundary", "gpt-6-sol", "gpt-6-sol-high", dto.Usage{PromptTokens: 272000, CompletionTokens: 1000}, false, 277000},
		{"sol long context", "gpt-6-sol", "gpt-6-sol-high", dto.Usage{PromptTokens: 272001, CompletionTokens: 1000}, false, 551502},
		{"luna cache rates", "gpt-6-luna", "gpt-6-luna-none", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200, CacheWriteTokens: 100}}, false, 67},
		{"luna context boundary", "gpt-6-luna", "gpt-6-luna-high", dto.Usage{PromptTokens: 272000, CompletionTokens: 1000}, false, 13850},
		{"luna long context", "gpt-6-luna", "gpt-6-luna-high", dto.Usage{PromptTokens: 272001, CompletionTokens: 1000}, false, 27575},
		{"opus cache rates", "claude-opus-5-5", "claude-opus-5-5-max", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200}, ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, true, 3470},
		{"opus long context standard rates", "claude-opus-5-5", "claude-opus-5-5-high", dto.Usage{PromptTokens: 300000, CompletionTokens: 1000, UsageSemantic: "anthropic"}, true, 610000},
		{"sonnet 5.5 cache rates", "claude-sonnet-5-5", "claude-sonnet-5-5-high", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200}, ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, true, 1735},
		{"fable 5.1 cache rates", "claude-fable-5-1", "claude-fable-5-1-max", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200}, ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, true, 8650},
		{"mythos 5.1 cache rates", "claude-mythos-5-1", "claude-mythos-5-1-high", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200}, ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, true, 8650},
		{"sonnet 5 dated alias", "claude-sonnet-5", "claude-sonnet-5-20260630-xhigh", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200}, ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, true, 1745},
		{"fable 5 cache rates", "claude-fable-5", "claude-fable-5-high", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200}, ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, true, 8725},
		{"mythos 5 cache rates", "claude-mythos-5", "claude-mythos-5", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200}, ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, true, 8725},
		{"haiku cache rates", "claude-haiku-5-5", "claude-haiku-5-5-low", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200}, ClaudeCacheCreation5mTokens: 400, ClaudeCacheCreation1hTokens: 100}, true, 111},
		{"haiku context boundary counts cache", "claude-haiku-5-5", "claude-haiku-5-5-high", dto.Usage{PromptTokens: 60000, CompletionTokens: 1000, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 40000}}, true, 3450},
		{"haiku long context counts cache", "claude-haiku-5-5", "claude-haiku-5-5-high", dto.Usage{PromptTokens: 60000, CompletionTokens: 1000, UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 40002}}, true, 17250},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingMode(tc.alias))
			expression, ok := billing_setting.GetBillingExpr(tc.alias)
			require.True(t, ok)
			params := service.BuildTieredTokenParams(&tc.usage, tc.claude, billingexpr.UsedVars(expression))
			result, err := billingexpr.ComputeTieredQuota(&billingexpr.BillingSnapshot{
				ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), GroupRatio: 1, QuotaPerUnit: 500000,
			}, params)
			require.NoError(t, err)
			assert.Equal(t, tc.wantQuota, result.ActualQuotaAfterGroup)
		})
	}

	// A default legacy ratio row would take precedence over these built-in prices.
	defaultRatios, err := common.Marshal(ratio_setting.GetDefaultModelRatioMap())
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(string(defaultRatios)))
	for _, alias := range []string{"claude-sonnet-5", "claude-sonnet-5-high", "claude-fable-5", "claude-fable-5-max", "claude-mythos-5", "claude-mythos-5-1"} {
		assert.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingMode(alias), alias)
	}
}

func TestGeminiArgonBuiltinBillingAndOverrides(t *testing.T) {
	settings := config.GlobalConfig.Get("billing_setting").(*billing_setting.BillingSetting)
	saved := *settings
	savedRatios, savedPrices := ratio_setting.ModelRatio2JSONString(), ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		*settings = saved
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(savedRatios))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(savedPrices))
	})
	*settings = billing_setting.BillingSetting{BillingMode: map[string]string{}, BillingExpr: map[string]string{}}
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
	wantExpression := `tier("standard", p * 4 + c * 20 + cr * 0.2)`
	wantQuota := 2620
	for _, name := range []string{"gemini-4-argon", "gemini-4-argon-low", "gemini-4-argon-medium", "gemini-4-argon-high"} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingMode(name))
			expression, ok := billing_setting.GetBillingExpr(name)
			require.True(t, ok)
			assert.Equal(t, wantExpression, expression)
			usage := &dto.Usage{PromptTokens: 1000, CompletionTokens: 100, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 200, AudioTokens: 300}}
			result, err := billingexpr.ComputeTieredQuota(&billingexpr.BillingSnapshot{
				ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), GroupRatio: 1, QuotaPerUnit: 500000,
			}, service.BuildTieredTokenParams(usage, false, billingexpr.UsedVars(expression)))
			require.NoError(t, err)
			assert.Equal(t, wantQuota, result.ActualQuotaAfterGroup)
		})
	}
	assert.Equal(t, wantExpression, billing_setting.GetBuiltinBillingExprCopy()["gemini-4-argon"])
	assert.Equal(t, wantExpression, billing_setting.GetBillingExprCopy()["gemini-4-argon"])
	assert.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingModeCopy()["gemini-4-argon"])

	for _, priceKey := range []string{"gemini-4-argon", "gemini-4-argon-high", "gemini-4-argon-*"} {
		encoded, err := common.Marshal(map[string]float64{priceKey: 0})
		require.NoError(t, err)
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(string(encoded)))
		assert.Equal(t, billing_setting.BillingModeRatio, billing_setting.GetBillingMode("gemini-4-argon-high"))
		_, ok := billing_setting.GetBillingExpr("gemini-4-argon-high")
		assert.False(t, ok)
	}
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"gemini-4-argon":0.01}`))
	assert.Equal(t, billing_setting.BillingModeRatio, billing_setting.GetBillingMode("gemini-4-argon-low"))
	settings.BillingMode["gemini-4-argon"] = billing_setting.BillingModeTieredExpr
	settings.BillingExpr["gemini-4-argon"] = `tier("custom", p * 7)`
	expression, ok := billing_setting.GetBillingExpr("gemini-4-argon-medium")
	require.True(t, ok)
	assert.Equal(t, `tier("custom", p * 7)`, expression)
}

func TestKimiK31BuiltinBillingAndOverrides(t *testing.T) {
	settings := config.GlobalConfig.Get("billing_setting").(*billing_setting.BillingSetting)
	saved := *settings
	savedRatios, savedPrices := ratio_setting.ModelRatio2JSONString(), ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		*settings = saved
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(savedRatios))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(savedPrices))
	})

	for _, modelCase := range []struct {
		model, expression                 string
		mixedQuota, writeQuota, longQuota int
	}{
		{"kimi-k3.1-pro", `tier("standard", p * 2 + c * 10 + cr * 0.2 + cc * 2 + cc1h * 4)`, 1720, 1720, 300500},
		{"kimi-k3.1-pro-highspeed", `tier("standard", p * 4 + c * 20 + cr * 0.2 + cc * 4 + cc1h * 8)`, 3420, 3420, 601000},
	} {
		t.Run(modelCase.model, func(t *testing.T) {
			*settings = billing_setting.BillingSetting{BillingMode: map[string]string{}, BillingExpr: map[string]string{}}
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
			for _, alias := range []string{modelCase.model, modelCase.model + "-low", modelCase.model + "-high", modelCase.model + "-max"} {
				assert.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingMode(alias), alias)
				expression, ok := billing_setting.GetBillingExpr(alias)
				require.True(t, ok, alias)
				assert.Equal(t, modelCase.expression, expression, alias)
			}
			for _, usageCase := range []struct {
				name  string
				usage dto.Usage
				quota int
			}{
				{"openai 5m and 1h writes", dto.Usage{PromptTokens: 1350, CompletionTokens: 100, UsageSemantic: "openai",
					PromptTokensDetails:         dto.InputTokenDetails{CachedTokens: 200, CacheWriteTokens: 150},
					ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, modelCase.mixedQuota},
				{"anthropic 5m and 1h writes", dto.Usage{PromptTokens: 1000, CompletionTokens: 100, UsageSemantic: "anthropic",
					PromptTokensDetails:         dto.InputTokenDetails{CachedTokens: 200, CacheWriteTokens: 150},
					ClaudeCacheCreation5mTokens: 100, ClaudeCacheCreation1hTokens: 50}, modelCase.mixedQuota},
				{"openai 1h writes", dto.Usage{PromptTokens: 1300, CompletionTokens: 100, UsageSemantic: "openai",
					PromptTokensDetails:         dto.InputTokenDetails{CachedTokens: 200, CacheWriteTokens: 100},
					ClaudeCacheCreation1hTokens: 100}, modelCase.writeQuota},
				{"long context keeps standard price", dto.Usage{PromptTokens: 300000, CompletionTokens: 100}, modelCase.longQuota},
			} {
				t.Run(usageCase.name, func(t *testing.T) {
					params := service.BuildTieredTokenParams(&usageCase.usage, usageCase.usage.UsageSemantic == "anthropic", billingexpr.UsedVars(modelCase.expression))
					result, err := billingexpr.ComputeTieredQuota(&billingexpr.BillingSnapshot{
						ExprString: modelCase.expression, ExprHash: billingexpr.ExprHashString(modelCase.expression), GroupRatio: 1, QuotaPerUnit: 500000,
					}, params)
					require.NoError(t, err)
					assert.Equal(t, "standard", result.MatchedTier)
					assert.Equal(t, usageCase.quota, result.ActualQuotaAfterGroup)
				})
			}

			alias := modelCase.model + "-max"
			for _, priceKey := range []string{modelCase.model, alias, modelCase.model + "-*"} {
				encoded, err := common.Marshal(map[string]float64{priceKey: 0})
				require.NoError(t, err)
				require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(string(encoded)))
				assert.Equal(t, billing_setting.BillingModeRatio, billing_setting.GetBillingMode(alias), priceKey)
				_, ok := billing_setting.GetBillingExpr(alias)
				assert.False(t, ok, priceKey)
			}
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
			encoded, err := common.Marshal(map[string]float64{modelCase.model: 0.01})
			require.NoError(t, err)
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(string(encoded)))
			assert.Equal(t, billing_setting.BillingModeRatio, billing_setting.GetBillingMode(alias))
			settings.BillingMode[modelCase.model] = billing_setting.BillingModeTieredExpr
			settings.BillingExpr[modelCase.model] = `tier("custom", p * 7)`
			expression, ok := billing_setting.GetBillingExpr(alias)
			require.True(t, ok)
			assert.Equal(t, `tier("custom", p * 7)`, expression)
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
			settings.BillingMode[modelCase.model] = billing_setting.BillingModeRatio
			delete(settings.BillingExpr, modelCase.model)
			assert.Equal(t, billing_setting.BillingModeRatio, billing_setting.GetBillingMode(alias))
			_, ok = billing_setting.GetBillingExpr(alias)
			assert.False(t, ok)
		})
	}
}

func TestImageModelBuiltinPricesAndOverrides(t *testing.T) {
	settings := config.GlobalConfig.Get("billing_setting").(*billing_setting.BillingSetting)
	saved := *settings
	savedRatios, savedPrices := ratio_setting.ModelRatio2JSONString(), ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		*settings = saved
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(savedRatios))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(savedPrices))
	})
	for _, name := range []string{"gpt-image-2", "gpt-image-2.5-sunburst", "gpt-image-2.5-flare"} {
		t.Run(name, func(t *testing.T) {
			*settings = billing_setting.BillingSetting{BillingMode: map[string]string{}, BillingExpr: map[string]string{}}
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
			expression, ok := billing_setting.GetBillingExpr(name)
			require.True(t, ok)
			usage := &dto.Usage{PromptTokens: 1000, CompletionTokens: 100, PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens: 300, ImageTokens: 600, CachedTokensDetails: &dto.CachedTokenDetails{ImageTokens: common.GetPointer(200)},
			}}
			result, err := billingexpr.ComputeTieredQuota(&billingexpr.BillingSnapshot{ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), GroupRatio: 1, QuotaPerUnit: 500000},
				service.BuildTieredTokenParams(usage, false, billingexpr.UsedVars(expression)))
			require.NoError(t, err)
			assert.Equal(t, 4113, result.ActualQuotaAfterGroup)
			encoded, err := common.Marshal(map[string]float64{name: 0})
			require.NoError(t, err)
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(string(encoded)))
			assert.Equal(t, "ratio", billing_setting.GetBillingMode(name))
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(string(encoded)))
			assert.Equal(t, "ratio", billing_setting.GetBillingMode(name))
			settings.BillingMode[name] = "tiered_expr"
			settings.BillingExpr[name] = `tier("custom", p * 7)`
			actual, ok := billing_setting.GetBillingExpr(name)
			require.True(t, ok)
			assert.Equal(t, settings.BillingExpr[name], actual)
		})
	}
}

// DeepSeek prices are peak/off-peak expressions in USD per 1M tokens; peak is
// twice off-peak. The same usage settles to the same quota from a Chat,
// Responses or Anthropic response, and the Search/Fetch routing names cost
// $0.75 per 1,000 requests with no token price. Which tier runs depends on the clock; both are pinned here.
func TestDeepSeekBuiltinBilling(t *testing.T) {
	settings := config.GlobalConfig.Get("billing_setting").(*billing_setting.BillingSetting)
	saved := *settings
	savedRatios, savedPrices := ratio_setting.ModelRatio2JSONString(), ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		*settings = saved
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(savedRatios))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(savedPrices))
	})
	*settings = billing_setting.BillingSetting{BillingMode: map[string]string{}, BillingExpr: map[string]string{}}
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))

	// 1000 cache-miss + 2000 cache-hit input tokens and 500 output tokens.
	chat := &dto.Usage{PromptTokens: 3000, CompletionTokens: 500, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 2000}}
	responses := &dto.Usage{}
	service.ApplyResponsesUsage(responses, &dto.Usage{InputTokens: 3000, OutputTokens: 500, InputTokensDetails: &dto.InputTokenDetails{CachedTokens: 2000}})
	anthropic := &dto.Usage{PromptTokens: 1000, CompletionTokens: 500, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 2000}, UsageSemantic: "anthropic"}
	quota := func(expression string, usage *dto.Usage) billingexpr.TieredResult {
		params := service.BuildTieredTokenParams(usage, usage.UsageSemantic == "anthropic", billingexpr.UsedVars(expression))
		result, err := billingexpr.ComputeTieredQuota(&billingexpr.BillingSnapshot{
			ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), GroupRatio: 1, QuotaPerUnit: 500000,
		}, params)
		require.NoError(t, err)
		return result
	}
	for model, want := range map[string]map[string]int{
		// flash: 1000*0.15 + 2000*0.003 + 500*0.6 = $0.000456 off-peak.
		"deepseek-flash":               {"off_peak": 228, "peak": 456},
		"deepseek-v4-flash":            {"off_peak": 228, "peak": 456},
		"deepseek-v4-flash-vision-exp": {"off_peak": 228, "peak": 456},
		// pro: 1000*0.66 + 2000*0.022 + 500*1.98 = $0.001694 off-peak.
		"deepseek-v4-pro": {"off_peak": 847, "peak": 1694},
	} {
		t.Run(model, func(t *testing.T) {
			assert.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingMode(model))
			expression, ok := billing_setting.GetBillingExpr(model)
			require.True(t, ok)
			for name, usage := range map[string]*dto.Usage{"chat": chat, "responses": responses, "anthropic": anthropic} {
				result := quota(expression, usage)
				require.Contains(t, want, result.MatchedTier, name)
				assert.Equal(t, want[result.MatchedTier], result.ActualQuotaAfterGroup, name)
			}
		})
	}
	for _, model := range []string{"deepseek-search", "deepseek-fetch"} {
		expression, ok := billing_setting.GetBillingExpr(model)
		require.True(t, ok, model)
		result := quota(expression, &dto.Usage{})
		assert.Equal(t, billingexpr.BillingUnitRequest, result.BillingUnit, model)
		assert.Equal(t, 375, result.ActualQuotaAfterGroup, model)
	}
}
