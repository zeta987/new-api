package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAstraResponsesAliases(t *testing.T) {
	for _, mode := range []string{"", "standard", "pro"} {
		for _, effort := range []string{"", "low", "medium", "high", "xhigh", "max"} {
			name := "gpt-6-astra"
			if mode != "" {
				name += "-" + mode
			}
			if effort != "" {
				name += "-" + effort
			}
			t.Run(name, func(t *testing.T) {
				req := &dto.OpenAIResponsesRequest{Model: name}
				info := &relaycommon.RelayInfo{OriginModelName: name, Request: req, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, UpstreamModelName: name}}
				require.NoError(t, helper.ApplyReasoningModelSuffix(nil, info, req))
				converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, *req)
				require.NoError(t, err)
				got := converted.(dto.OpenAIResponsesRequest)
				assert.Equal(t, "gpt-6-astra", got.Model)
				if mode == "" && effort == "" {
					assert.Nil(t, got.Reasoning)
					return
				}
				require.NotNil(t, got.Reasoning)
				assert.Equal(t, effort, got.Reasoning.Effort)
				if mode != "" {
					assert.JSONEq(t, `"`+mode+`"`, string(got.Reasoning.Mode))
				}
				assert.Equal(t, effort, info.ReasoningEffort)
			})
		}
	}
}

func TestAstraMappedModelKeepsCapabilityControls(t *testing.T) {
	value := 0.5
	info := &relaycommon.RelayInfo{OriginModelName: "gpt-6-astra", ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, UpstreamModelName: "vendor-astra"}}
	request := dto.OpenAIResponsesRequest{Model: "vendor-astra", Temperature: &value, Reasoning: &dto.Reasoning{Effort: "max"}}
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got := converted.(dto.OpenAIResponsesRequest)
	assert.Equal(t, "vendor-astra", got.Model)
	assert.Equal(t, "max", got.Reasoning.Effort)
	assert.Nil(t, got.Temperature)
	info.OriginModelName = "gpt-6-astra-pro-high"
	_, err = (&Adaptor{}).ConvertOpenAIRequest(nil, info, &dto.GeneralOpenAIRequest{Model: "vendor-astra"})
	require.ErrorContains(t, err, "Responses")
}

func TestAstraBridgeOmitsNullReasoningControls(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{Model: "gpt-6-astra", Reasoning: []byte(`{"mode":null,"context":null}`), Messages: []dto.Message{{Role: "user", Content: "hello"}}}
	converted, err := relayconvert.ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	assert.Nil(t, converted.Reasoning)
}

func TestAstraPreservesExplicitModeAndMaxThroughChatBridge(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{Model: "gpt-6-astra", ReasoningEffort: "max", Reasoning: []byte(`{"mode":"pro","summary":"auto"}`), Messages: []dto.Message{{Role: "user", Content: "hello"}}}
	converted, err := relayconvert.ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, converted.Reasoning)
	assert.Equal(t, "max", converted.Reasoning.Effort)
	assert.JSONEq(t, `"pro"`, string(converted.Reasoning.Mode))
}

func TestAstraRejectsUnsupportedControls(t *testing.T) {
	for _, effort := range []string{"none", "minimal", "ultra"} {
		t.Run(effort, func(t *testing.T) {
			req := dto.OpenAIResponsesRequest{Model: "gpt-6-astra", Reasoning: &dto.Reasoning{Effort: effort}}
			_, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, nil, req)
			require.Error(t, err)
		})
	}
	for _, mode := range []string{`"ultra"`, `true`, `{}`} {
		_, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, nil, dto.OpenAIResponsesRequest{Model: "gpt-6-astra", Reasoning: &dto.Reasoning{Mode: []byte(mode)}})
		require.Error(t, err)
	}
}

func TestAstraRemovesUnsupportedSampling(t *testing.T) {
	value := 0.7
	logprobs := true
	topLogprobs := 3
	chat := &dto.GeneralOpenAIRequest{Model: "gpt-6-astra", Temperature: &value, TopP: &value, LogProbs: &logprobs, TopLogProbs: &topLogprobs, ReasoningEffort: "max"}
	info := &relaycommon.RelayInfo{OriginModelName: chat.Model, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, UpstreamModelName: chat.Model}}
	_, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, chat)
	require.NoError(t, err)
	assert.Nil(t, chat.Temperature)
	assert.Nil(t, chat.TopP)
	assert.Nil(t, chat.LogProbs)
	assert.Nil(t, chat.TopLogProbs)
	assert.Equal(t, "max", chat.ReasoningEffort)
	responses := dto.OpenAIResponsesRequest{Model: "gpt-6-astra", Temperature: &value, TopP: &value, Include: []byte(`["message.output_text.logprobs","reasoning.encrypted_content"]`)}
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, nil, responses)
	require.NoError(t, err)
	got := converted.(dto.OpenAIResponsesRequest)
	assert.Nil(t, got.Temperature)
	assert.Nil(t, got.TopP)
	assert.JSONEq(t, `["reasoning.encrypted_content"]`, string(got.Include))
}

func TestAstraChatRequiresResponsesForToolsAndMode(t *testing.T) {
	for _, payload := range []string{
		`{"model":"gpt-6-astra","tools":[{"type":"function","function":{"name":"lookup"}}]}`,
		`{"model":"gpt-6-astra","reasoning":{"mode":"pro"}}`,
		`{"model":"gpt-6-astra-pro-high"}`,
	} {
		var req dto.GeneralOpenAIRequest
		require.NoError(t, common.UnmarshalJsonStr(payload, &req))
		info := &relaycommon.RelayInfo{OriginModelName: req.Model, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, UpstreamModelName: req.Model}}
		_, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, &req)
		require.ErrorContains(t, err, "Responses")
	}
}
