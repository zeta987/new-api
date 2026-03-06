package common

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestNormalizeClaudeThinkingSamplingClearsTopPWhenThinkingPresent(t *testing.T) {
	topP := 0.8
	request := &dto.ClaudeRequest{
		TopP: &topP,
		Thinking: &dto.Thinking{
			Type: "adaptive",
		},
	}

	NormalizeClaudeThinkingSampling(request)

	if request.TopP != nil {
		t.Fatalf("expected TopP to be cleared when thinking is present")
	}
}

func TestNormalizeClaudeThinkingSamplingKeepsTopPWithoutThinking(t *testing.T) {
	topP := 0.8
	request := &dto.ClaudeRequest{
		TopP: &topP,
	}

	NormalizeClaudeThinkingSampling(request)

	if request.TopP == nil || *request.TopP != 0.8 {
		t.Fatalf("expected TopP to remain when thinking is absent, got: %#v", request.TopP)
	}
}

func TestRemoveClaudeTopPWhenThinkingJSONDeletesTopP(t *testing.T) {
	out, err := RemoveClaudeTopPWhenThinkingJSON([]byte(`{"model":"claude-opus-4-6","thinking":{"type":"adaptive"},"top_p":0.8}`))
	if err != nil {
		t.Fatalf("RemoveClaudeTopPWhenThinkingJSON returned error: %v", err)
	}

	assertJSONEqual(t, `{"model":"claude-opus-4-6","thinking":{"type":"adaptive"}}`, string(out))
}

func TestRemoveClaudeTopPWhenThinkingJSONKeepsTopPWithoutThinking(t *testing.T) {
	out, err := RemoveClaudeTopPWhenThinkingJSON([]byte(`{"model":"claude-opus-4-6","top_p":0.8}`))
	if err != nil {
		t.Fatalf("RemoveClaudeTopPWhenThinkingJSON returned error: %v", err)
	}

	assertJSONEqual(t, `{"model":"claude-opus-4-6","top_p":0.8}`, string(out))
}

func TestRemoveClaudeTopPWhenThinkingJSONDeletesTopPAfterOverride(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			ParamOverride: map[string]interface{}{
				"top_p": 0.8,
			},
		},
	}

	input := []byte(`{"model":"claude-opus-4-6","thinking":{"type":"adaptive"}}`)
	overridden, err := ApplyParamOverrideWithRelayInfo(input, info)
	if err != nil {
		t.Fatalf("ApplyParamOverrideWithRelayInfo returned error: %v", err)
	}

	out, err := RemoveClaudeTopPWhenThinkingJSON(overridden)
	if err != nil {
		t.Fatalf("RemoveClaudeTopPWhenThinkingJSON returned error: %v", err)
	}

	assertJSONEqual(t, `{"model":"claude-opus-4-6","thinking":{"type":"adaptive"}}`, string(out))
}
