package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/relayconvert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIChatRequestToClaudeMessagesEnablesSonnet46AdaptiveThinking(t *testing.T) {
	req := dto.GeneralOpenAIRequest{
		Model: "claude-sonnet-4-6-high",
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-sonnet-4-6", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "high", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
	require.NotNil(t, claudeReq.Temperature)
	require.Equal(t, 1.0, *claudeReq.Temperature)
	require.Nil(t, claudeReq.TopP)
}

func TestOpenAIChatRequestToClaudeMessagesKeepsOpus47AdaptiveThinkingRestrictions(t *testing.T) {
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

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
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

func TestOpenAIChatRequestToClaudeMessagesKeepsOpus48AdaptiveThinkingRestrictions(t *testing.T) {
	topP := 0.8
	topK := 4
	temperature := 0.7
	req := dto.GeneralOpenAIRequest{
		Model:       "claude-opus-4-8-xhigh",
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

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-opus-4-8", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "summarized", claudeReq.Thinking.Display)
	require.Equal(t, "xhigh", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
	require.Nil(t, claudeReq.Temperature)
	require.Nil(t, claudeReq.TopP)
	require.Nil(t, claudeReq.TopK)
}

func TestOpenAIChatRequestToClaudeMessagesEnablesFable5AdaptiveThinking(t *testing.T) {
	topP := 0.8
	topK := 4
	temperature := 0.7
	req := dto.GeneralOpenAIRequest{
		Model:       "claude-fable-5-max",
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

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-fable-5", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "summarized", claudeReq.Thinking.Display)
	require.Equal(t, "max", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
	require.Nil(t, claudeReq.Temperature)
	require.Nil(t, claudeReq.TopP)
	require.Nil(t, claudeReq.TopK)
}

func TestOpenAIChatRequestToClaudeMessagesOmitsSamplingForSonnet5(t *testing.T) {
	topP := 0.8
	topK := 4
	temperature := 0.7
	req := dto.GeneralOpenAIRequest{
		Model:       "claude-sonnet-5",
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

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-sonnet-5", claudeReq.Model)
	require.Nil(t, claudeReq.Thinking)
	require.Empty(t, claudeReq.OutputConfig)
	require.Nil(t, claudeReq.Temperature)
	require.Nil(t, claudeReq.TopP)
	require.Nil(t, claudeReq.TopK)
}

func TestOpenAIChatRequestToClaudeMessagesEnablesSonnet5EffortSuffix(t *testing.T) {
	topP := 0.8
	topK := 4
	temperature := 0.7
	req := dto.GeneralOpenAIRequest{
		Model:       "claude-sonnet-5-xhigh",
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

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-sonnet-5", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "summarized", claudeReq.Thinking.Display)
	require.Equal(t, "xhigh", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
	require.Nil(t, claudeReq.Temperature)
	require.Nil(t, claudeReq.TopP)
	require.Nil(t, claudeReq.TopK)
}

func TestOpenAIChatRequestToClaudeMessagesMapsSonnet5ReasoningEffort(t *testing.T) {
	req := dto.GeneralOpenAIRequest{
		Model:           "claude-sonnet-5",
		ReasoningEffort: "low",
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-sonnet-5", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "summarized", claudeReq.Thinking.Display)
	require.Nil(t, claudeReq.Thinking.BudgetTokens)
	require.Equal(t, "low", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
}

func TestOpenAIChatRequestToClaudeMessagesMapsSonnet5ReasoningBudgetToAdaptive(t *testing.T) {
	req := dto.GeneralOpenAIRequest{
		Model:     "claude-sonnet-5",
		Reasoning: []byte(`{"max_tokens":32000}`),
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-sonnet-5", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "summarized", claudeReq.Thinking.Display)
	require.Nil(t, claudeReq.Thinking.BudgetTokens)
	require.Equal(t, "high", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
}

func TestOpenAIChatRequestToClaudeMessagesMapsOpus48ThinkingToAdaptiveHigh(t *testing.T) {
	topP := 0.8
	topK := 4
	temperature := 0.7
	req := dto.GeneralOpenAIRequest{
		Model:       "claude-opus-4-8-thinking",
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

	claudeReq, err := relayconvert.OpenAIChatRequestToClaudeMessages(nil, req)
	require.NoError(t, err)

	require.Equal(t, "claude-opus-4-8", claudeReq.Model)
	require.NotNil(t, claudeReq.Thinking)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "summarized", claudeReq.Thinking.Display)
	require.Equal(t, "high", gjson.GetBytes(claudeReq.OutputConfig, "effort").String())
	require.Nil(t, claudeReq.Temperature)
	require.Nil(t, claudeReq.TopP)
	require.Nil(t, claudeReq.TopK)
}
