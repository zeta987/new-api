package claude

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newClaudeTestContext() *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	return ctx
}

func TestRequestOpenAI2ClaudeMessageOmitsTopPForAdaptiveMaxModel(t *testing.T) {
	topP := 0.8
	request := dto.GeneralOpenAIRequest{
		Model: "claude-opus-4-6-max",
		TopP:  &topP,
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(newClaudeTestContext(), request)
	require.NoError(t, err)
	require.NotNil(t, claudeRequest.Thinking)
	require.Equal(t, "adaptive", claudeRequest.Thinking.Type)
	require.Equal(t, "claude-opus-4-6", claudeRequest.Model)
	require.Nil(t, claudeRequest.TopP)
}

func TestRequestOpenAI2ClaudeMessageOmitsTopPForAdaptiveLowModel(t *testing.T) {
	topP := 0.8
	request := dto.GeneralOpenAIRequest{
		Model: "claude-sonnet-4-6-low",
		TopP:  &topP,
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(newClaudeTestContext(), request)
	require.NoError(t, err)
	require.NotNil(t, claudeRequest.Thinking)
	require.Equal(t, "adaptive", claudeRequest.Thinking.Type)
	require.Equal(t, "claude-sonnet-4-6", claudeRequest.Model)
	require.Nil(t, claudeRequest.TopP)
}

func TestRequestOpenAI2ClaudeMessageOmitsTopPForThinkingSuffix(t *testing.T) {
	topP := 0.8
	request := dto.GeneralOpenAIRequest{
		Model: "claude-sonnet-4-6-thinking",
		TopP:  &topP,
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(newClaudeTestContext(), request)
	require.NoError(t, err)
	require.NotNil(t, claudeRequest.Thinking)
	require.Equal(t, "enabled", claudeRequest.Thinking.Type)
	require.Nil(t, claudeRequest.TopP)
}

func TestRequestOpenAI2ClaudeMessageOmitsTopPForReasoningEffort(t *testing.T) {
	topP := 0.8
	request := dto.GeneralOpenAIRequest{
		Model:           "claude-sonnet-4-6",
		TopP:            &topP,
		ReasoningEffort: "low",
		Messages:        []dto.Message{{Role: "user", Content: "hi"}},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(newClaudeTestContext(), request)
	require.NoError(t, err)
	require.NotNil(t, claudeRequest.Thinking)
	require.Equal(t, "enabled", claudeRequest.Thinking.Type)
	require.Nil(t, claudeRequest.TopP)
}

func TestRequestOpenAI2ClaudeMessageOmitsTopPForReasoningPayload(t *testing.T) {
	topP := 0.8
	reasoning, err := json.Marshal(map[string]any{"max_tokens": 2048})
	require.NoError(t, err)

	request := dto.GeneralOpenAIRequest{
		Model:     "claude-sonnet-4-6",
		TopP:      &topP,
		Reasoning: reasoning,
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(newClaudeTestContext(), request)
	require.NoError(t, err)
	require.NotNil(t, claudeRequest.Thinking)
	require.Equal(t, "enabled", claudeRequest.Thinking.Type)
	require.Equal(t, 2048, claudeRequest.Thinking.GetBudgetTokens())
	require.Nil(t, claudeRequest.TopP)
}
