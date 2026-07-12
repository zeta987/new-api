package oairesponses

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIResponsesRequestToClaudeMessagesUsesSonnet5AdaptiveEffort(t *testing.T) {
	topP := 0.8
	temperature := 0.7
	request := &dto.OpenAIResponsesRequest{
		Model:       "claude-sonnet-5",
		Input:       []byte(`"hello"`),
		Temperature: &temperature,
		TopP:        &topP,
		Reasoning:   &dto.Reasoning{Effort: "xhigh"},
	}

	converted, err := OpenAIResponsesRequestToClaudeMessages(nil, request)
	require.NoError(t, err)
	require.Equal(t, "claude-sonnet-5", converted.Model)
	require.NotNil(t, converted.Thinking)
	require.Equal(t, "adaptive", converted.Thinking.Type)
	require.Equal(t, "summarized", converted.Thinking.Display)
	require.Nil(t, converted.Thinking.BudgetTokens)
	require.Equal(t, "xhigh", gjson.GetBytes(converted.OutputConfig, "effort").String())
	require.Nil(t, converted.Temperature)
	require.Nil(t, converted.TopP)
	require.Nil(t, converted.TopK)
}

func TestOpenAIResponsesRequestToClaudeMessagesUsesSonnet5EffortSuffix(t *testing.T) {
	request := &dto.OpenAIResponsesRequest{
		Model: "claude-sonnet-5-max",
		Input: []byte(`"hello"`),
	}

	converted, err := OpenAIResponsesRequestToClaudeMessages(nil, request)
	require.NoError(t, err)
	require.Equal(t, "claude-sonnet-5", converted.Model)
	require.NotNil(t, converted.Thinking)
	require.Equal(t, "adaptive", converted.Thinking.Type)
	require.Equal(t, "max", gjson.GetBytes(converted.OutputConfig, "effort").String())
}

func TestOpenAIResponsesRequestToClaudeMessagesUsesSonnet5ThinkingSuffix(t *testing.T) {
	request := &dto.OpenAIResponsesRequest{
		Model: "claude-sonnet-5-thinking",
		Input: []byte(`"hello"`),
	}

	converted, err := OpenAIResponsesRequestToClaudeMessages(nil, request)
	require.NoError(t, err)
	require.Equal(t, "claude-sonnet-5", converted.Model)
	require.NotNil(t, converted.Thinking)
	require.Equal(t, "adaptive", converted.Thinking.Type)
	require.Equal(t, "summarized", converted.Thinking.Display)
	require.Nil(t, converted.Thinking.BudgetTokens)
	require.Equal(t, "high", gjson.GetBytes(converted.OutputConfig, "effort").String())
}

func TestOpenAIResponsesRequestToClaudeMessagesOmitsSamplingForSonnet5(t *testing.T) {
	topP := 0.8
	temperature := 0.7
	request := &dto.OpenAIResponsesRequest{
		Model:       "claude-sonnet-5",
		Input:       []byte(`"hello"`),
		Temperature: &temperature,
		TopP:        &topP,
	}

	converted, err := OpenAIResponsesRequestToClaudeMessages(nil, request)
	require.NoError(t, err)
	require.Nil(t, converted.Thinking)
	require.Empty(t, converted.OutputConfig)
	require.Nil(t, converted.Temperature)
	require.Nil(t, converted.TopP)
	require.Nil(t, converted.TopK)
}
