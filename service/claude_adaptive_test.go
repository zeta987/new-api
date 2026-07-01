package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestNormalizeClaudePost46AdaptiveRequestOmitsSamplingForSonnet5(t *testing.T) {
	topP := 0.8
	topK := 4
	temperature := 0.7
	request := &dto.ClaudeRequest{
		Model:       "claude-sonnet-5",
		Temperature: &temperature,
		TopP:        &topP,
		TopK:        &topK,
	}

	NormalizeClaudePost46AdaptiveRequest(request)

	require.Nil(t, request.Temperature)
	require.Nil(t, request.TopP)
	require.Nil(t, request.TopK)
	require.Nil(t, request.Thinking)
	require.Empty(t, request.OutputConfig)
}

func TestNormalizeClaudePost46AdaptiveRequestConvertsManualBudget(t *testing.T) {
	topP := 0.8
	topK := 4
	temperature := 0.7
	request := &dto.ClaudeRequest{
		Model:       "claude-sonnet-5",
		Temperature: &temperature,
		TopP:        &topP,
		TopK:        &topK,
		Thinking: &dto.Thinking{
			Type:         "enabled",
			BudgetTokens: common.GetPointer(32000),
		},
	}

	NormalizeClaudePost46AdaptiveRequest(request)

	require.Nil(t, request.Temperature)
	require.Nil(t, request.TopP)
	require.Nil(t, request.TopK)
	require.NotNil(t, request.Thinking)
	require.Equal(t, "adaptive", request.Thinking.Type)
	require.Equal(t, "summarized", request.Thinking.Display)
	require.Nil(t, request.Thinking.BudgetTokens)
	require.JSONEq(t, `{"effort":"high"}`, string(request.OutputConfig))
}

func TestNormalizeClaudePost46AdaptiveRequestReplacesUnsupportedEffort(t *testing.T) {
	request := &dto.ClaudeRequest{
		Model:        "claude-sonnet-5",
		OutputConfig: []byte(`{"effort":"minimal"}`),
		Thinking: &dto.Thinking{
			Type: "adaptive",
		},
	}

	NormalizeClaudePost46AdaptiveRequest(request)

	require.NotNil(t, request.Thinking)
	require.Equal(t, "adaptive", request.Thinking.Type)
	require.Equal(t, "summarized", request.Thinking.Display)
	require.JSONEq(t, `{"effort":"high"}`, string(request.OutputConfig))
}

func TestSetClaudeAdaptiveEffortRejectsUnsupportedEffort(t *testing.T) {
	request := &dto.ClaudeRequest{Model: "claude-sonnet-5"}

	require.False(t, SetClaudeAdaptiveEffort(request, "minimal"))

	require.Nil(t, request.Thinking)
	require.Empty(t, request.OutputConfig)
}
