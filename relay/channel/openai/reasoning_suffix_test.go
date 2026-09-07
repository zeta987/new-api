package openai

import (
	"slices"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponsesRequestAppliesGPT56ReasoningSuffix(t *testing.T) {
	tests := []struct {
		name          string
		request       dto.OpenAIResponsesRequest
		wantModel     string
		wantMode      string
		wantEffort    string
		wantSummary   string
		wantReasoning bool
	}{
		{
			name:          "max keeps mode omitted",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.6-luna-max"},
			wantModel:     "gpt-5.6-luna",
			wantEffort:    "max",
			wantReasoning: true,
		},
		{
			name: "body mode works without a suffix",
			request: dto.OpenAIResponsesRequest{
				Model:     "gpt-5.6-luna",
				Reasoning: &dto.Reasoning{Mode: []byte(`"pro"`), Summary: "auto"},
			},
			wantModel:     "gpt-5.6-luna",
			wantMode:      "pro",
			wantSummary:   "auto",
			wantReasoning: true,
		},
		{
			name:          "pro max sets both fields",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.6-luna-pro-max"},
			wantModel:     "gpt-5.6-luna",
			wantMode:      "pro",
			wantEffort:    "max",
			wantReasoning: true,
		},
		{
			name: "effort suffix preserves body mode",
			request: dto.OpenAIResponsesRequest{
				Model:     "gpt-5.6-luna-high",
				Reasoning: &dto.Reasoning{Mode: []byte(`"pro"`), Effort: "low", Summary: "auto"},
			},
			wantModel:     "gpt-5.6-luna",
			wantMode:      "pro",
			wantEffort:    "high",
			wantSummary:   "auto",
			wantReasoning: true,
		},
		{
			name: "explicit standard suffix overrides body mode",
			request: dto.OpenAIResponsesRequest{
				Model:     "gpt-5.6-luna-standard-medium",
				Reasoning: &dto.Reasoning{Mode: []byte(`"pro"`), Effort: "low"},
			},
			wantModel:     "gpt-5.6-luna",
			wantMode:      "standard",
			wantEffort:    "medium",
			wantReasoning: true,
		},
		{
			name:          "invalid suffix is untouched",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.6-luna-pro-ultra"},
			wantModel:     "gpt-5.6-luna-pro-ultra",
			wantReasoning: false,
		},
		{
			name:          "minimal is not a GPT-5.6 effort",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.6-luna-minimal"},
			wantModel:     "gpt-5.6-luna-minimal",
			wantReasoning: false,
		},
		{
			name:          "codex max model remains intact",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.1-codex-max"},
			wantModel:     "gpt-5.1-codex-max",
			wantReasoning: false,
		},
		{
			name:          "unrelated pro model remains intact",
			request:       dto.OpenAIResponsesRequest{Model: "gpt-5.4-pro"},
			wantModel:     "gpt-5.4-pro",
			wantReasoning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tt.request.Model},
			}
			converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, tt.request)
			require.NoError(t, err)

			got, ok := converted.(dto.OpenAIResponsesRequest)
			require.True(t, ok)
			assert.Equal(t, tt.wantModel, got.Model)
			assert.Equal(t, tt.wantModel, info.UpstreamModelName)
			if !tt.wantReasoning {
				assert.Nil(t, got.Reasoning)
				return
			}
			require.NotNil(t, got.Reasoning)
			if tt.wantMode == "" {
				assert.Nil(t, got.Reasoning.Mode)
			} else {
				require.NotNil(t, got.Reasoning.Mode)
				assert.Equal(t, `"`+tt.wantMode+`"`, string(got.Reasoning.Mode))
			}
			assert.Equal(t, tt.wantEffort, got.Reasoning.Effort)
			assert.Equal(t, tt.wantSummary, got.Reasoning.Summary)
			assert.Equal(t, tt.wantEffort, info.ReasoningEffort)

			encoded, err := common.Marshal(got)
			require.NoError(t, err)
			if tt.wantMode == "" {
				assert.NotContains(t, string(encoded), `"mode"`)
			} else {
				assert.Contains(t, string(encoded), `"mode":"`+tt.wantMode+`"`)
			}
		})
	}
}

func TestConvertOpenAIResponsesRequestUsesOriginalModelSuffixAfterMapping(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.6-luna-pro-max",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.6-terra"},
	}
	request := dto.OpenAIResponsesRequest{Model: "gpt-5.6-terra"}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.NotNil(t, got.Reasoning)

	assert.Equal(t, "gpt-5.6-terra", got.Model)
	assert.Equal(t, "gpt-5.6-terra", info.UpstreamModelName)
	require.NotNil(t, got.Reasoning.Mode)
	assert.Equal(t, `"pro"`, string(got.Reasoning.Mode))
	assert.Equal(t, "max", got.Reasoning.Effort)
}

func TestModelListIncludesGPT56Models(t *testing.T) {
	for _, model := range []string{"gpt-5.6-luna", "gpt-5.6-terra", "gpt-5.6-sol"} {
		assert.Truef(t, slices.Contains(ModelList, model), "ModelList is missing %s", model)
	}
}

func TestQwenReasoningEffortSuffixBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantBase   string
		wantEffort string
		wantOK     bool
	}{
		{name: "minimum max", model: "qwen3.8-max-none", wantBase: "qwen3.8-max", wantEffort: "none", wantOK: true},
		{name: "minimum flash", model: "qwen3.8-flash-low", wantBase: "qwen3.8-flash", wantEffort: "low", wantOK: true},
		{name: "newer minor", model: "qwen3.12-max-medium", wantBase: "qwen3.12-max", wantEffort: "medium", wantOK: true},
		{name: "newer major snapshot", model: "vendor/qwen4.0-flash-2026-09-01-xhigh", wantBase: "vendor/qwen4.0-flash-2026-09-01", wantEffort: "xhigh", wantOK: true},
		{name: "version below minimum", model: "qwen3.7-max-low"},
		{name: "unsupported family", model: "qwen3.8-coder-low"},
		{name: "high is not an alias", model: "qwen3.8-max-high"},
		{name: "max is the family name", model: "qwen3.8-max"},
		{name: "minimal is not an alias", model: "qwen3.8-flash-minimal"},
		{name: "malformed version", model: "qwen03.8-max-low"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			base, effort, ok := reasoning.ParseQwenReasoningEffortSuffix(testCase.model)
			assert.Equal(t, testCase.wantOK, ok)
			if !testCase.wantOK {
				assert.Equal(t, testCase.model, base)
				assert.Empty(t, effort)
				return
			}
			assert.Equal(t, testCase.wantBase, base)
			assert.Equal(t, testCase.wantEffort, effort)
		})
	}
}

func TestQwenReasoningModelsExpandAndMatchTheirBase(t *testing.T) {
	assert.Equal(t, []string{
		"qwen3.8-max",
		"qwen3.8-max-none",
		"qwen3.8-max-low",
		"qwen3.8-max-medium",
		"qwen3.8-max-xhigh",
		"qwen3.7-max",
	}, reasoning.ExpandOpenAIReasoningModels([]string{"qwen3.8-max", "qwen3.7-max"}))

	assert.Equal(t,
		[]string{"qwen3.8-flash-medium", "qwen3.8-flash"},
		model.ModelMatchCandidates("qwen3.8-flash-medium"),
	)
}

func TestQwenReasoningAliasPricingPrefersExactThenBase(t *testing.T) {
	original := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(original))
	})
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{
		"qwen3.8-max": 1.25,
		"qwen3.8-max-low": 2.5
	}`))

	price, ok := ratio_setting.GetModelPrice("qwen3.8-max-medium", false)
	require.True(t, ok)
	assert.Equal(t, 1.25, price)
	price, ok = ratio_setting.GetModelPrice("qwen3.8-max-low", false)
	require.True(t, ok)
	assert.Equal(t, 2.5, price)
}

func TestOpenAIChatQwenAliasesWriteTopLevelEffortAndClearVendorControls(t *testing.T) {
	for _, family := range []string{"max", "flash"} {
		for _, effort := range []string{"none", "low", "medium", "xhigh"} {
			t.Run(family+"_"+effort, func(t *testing.T) {
				alias := "qwen3.8-" + family + "-" + effort
				request := dto.GeneralOpenAIRequest{
					Model:           alias,
					ReasoningEffort: "low",
					EnableThinking:  []byte(`true`),
					ThinkingBudget:  []byte(`4096`),
				}

				got, info := convertOpenAIQwenChat(t, request, "")

				assert.Equal(t, "qwen3.8-"+family, got.Model)
				assert.Equal(t, effort, got.ReasoningEffort)
				assert.Nil(t, got.EnableThinking)
				assert.Nil(t, got.ThinkingBudget)
				assert.Equal(t, effort, info.ReasoningEffort)
			})
		}
	}
}

func TestOpenAIChatBareQwenPreservesProviderDefaultsAndExplicitEffort(t *testing.T) {
	tests := []struct {
		name   string
		effort string
	}{
		{name: "provider default"},
		{name: "explicit effort", effort: "medium"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := dto.GeneralOpenAIRequest{
				Model:           "qwen3.8-max",
				ReasoningEffort: testCase.effort,
				EnableThinking:  []byte(`true`),
				ThinkingBudget:  []byte(`4096`),
			}

			got, info := convertOpenAIQwenChat(t, request, "")

			assert.Equal(t, "qwen3.8-max", got.Model)
			assert.Equal(t, testCase.effort, got.ReasoningEffort)
			assert.JSONEq(t, `true`, string(got.EnableThinking))
			assert.JSONEq(t, `4096`, string(got.ThinkingBudget))
			assert.Equal(t, testCase.effort, info.ReasoningEffort)
		})
	}
}

func TestOpenAIChatQwenMappingAndModifiersKeepPrecedence(t *testing.T) {
	t.Run("base mapping keeps alias effort", func(t *testing.T) {
		request := dto.GeneralOpenAIRequest{
			Model:          "qwen3.8-max-low",
			EnableThinking: []byte(`true`),
			ThinkingBudget: []byte(`4096`),
		}
		got, info := convertOpenAIQwenChat(t, request, `{"qwen3.8-max":"provider/qwen-max"}`)

		assert.Equal(t, "provider/qwen-max", got.Model)
		assert.Equal(t, "low", got.ReasoningEffort)
		assert.Nil(t, got.EnableThinking)
		assert.Nil(t, got.ThinkingBudget)
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("mapped effort modifier wins", func(t *testing.T) {
		request := dto.GeneralOpenAIRequest{
			Model:          "qwen3.8-max-low",
			EnableThinking: []byte(`true`),
			ThinkingBudget: []byte(`4096`),
		}
		got, info := convertOpenAIQwenChat(t, request, `{"qwen3.8-max-low":"qwen3.8-flash@effort:xhigh"}`)

		assert.Equal(t, "qwen3.8-flash", got.Model)
		assert.Equal(t, "xhigh", got.ReasoningEffort)
		assert.Nil(t, got.EnableThinking)
		assert.Nil(t, got.ThinkingBudget)
		assert.Equal(t, "xhigh", info.ReasoningEffort)
	})

	t.Run("explicit budget modifier wins without clearing vendor controls", func(t *testing.T) {
		request := dto.GeneralOpenAIRequest{
			Model:          "qwen3.8-max-low@thinking:8192",
			EnableThinking: []byte(`true`),
			ThinkingBudget: []byte(`4096`),
		}
		got, _ := convertOpenAIQwenChat(t, request, "")

		assert.Equal(t, "qwen3.8-max", got.Model)
		assert.Equal(t, "medium", got.ReasoningEffort)
		assert.JSONEq(t, `true`, string(got.EnableThinking))
		assert.JSONEq(t, `4096`, string(got.ThinkingBudget))
	})
}

func TestOpenAIChatQwenParamOverrideIsFinalAndUpdatesTelemetry(t *testing.T) {
	request := dto.GeneralOpenAIRequest{Model: "qwen3.8-flash-low"}
	got, info := convertOpenAIQwenChat(t, request, "")
	info.ParamOverride = map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{"path": "reasoning_effort", "mode": "set", "value": "xhigh"},
		},
	}
	payload, err := common.Marshal(got)
	require.NoError(t, err)
	payload, err = relaycommon.ApplyParamOverrideWithRelayInfo(payload, info)
	require.NoError(t, err)

	assert.Equal(t, "xhigh", gjson.GetBytes(payload, "reasoning_effort").String())
	assert.Equal(t, "xhigh", info.ReasoningEffort)
}

func TestOpenAIChatQwenAliasHonorsPassThroughAndBlacklist(t *testing.T) {
	const alias = "qwen3.8-max-low"

	t.Run("global pass through", func(t *testing.T) {
		settings := model_setting.GetGlobalSettings()
		original := settings.PassThroughRequestEnabled
		t.Cleanup(func() { settings.PassThroughRequestEnabled = original })
		settings.PassThroughRequestEnabled = true

		request := dto.GeneralOpenAIRequest{Model: alias, EnableThinking: []byte(`true`)}
		got, info := convertOpenAIQwenChat(t, request, "")
		assert.Equal(t, alias, got.Model)
		assert.JSONEq(t, `true`, string(got.EnableThinking))
		assert.Empty(t, info.ReasoningEffort)
	})

	t.Run("thinking suffix blacklist", func(t *testing.T) {
		settings := model_setting.GetGlobalSettings()
		original := append([]string(nil), settings.ThinkingModelBlacklist...)
		t.Cleanup(func() { settings.ThinkingModelBlacklist = original })
		settings.ThinkingModelBlacklist = append(original, alias, "qwen3.8-max")
		originalPrices := ratio_setting.ModelPrice2JSONString()
		t.Cleanup(func() {
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(originalPrices))
		})
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"qwen3.8-max":1.25}`))

		request := dto.GeneralOpenAIRequest{Model: alias, EnableThinking: []byte(`true`)}
		got, info := convertOpenAIQwenChat(t, request, "")
		assert.Equal(t, alias, got.Model)
		assert.JSONEq(t, `true`, string(got.EnableThinking))
		assert.Empty(t, info.ReasoningEffort)
		assert.Equal(t, []string{"qwen3.8-max"}, reasoning.ExpandOpenAIReasoningModels([]string{"qwen3.8-max"}))
		assert.Equal(t, []string{alias}, model.ModelMatchCandidates(alias))
		_, hasPrice := ratio_setting.GetModelPrice(alias, false)
		assert.False(t, hasPrice)
	})
}

func TestQwenReasoningAliasParsingRequiresOpenAIChatChannel(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "qwen3.8-max-low"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "qwen3.8-max-low",
		Request:         request,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAzure,
			UpstreamModelName: "qwen3.8-max-low",
		},
	}

	require.NoError(t, helper.ApplyReasoningModelSuffix(nil, info, request))
	assert.Equal(t, "qwen3.8-max-low", request.Model)
	assert.Nil(t, info.ReasoningState())
}

func convertOpenAIQwenChat(t *testing.T, original dto.GeneralOpenAIRequest, modelMapping string) (*dto.GeneralOpenAIRequest, *relaycommon.RelayInfo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	if modelMapping != "" {
		c.Set("model_mapping", modelMapping)
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: original.Model,
		Request:         &original,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		RelayFormat:     types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: original.Model,
		},
	}
	outbound, err := common.DeepCopy(&original)
	require.NoError(t, err)
	require.NoError(t, helper.ModelMappedHelper(c, info, outbound))
	require.NoError(t, helper.ApplyReasoningModelSuffix(c, info, outbound))
	converted, err := (&Adaptor{}).ConvertOpenAIRequest(c, info, outbound)
	require.NoError(t, err)
	got, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	return got, info
}
