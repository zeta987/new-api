package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestRequestOpenAI2ClaudeMessageEnablesSonnet46AdaptiveThinking(t *testing.T) {
	req := dto.GeneralOpenAIRequest{
		Model: "claude-sonnet-4-6-high",
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}

	claudeReq, err := RequestOpenAI2ClaudeMessage(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-sonnet-4-6", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "high", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
	require.NotNil(t, claudeReq.Temperature)
	require.Equal(t, 1.0, *claudeReq.Temperature)
	require.Nil(t, claudeReq.TopP)
}

func TestRequestOpenAI2ClaudeMessageKeepsOpus47AdaptiveThinkingRestrictions(t *testing.T) {
	topP := 0.8
	topK := 4
	temperature := 0.7
	req := dto.GeneralOpenAIRequest{
		Model:       "claude-opus-4-7-medium",
		TopP:        &topP,
		TopK:        &topK,
		Temperature: &temperature,
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}

	claudeReq, err := RequestOpenAI2ClaudeMessage(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-opus-4-7", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "summarized", claudeReq.Thinking.Display)
	require.Equal(t, "medium", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
	require.Nil(t, claudeReq.Temperature)
	require.Nil(t, claudeReq.TopP)
	require.Nil(t, claudeReq.TopK)
}
