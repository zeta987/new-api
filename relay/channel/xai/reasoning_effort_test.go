package xai_test

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel/xai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGrokReasoningEffortChatAliases(t *testing.T) {
	tests := []struct {
		alias  string
		model  string
		effort string
	}{
		{alias: "grok-4.5-low", model: "grok-4.5", effort: "low"},
		{alias: "grok-4.5-medium", model: "grok-4.5", effort: "medium"},
		{alias: "grok-4.5-high", model: "grok-4.5", effort: "high"},
		{alias: "grok-4.6-low", model: "grok-4.6", effort: "low"},
		{alias: "grok-4.6-medium", model: "grok-4.6", effort: "medium"},
		{alias: "grok-4.6-high", model: "grok-4.6", effort: "high"},
		{alias: "grok-4.6-xhigh", model: "grok-4.6", effort: "xhigh"},
		{alias: "grok-4.20-xhigh", model: "grok-4.20", effort: "xhigh"},
		{alias: "grok-5.0-xhigh", model: "grok-5.0", effort: "xhigh"},
	}

	for _, test := range tests {
		t.Run(test.alias, func(t *testing.T) {
			request := &dto.GeneralOpenAIRequest{
				Model:           test.alias,
				ReasoningEffort: "low",
			}
			info := xaiTestRelayInfo(test.alias)

			converted, err := (&xai.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
			got := requireXAIChatRequest(t, converted, err)

			assert.Equal(t, test.model, got.Model)
			assert.Equal(t, test.effort, got.ReasoningEffort)
			assert.Equal(t, test.model, info.UpstreamModelName)
			assert.Equal(t, test.effort, info.ReasoningEffort)

			payload, marshalErr := common.Marshal(got)
			require.NoError(t, marshalErr)
			assert.Equal(t, test.model, gjson.GetBytes(payload, "model").String())
			assert.Equal(t, test.effort, gjson.GetBytes(payload, "reasoning_effort").String())
		})
	}
}

func TestGrokReasoningEffortChatUsesMappedUpstreamAlias(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:           "friendly-grok",
		ReasoningEffort: "low",
	}
	info := xaiTestRelayInfo("grok-4.6-xhigh")

	converted, err := (&xai.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	got := requireXAIChatRequest(t, converted, err)

	assert.Equal(t, "grok-4.6", got.Model)
	assert.Equal(t, "xhigh", got.ReasoningEffort)
	assert.Equal(t, "grok-4.6", info.UpstreamModelName)
	assert.Equal(t, "xhigh", info.ReasoningEffort)
}

func TestGrokReasoningEffortChatPassesExplicitEffortWithoutSuffix(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:           "grok-4.6",
		ReasoningEffort: "medium",
	}
	info := xaiTestRelayInfo("grok-4.6")

	converted, err := (&xai.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	got := requireXAIChatRequest(t, converted, err)

	assert.Equal(t, "grok-4.6", got.Model)
	assert.Equal(t, "medium", got.ReasoningEffort)
	assert.Equal(t, "medium", info.ReasoningEffort)
}

func TestGrokReasoningEffortChatLeavesUnsupportedAliasesUnchanged(t *testing.T) {
	aliases := []string{
		"grok-4.4-high",
		"grok-4.5-xhigh",
		"grok-4.6-ultra",
		"grok-4.6-latest-xhigh",
		"grok-4.6-20260801-xhigh",
		"grok-4.6-reasoning-xhigh",
		"grok-4.20-multi-agent-xhigh",
		"grok-4.6-search-xhigh",
	}

	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			request := &dto.GeneralOpenAIRequest{Model: alias}
			info := xaiTestRelayInfo(alias)

			converted, err := (&xai.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
			got := requireXAIChatRequest(t, converted, err)

			assert.Equal(t, alias, got.Model)
			assert.Empty(t, got.ReasoningEffort)
			assert.Equal(t, alias, info.UpstreamModelName)
			assert.Empty(t, info.ReasoningEffort)
		})
	}
}

func TestGrokReasoningEffortResponsesAliases(t *testing.T) {
	tests := []struct {
		alias  string
		model  string
		effort string
	}{
		{alias: "grok-4.5-high", model: "grok-4.5", effort: "high"},
		{alias: "grok-4.6-xhigh", model: "grok-4.6", effort: "xhigh"},
		{alias: "grok-5.0-xhigh", model: "grok-5.0", effort: "xhigh"},
	}

	for _, test := range tests {
		t.Run(test.alias, func(t *testing.T) {
			request := dto.OpenAIResponsesRequest{
				Model: test.alias,
				Reasoning: &dto.Reasoning{
					Effort:  "low",
					Summary: "detailed",
					Mode:    json.RawMessage(`"standard"`),
					Context: json.RawMessage(`{"id":"ctx_123"}`),
				},
			}
			info := xaiTestRelayInfo(test.alias)

			converted, err := (&xai.Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
			require.NoError(t, err)
			got, ok := converted.(dto.OpenAIResponsesRequest)
			require.True(t, ok)

			assert.Equal(t, test.model, got.Model)
			require.NotNil(t, got.Reasoning)
			assert.Equal(t, test.effort, got.Reasoning.Effort)
			assert.Equal(t, "detailed", got.Reasoning.Summary)
			assert.JSONEq(t, `"standard"`, string(got.Reasoning.Mode))
			assert.JSONEq(t, `{"id":"ctx_123"}`, string(got.Reasoning.Context))
			assert.Equal(t, test.model, info.UpstreamModelName)
			assert.Equal(t, test.effort, info.ReasoningEffort)

			payload, marshalErr := common.Marshal(got)
			require.NoError(t, marshalErr)
			assert.Equal(t, test.model, gjson.GetBytes(payload, "model").String())
			assert.Equal(t, test.effort, gjson.GetBytes(payload, "reasoning.effort").String())
		})
	}
}

func TestGrokReasoningEffortResponsesCreatesReasoningForMappedAlias(t *testing.T) {
	request := dto.OpenAIResponsesRequest{Model: "friendly-grok"}
	info := xaiTestRelayInfo("grok-4.6-xhigh")

	converted, err := (&xai.Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "grok-4.6", got.Model)
	require.NotNil(t, got.Reasoning)
	assert.Equal(t, "xhigh", got.Reasoning.Effort)
	assert.Equal(t, "grok-4.6", info.UpstreamModelName)
	assert.Equal(t, "xhigh", info.ReasoningEffort)
}

func TestGrokReasoningEffortResponsesPassesExplicitEffortWithoutSuffix(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model:     "grok-4.6",
		Reasoning: &dto.Reasoning{Effort: "medium"},
	}
	info := xaiTestRelayInfo("grok-4.6")

	converted, err := (&xai.Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "grok-4.6", got.Model)
	require.NotNil(t, got.Reasoning)
	assert.Equal(t, "medium", got.Reasoning.Effort)
	assert.Equal(t, "medium", info.ReasoningEffort)
}

func TestGrokReasoningEffortResponsesLeavesGrok45XHighUnchanged(t *testing.T) {
	request := dto.OpenAIResponsesRequest{Model: "grok-4.5-xhigh"}
	info := xaiTestRelayInfo("grok-4.5-xhigh")

	converted, err := (&xai.Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "grok-4.5-xhigh", got.Model)
	assert.Nil(t, got.Reasoning)
	assert.Equal(t, "grok-4.5-xhigh", info.UpstreamModelName)
	assert.Empty(t, info.ReasoningEffort)
}

func TestXAIModelListIncludesSupportedGrokEffortAliases(t *testing.T) {
	models := (&xai.Adaptor{}).GetModelList()

	for _, model := range []string{
		"grok-4.5",
		"grok-4.5-low",
		"grok-4.5-medium",
		"grok-4.5-high",
		"grok-4.6",
		"grok-4.6-low",
		"grok-4.6-medium",
		"grok-4.6-high",
		"grok-4.6-xhigh",
	} {
		t.Run(model, func(t *testing.T) {
			assert.Contains(t, models, model)
		})
	}
	assert.NotContains(t, models, "grok-4.5-xhigh")
}

func TestGrok3MiniEffortAliasRegression(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "grok-3-mini-high"}
	info := xaiTestRelayInfo("grok-3-mini-high")

	converted, err := (&xai.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	got := requireXAIChatRequest(t, converted, err)

	assert.Equal(t, "grok-3-mini", got.Model)
	assert.Equal(t, "high", got.ReasoningEffort)
	assert.Equal(t, "grok-3-mini", info.UpstreamModelName)
	assert.Equal(t, "high", info.ReasoningEffort)
}

func xaiTestRelayInfo(model string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeXai,
			UpstreamModelName: model,
		},
	}
}

func requireXAIChatRequest(t *testing.T, converted any, conversionErr error) *dto.GeneralOpenAIRequest {
	t.Helper()
	require.NoError(t, conversionErr)
	request, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	return request
}
