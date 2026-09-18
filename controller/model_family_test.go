package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListModelsExpandsReasoningFamilies(t *testing.T) {
	withSelfUseModeDisabled(t)
	prices, ratios := ratio_setting.ModelPrice2JSONString(), ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(prices))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(ratios))
		model.InvalidatePricingCache()
	})
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"gpt-5.6-sol":2,"gpt-5.6-terra":1,"gpt-5.6-luna":0.1,"gpt-6-astra":0}`))
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.Channel{Id: 811, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Key: "fixture"}).Error)
	for _, name := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-6-astra", "gpt-6-astra-*"} {
		require.NoError(t, db.Create(&model.Ability{Group: "default", Model: name, ChannelId: 811, Enabled: true}).Error)
	}
	require.NoError(t, db.Create(&model.Ability{Group: "vip", Model: "private-model", ChannelId: 811, Enabled: true}).Error)

	// Counts describe the public legal combinations: 21 per GPT-5.6 family,
	// 18 for Astra, and 17 Astra aliases without its bare base.
	for _, tc := range []struct {
		name  string
		limit map[string]bool
		count int
	}{
		{name: "all base families", count: 81},
		{name: "base token", limit: map[string]bool{"gpt-6-astra": true}, count: 18},
		{name: "wildcard token", limit: map[string]bool{"gpt-6-astra-*": true}, count: 17},
		{name: "specific token", limit: map[string]bool{"gpt-6-astra-pro-max": true}, count: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
			if tc.limit != nil {
				common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
				common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, tc.limit)
			}
			ListModels(ctx, constant.ChannelTypeOpenAI)
			payload := decodeListModelsPayload(t, recorder)
			assert.Len(t, payload.Data, tc.count)
			ids := make(map[string]bool)
			for _, item := range payload.Data {
				assert.NotContains(t, item.Id, "*")
				assert.False(t, ids[item.Id], "duplicate model %s", item.Id)
				ids[item.Id] = true
				assert.Equal(t, "openai", item.OwnedBy)
			}
			assert.True(t, ids["gpt-6-astra-pro-max"])
			assert.False(t, ids["gpt-6-astra-none"])
			assert.False(t, ids["gpt-6-astra-minimal"])
			assert.False(t, ids["private-model"])
		})
	}
}

func TestEnabledModelsOffersBasePricingOnce(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.Channel{Id: 811, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Key: "fixture"}).Error)
	for _, name := range []string{"gpt-6-astra", "gpt-6-astra-*", "gpt-6-astra-pro-max", "gpt-5.6-sol-*", "gpt-5.6-sol-high", "custom-model"} {
		require.NoError(t, db.Create(&model.Ability{Group: "default", Model: name, ChannelId: 811, Enabled: true}).Error)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	EnabledListModels(ctx)
	assert.ElementsMatch(t, []string{"gpt-6-astra", "gpt-5.6-sol", "custom-model"}, decodeUserModelsResponse(t, recorder))
}

func TestReasoningFamilySelectorsAndMetadata(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{Id: 1814, Username: "family-user", Group: "default", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.Channel{Id: 814, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Key: "fixture"}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "gpt-6-astra", ChannelId: 814, Enabled: true}).Error)
	model.InvalidatePricingCache()
	model.GetPricing()
	t.Cleanup(model.InvalidatePricingCache)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=default", nil)
	ctx.Set("id", 1814)
	GetUserModels(ctx)
	assert.Contains(t, decodeUserModelsResponse(t, recorder), "gpt-6-astra-pro-max")

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	DashboardListModels(ctx)
	var dashboard struct {
		Data map[int][]string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &dashboard))
	assert.Contains(t, dashboard.Data[constant.ChannelTypeOpenAI], "gpt-5.6-sol-pro-max")
	assert.Contains(t, dashboard.Data[constant.ChannelTypeOpenAI], "gpt-6-astra-high")

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "model", Value: "gpt-6-astra-pro-max"}}
	RetrieveModel(ctx, constant.ChannelTypeOpenAI)
	var metadata struct {
		ID string `json:"id"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &metadata))
	assert.Equal(t, "gpt-6-astra-pro-max", metadata.ID)
	assert.Contains(t, buildOpenAIModel("gpt-6-astra-pro-max", nil).SupportedEndpointTypes, constant.EndpointTypeOpenAIResponse)
}

// The routing guarantee behind base-only channel registration: every published
// variant resolves back to the base that a channel actually registered. This
// covers GLM too, whose collapse lives in FormatMatchingModelName rather than
// in the suffix normalizer.
func TestExpandedVariantsRouteBackToRegisteredBase(t *testing.T) {
	for _, base := range []string{
		"claude-fable-5", "claude-fable-5-1", "claude-opus-5", "claude-sonnet-5",
		"gemini-3.1-flash-lite", "gemini-3.5-flash", "gemini-3.7-flash",
		"gemini-3.8-flash", "gemini-3.1-pro-preview",
		"glm-5.3", "glm-5.3-flash", "glm-5.3-flashx",
		"deepseek-flash", "deepseek-v4-pro", "kimi-k3", "grok-4.6",
	} {
		t.Run(base, func(t *testing.T) {
			variants := reasoning.ExpandOpenAIReasoningModels([]string{base})
			require.Greater(t, len(variants), 1)
			for _, variant := range variants {
				assert.Contains(t, model.ModelMatchCandidates(variant), base, variant)
			}
		})
	}
}

func TestBuildOpenAIModelGivesVariantsTheBaseOwner(t *testing.T) {
	for _, tc := range []struct {
		base    string
		variant string
	}{
		{base: "claude-opus-5", variant: "claude-opus-5-high"},
		{base: "gemini-3.5-flash", variant: "gemini-3.5-flash-minimal"},
		{base: "deepseek-v4-pro", variant: "deepseek-v4-pro-none"},
		{base: "kimi-k3", variant: "kimi-k3-max"},
	} {
		t.Run(tc.variant, func(t *testing.T) {
			owners := map[string]string{tc.base: "fixture-owner"}
			assert.Equal(t, "fixture-owner", buildOpenAIModel(tc.variant, owners).OwnedBy)
			// Without an owner entry the variant must not be left as "custom"
			// when the base model is a known one.
			assert.Equal(t,
				buildOpenAIModel(tc.base, nil).OwnedBy,
				buildOpenAIModel(tc.variant, nil).OwnedBy,
			)
		})
	}
}

func TestEnabledModelsCollapsesEveryFamilyToItsBase(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.Channel{Id: 812, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Key: "fixture"}).Error)
	for _, name := range []string{
		"claude-fable-5", "claude-fable-5-high", "claude-fable-5-max",
		"deepseek-v4-pro-none", "deepseek-v4-pro-high",
		"kimi-k3-low", "grok-4.6-xhigh", "custom-model",
	} {
		require.NoError(t, db.Create(&model.Ability{Group: "default", Model: name, ChannelId: 812, Enabled: true}).Error)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	EnabledListModels(ctx)
	assert.ElementsMatch(t,
		[]string{"claude-fable-5", "deepseek-v4-pro", "kimi-k3", "grok-4.6", "custom-model"},
		decodeUserModelsResponse(t, recorder),
	)
}
